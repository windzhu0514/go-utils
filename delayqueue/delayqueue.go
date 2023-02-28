// 基于rabbitmq的重试
package delayqueue

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	uuid "github.com/satori/go.uuid"
	"github.com/windzhu0514/go-utils/utils"
)

type CallBackHandler interface {
	DelayQueueHandler()
}

type Message struct {
	InitialDelay  time.Duration          `json:"initialDelay"`  // 首次发送延迟时间
	FixedDelay    time.Duration          `json:"fixedDelay"`    // 非首次发送延迟时间
	Times         int                    `json:"times"`         // 当前重试次数
	TotalTimes    int                    `json:"totalTimes"`    // 总重试次数
	CreateAt      time.Time              `json:"createAt"`      // 首次发送时间
	LastPublishAt time.Time              `json:"lastPublishAt"` // 上次发送时间
	Body          []byte                 `json:"body"`          // 消息载体
	Metadata      map[string]interface{} `json:"metadata"`      // 附加信息
	TraceID       string                 `json:"traceID"`       // 每次请求的TraceID
}

type KeyValue struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type DelayQueue struct {
	amqpUrl      string
	concurrent   int // 并发数量
	exchangeName string
	queueName    string
	handler      func(msg *Message) error // 返回nil，不再进行重试

	amqpConn        *amqp.Connection
	amqpChannel     *amqp.Channel
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	quitChan        chan struct{}
	backof          BackOffPolicy
}

const (
	reconnectDelay = 5 * time.Second
	reInitDelay    = 2 * time.Second
)

func New(amqpUrl string, exchangeName, queueName string, concurrent int, handler func(msg *Message) error) (*DelayQueue, error) {
	r := &DelayQueue{
		amqpUrl:      amqpUrl,
		exchangeName: exchangeName,
		queueName:    queueName,
		concurrent:   concurrent,
		handler:      handler,
	}

	if r.handler == nil {
		return nil, errors.New("handler is nil")
	}

	r.quitChan = make(chan struct{})

	if err := r.connect(); err != nil {
		return nil, err
	}

	go r.handleReconnect()

	return r, nil
}

func (r *DelayQueue) Shutdown() {
	r.quitChan <- struct{}{}
}

func (r *DelayQueue) Publish(msg *Message) error {
	return r.publish(msg, true)
}

func (r *DelayQueue) publish(msg *Message, first bool) error {
	msg.CreateAt = time.Now()
	msg.LastPublishAt = msg.CreateAt
	msg.TraceID = uuid.NewV4().String()

	headers := make(amqp.Table)
	if first {
		if msg.InitialDelay != 0 {
			headers["x-delay"] = msg.InitialDelay.Milliseconds()
		}
	} else {
		if msg.Delayed != 0 {
			headers["x-delay"] = msg.Delayed.Milliseconds()
		}
	}

	err := r.amqpChannel.Publish(
		r.exchangeName, // exchange
		"",             // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "application/json",
			Headers:      headers,
			Body:         utils.JsonMarshalByte(msg),
		})

	return err
}

func (r *DelayQueue) connect() (err error) {
	r.amqpConn, err = amqp.Dial(r.amqpUrl)
	if err != nil {
		return err
	}

	r.notifyConnClose = make(chan *amqp.Error)
	r.amqpConn.NotifyClose(r.notifyConnClose)

	if err = r.init(); err != nil {
		return err
	}

	return nil
}

func (r *DelayQueue) init() (err error) {
	if r.amqpConn == nil {
		return errors.New("r.amqpConn is nil")
	}

	r.amqpChannel, err = r.amqpConn.Channel()
	if err != nil {
		return err
	}

	r.notifyChanClose = make(chan *amqp.Error)
	r.amqpChannel.NotifyClose(r.notifyChanClose)

	err = r.amqpChannel.Qos(1, 0, false)
	if err != nil {
		return err
	}

	args := make(amqp.Table)
	args["x-delayed-type"] = "direct"
	err = r.amqpChannel.ExchangeDeclare(r.exchangeName, "x-delayed-message", true, false, false, false, args)
	if err != nil {
		return fmt.Errorf("ExchangeDeclare:%s err: %s", r.exchangeName, err.Error())
	}

	_, err = r.amqpChannel.QueueDeclare(r.queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("QueueDeclare:%s err: %s", r.queueName, err.Error())
	}

	err = r.amqpChannel.QueueBind(r.queueName, "", r.exchangeName, false, nil)
	if err != nil {
		return fmt.Errorf("QueueBind queueName:%s exchangeName:%s err: %s", r.queueName, r.exchangeName, err.Error())
	}

	chMsgs, err := r.amqpChannel.Consume(
		r.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	go r.consume(chMsgs)

	return nil
}

func (r *DelayQueue) handleReconnect() {
	for {
		select {
		case amqpErr := <-r.notifyConnClose:
			if err := r.connect(); err != nil {
				select {
				case <-r.quitChan:
					return
				case <-time.After(reconnectDelay):
				}
				continue
			}

		case amqpErr := <-r.notifyChanClose:
			if err := r.init(); err != nil {
				select {
				case <-r.quitChan:
					return
				case <-time.After(reInitDelay):
				}
				continue
			}

		case <-r.quitChan:
			r.amqpConn.Close()
			r.amqpChannel.Close()
			return
		}
	}
}

func (r *DelayQueue) consume(chMsgs <-chan amqp.Delivery) {
	if r.concurrent == 0 {
		r.concurrent = 10
	}

	limit := make(chan struct{}, r.concurrent)
	for d := range chMsgs {
		d := d
		limit <- struct{}{}
		go func() {
			defer func() {
				if err := recover(); err != nil {
					buf := make([]byte, 64<<10)
					n := runtime.Stack(buf, false)
					buf = buf[:n]

				}
			}()

			r.do(d)
			if err := d.Ack(false); err != nil {
			}

			<-limit
		}()
	}
}

func (r *DelayQueue) do(msg amqp.Delivery) {
	var retryMsg Message
	if err := json.Unmarshal(msg.Body, &retryMsg); err != nil {
		return
	}

	if err := r.handler(&retryMsg); err != nil {

		retryMsg.LastPublishAt = time.Now()
		retryMsg.Times++

		if retryMsg.TotalTimes > 0 && retryMsg.Times < retryMsg.TotalTimes {
			// 重新入队
			if err := r.publish(&retryMsg, false); err != nil {
			}
			return
		}

		return
	}
}
