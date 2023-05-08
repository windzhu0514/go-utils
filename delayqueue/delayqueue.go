// 基于rabbitmq的重试
package delayqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	amqp "github.com/rabbitmq/amqp091-go"
	uuid "github.com/satori/go.uuid"
	"github.com/windzhu0514/go-utils/delayqueue/backoff"
	"github.com/windzhu0514/go-utils/utils"
)

type DelayMessage struct {
	Body        []byte                 `json:"body"`        // 消息载体
	TotalTimes  int                    `json:"totalTimes"`  // 总重试次数
	ContentType string                 `json:"contentType"` // 可为空
	Metadata    map[string]interface{} `json:"metadata"`    // 附加信息
}

type Message struct {
	*DelayMessage
	Times         int       `json:"times"`         // 当前重试次数
	CreateAt      time.Time `json:"createAt"`      // 首次发送时间
	LastPublishAt time.Time `json:"lastPublishAt"` // 上次发送时间
	TraceID       string    `json:"traceID"`       // 每次请求的TraceID
}

type DelayQueue struct {
	log          *log.Helper
	amqpUrl      string
	opt          *Option
	exchangeName string
	queueName    string
	handler      func(msg *Message) error // 返回nil，不再进行重试

	amqpConn        *amqp.Connection
	amqpChannel     *amqp.Channel
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	quitChan        chan struct{}
}

type Option struct {
	Concurrent  int            // 并发数量 默认为1
	BackOff     backoff.Policy // 默认 noPolicy
	ConsumerTag string         // 消费者标识
}

const (
	reconnectDelay = 5 * time.Second
	reInitDelay    = 2 * time.Second
)

func New(logger log.Logger, amqpUrl string, exchangeName, queueName string, opt *Option, handler func(msg *Message) error) (*DelayQueue, error) {
	r := &DelayQueue{
		log:          log.NewHelper(log.With(logger, "module", "retry")),
		amqpUrl:      amqpUrl,
		exchangeName: exchangeName,
		queueName:    queueName,
		opt:          opt,
		handler:      handler,
	}

	if r.handler == nil {
		return nil, errors.New("handler is nil")
	}

	if r.opt == nil {
		r.opt = &Option{Concurrent: 1, BackOff: backoff.NewNoPolicy()}
	} else {
		if r.opt.Concurrent < 1 {
			r.opt.Concurrent = 1
		}

		if r.opt.BackOff == nil {
			r.opt.BackOff = backoff.NewNoPolicy()
		}
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

func (r *DelayQueue) Publish(delayMsg *DelayMessage) error {
	msg := &Message{DelayMessage: delayMsg}
	msg.Times = 1
	msg.CreateAt = time.Now()
	msg.LastPublishAt = msg.CreateAt
	msg.TraceID = uuid.NewV4().String()

	return r.publish(msg)
}

func (r *DelayQueue) publish(msg *Message) error {
	r.log.Debugw(log.DefaultMessageKey, "publish msg", "jsonContent", utils.JsonMarshalString(msg))
	defer r.log.Debugw(log.DefaultMessageKey, "publish msg end", "jsonContent", utils.JsonMarshalString(msg))

	headers := make(amqp.Table)
	delay := r.opt.BackOff.BackOff(msg.Times).Milliseconds()
	if delay != 0 {
		headers["x-delay"] = delay
	}

	err := r.amqpChannel.PublishWithContext(context.Background(),
		r.exchangeName, // exchange
		"",             // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  msg.ContentType,
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

	r.log.Debug("declare exchange and queue")

	args := make(amqp.Table)
	args["x-delayed-type"] = "direct"
	err = r.amqpChannel.ExchangeDeclare(r.exchangeName, "x-delayed-message", true, false, false, false, args)
	if err != nil {
		return fmt.Errorf("ExchangeDeclare:%s err: %s", r.exchangeName, err.Error())
	}
	r.log.Debug("declare exchange success")

	_, err = r.amqpChannel.QueueDeclare(r.queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("QueueDeclare:%s err: %s", r.queueName, err.Error())
	}

	r.log.Debug("declare queue success")

	err = r.amqpChannel.QueueBind(r.queueName, "", r.exchangeName, false, nil)
	if err != nil {
		return fmt.Errorf("QueueBind queueName:%s exchangeName:%s err: %s", r.queueName, r.exchangeName, err.Error())
	}

	chMsgs, err := r.amqpChannel.Consume(
		r.queueName,       // queue
		r.opt.ConsumerTag, // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
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
	r.log.Debug("begin consume mq messages")

	limit := make(chan struct{}, r.opt.Concurrent)
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

		if retryMsg.TotalTimes > 0 && retryMsg.Times < retryMsg.TotalTimes {
			// 重新入队
			retryMsg.LastPublishAt = time.Now()
			retryMsg.Times++
			if err := r.publish(&retryMsg); err != nil {
				r.log.Error("publish: " + err.Error())
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
