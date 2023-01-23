// 基于rabbitmq的重试
package delayqueue

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	amqp "github.com/rabbitmq/amqp091-go"
	uuid "github.com/satori/go.uuid"
	"github.com/windzhu0514/go-utils/utils"
)

type CallBackHandler interface {
	DelayQueueHandler()
}

// DelayMsg
// Host 消息发送地址
// Path 消息发送地址utl path
// Heads 请求头
// Body 请求体
// Content-Type 默认使用 head 里 Content-Type 的值，如果为空，按以下规则设置
// Struct Map Slice Content-Type 自动使用 "application/json"
// String Content-Type 自动使用 "text/plain; charset=utf-8"
// []byte Content-Type 使用 http.DetectContentType 来探测 Content-Type 的值
// FirstDelayed 首次发送延迟时间
// Delayed 每次发送的延迟时间
// Times 当前发送次数
// TotalTimes 总共发送次数，默认和 Times 相同，0 表示未收到正确的回复时，一直重试
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
	log          *log.Helper
	concurrent   int // 并发数量
	exchangeName string
	queueName    string
	handler      func(msg *Message) error // 返回nil，不再进行重试

	amqpConn        *amqp.Connection
	amqpChannel     *amqp.Channel
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	quitChan        chan struct{}
}

const (
	reconnectDelay = 5 * time.Second
	reInitDelay    = 2 * time.Second
)

func New(amqpUrl string, logger log.Logger, exchangeName, queueName string, concurrent int, handler func(msg *Message) error) (*DelayQueue, error) {
	r := &DelayQueue{
		amqpUrl:      amqpUrl,
		log:          log.NewHelper(log.With(logger, "module", "retry")),
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
		r.log.Errorf("connect amqp error: %s", err.Error())
		return nil, err
	}

	go r.handleReconnect()

	r.log.Info("retry new success")

	return r, nil
}

func (r *DelayQueue) Shutdown() {
	r.quitChan <- struct{}{}

	r.log.Info("shutting down rabbitMQ's connection...")
}

func (r *DelayQueue) Publish(msg *Message) error {
	return r.publish(msg, true)
}

func (r *DelayQueue) publish(msg *Message, first bool) error {
	r.log.Debugw(log.DefaultMessageKey, "publish msg", "jsonContent", utils.JsonMarshalString(msg))
	defer r.log.Debugw(log.DefaultMessageKey, "publish msg end", "jsonContent", utils.JsonMarshalString(msg))

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
		r.log.Error(err)
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
		r.log.Error(err)
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
		r.log.Error(err)
		return err
	}

	go r.consume(chMsgs)

	return nil
}

func (r *DelayQueue) handleReconnect() {
	for {
		select {
		case amqpErr := <-r.notifyConnClose:
			r.log.Errorf("rabbitMQ connection notify: %v", amqpErr)
			if err := r.connect(); err != nil {
				select {
				case <-r.quitChan:
					r.log.Info("rabbitMQ has been shut down")
					return
				case <-time.After(reconnectDelay):
				}
				continue
			}

		case amqpErr := <-r.notifyChanClose:
			r.log.Errorf("rabbitMQ channel notify: %v", amqpErr)
			if err := r.init(); err != nil {
				select {
				case <-r.quitChan:
					r.log.Info("rabbitMQ has been shut down")
					return
				case <-time.After(reInitDelay):
				}
				continue
			}

		case <-r.quitChan:
			r.amqpConn.Close()
			r.amqpChannel.Close()
			r.log.Info("rabbitMQ has been shut down")
			return
		}
	}
}

func (r *DelayQueue) consume(chMsgs <-chan amqp.Delivery) {
	r.log.Infof("begin consume mq messages")
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

					r.log.Errorf("mqConsume panic: %v\n%s", err, buf)
				}
			}()

			r.log.Debugf("Received a message: %s", string(d.Body))
			r.do(d)
			if err := d.Ack(false); err != nil {
				r.log.Errorf("consume Ack error: %s", err.Error())
			}

			<-limit
		}()
	}
}

func (r *DelayQueue) do(msg amqp.Delivery) {
	var retryMsg Message
	if err := json.Unmarshal(msg.Body, &retryMsg); err != nil {
		r.log.Errorw("error", err.Error(), "jsonContent", string(msg.Body))
		return
	}

	if err := r.handler(&retryMsg); err != nil {
		r.log.Debugw("traceId", retryMsg.TraceID, "retryTimes", retryMsg.Times, "retryTotalTimes", retryMsg.TotalTimes,
			"retryMsg", utils.JsonMarshalString(retryMsg), log.DefaultMessageKey, "重试消息处理失败")

		retryMsg.LastPublishAt = time.Now()
		retryMsg.Times++

		if retryMsg.TotalTimes > 0 && retryMsg.Times < retryMsg.TotalTimes {
			// 重新入队
			if err := r.publish(&retryMsg, false); err != nil {
				r.log.Error(err)
			}
			return
		}

		r.log.Debugw("traceId", retryMsg.TraceID, "retryTimes", retryMsg.Times, "retryTotalTimes", retryMsg.TotalTimes,
			"retryMsg", utils.JsonMarshalString(retryMsg), log.DefaultMessageKey, "总重试次数为0或达到最大重试次数，结束重试")

		return
	}

	r.log.Debugw("traceId", retryMsg.TraceID, "retryTimes", retryMsg.Times, "retryTotalTimes", retryMsg.TotalTimes,
		"retryMsg", utils.JsonMarshalString(retryMsg), log.DefaultMessageKey, "重试处理成功，结束重试")
}
