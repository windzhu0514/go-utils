// package delayqueue 实现了基于 rabbitmq 的延迟消息发送和处理
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

// DelayMessage 延迟消息体
type DelayMessage struct {
	Body        []byte                 `json:"body"`        // 消息载体
	TotalTimes  int                    `json:"totalTimes"`  // 总重试次数
	ContentType string                 `json:"contentType"` // body 的 MIME类型，可为空
	BackOffName string                 `json:"backOffName"` // 重试策略名称
	Metadata    map[string]interface{} `json:"metadata"`    // 附加信息
}

// Message 包内部使用的延迟消息体
type Message struct {
	*DelayMessage
	Times         int       `json:"times"`         // 当前重试次数
	CreateAt      time.Time `json:"createAt"`      // 首次发送时间
	LastPublishAt time.Time `json:"lastPublishAt"` // 上次发送时间
	TraceID       string    `json:"traceID"`       // 每次请求的TraceID
}

// DelayQueue 实现了延迟消息发送和处理
type DelayQueue struct {
	opt *Option

	logger       log.Logger
	amqpUrl      string
	exchangeName string
	queueName    string
	handler      func(msg Message) error // 返回nil，不再进行重试
	backoffs     map[string]backoff.Policy

	amqpConn        *amqp.Connection
	amqpChannel     *amqp.Channel
	notifyConnClose chan *amqp.Error // rabbitMQ 连接关闭通知
	notifyChanClose chan *amqp.Error // rabbitMQ channel 关闭通知
	quitChan        chan struct{}    // 退出信号
}

// Option 配置选项
type Option struct {
	Concurrent  int            // 并发数量 默认为1
	BackOff     backoff.Policy // Deprecated: 使用 RegisterBackOff 注册重试策略
	ConsumerTag string         // 消费者标识
}

const (
	reconnectDelay = 5 * time.Second // 重连间隔
	reInitDelay    = 2 * time.Second // 重新初始化间隔
)

// New 创建一个 DelayQueue 实例
// opt 默认为 1 个goroutine，消息不进行延迟
// handler 消息处理函数，不能为空
func New(logger log.Logger, amqpUrl string, exchangeName, queueName string, opt *Option, handler func(msg Message) error) (*DelayQueue, error) {
	r := &DelayQueue{
		logger:       log.With(logger, "module", "retry", "exchangeName", exchangeName, "queueName", queueName),
		amqpUrl:      amqpUrl,
		exchangeName: exchangeName,
		queueName:    queueName,
		opt:          opt,
		handler:      handler,
		backoffs:     make(map[string]backoff.Policy),
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

// Shutdown 退出自动重状态
func (r *DelayQueue) Shutdown() {
	r.quitChan <- struct{}{}
}

func (r *DelayQueue) RegisterBackoff(name string, backoff backoff.Policy) {
	if name == "" {
		panic("name is empty")
	}

	if backoff == nil {
		panic("backOff is nil")
	}

	if _, ok := r.backoffs[name]; ok {
		panic("backOff already registered: " + name)
	}

	r.backoffs[name] = backoff
}

// Publish 发布一个延迟消息
// TotalTimes 等于 0 时，会一直进行重试，直到处理成功
func (r *DelayQueue) Publish(delayMsg *DelayMessage) error {
	msg := &Message{DelayMessage: delayMsg}
	msg.Times = 1
	msg.CreateAt = time.Now()
	msg.LastPublishAt = msg.CreateAt
	msg.TraceID = uuid.NewV4().String()
	if msg.TotalTimes < 0 {
		msg.TotalTimes = 0
	}

	return r.publish(msg)
}

func (r *DelayQueue) publish(msg *Message) error {
	lh := log.NewHelper(log.With(r.logger, "jsonContent", utils.JsonMarshalString(msg)))
	lh.Debug("publish msg")
	defer lh.Debug("publish msg end")

	headers := make(amqp.Table)
	backOff := r.backoffs[msg.BackOffName]
	if backOff == nil {
		backOff = r.opt.BackOff
	}
	delay := backOff.BackOff(msg.Times).Milliseconds()
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

	err = r.amqpChannel.Qos(r.opt.Concurrent, 0, false)
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
	lh := log.NewHelper(r.logger)
	for {
		select {
		case amqpErr := <-r.notifyConnClose:
			lh.Errorf("rabbitMQ connection notify: %v", amqpErr)
			if err := r.connect(); err != nil {
				select {
				case <-r.quitChan:
					lh.Info("rabbitMQ has been shut down")
					return
				case <-time.After(reconnectDelay):
				}
				continue
			}

		case amqpErr := <-r.notifyChanClose:
			lh.Errorf("rabbitMQ channel notify: %v", amqpErr)
			if err := r.init(); err != nil {
				select {
				case <-r.quitChan:
					lh.Info("rabbitMQ has been shut down")
					return
				case <-time.After(reInitDelay):
				}
				continue
			}

		case <-r.quitChan:
			r.amqpConn.Close()
			r.amqpChannel.Close()
			lh.Info("rabbitMQ has been shut down")
			return
		}
	}
}

func (r *DelayQueue) consume(chMsgs <-chan amqp.Delivery) {
	lh := log.NewHelper(r.logger)
	lh.Debug("begin consume mq messages")

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

					lh.Errorf("mqConsume panic: %v\n%s", err, buf)
				}
			}()

			r.do(d)
			if err := d.Ack(false); err != nil {
				lh.Errorf("consume Ack error: %s", err.Error())
			}

			<-limit
		}()
	}
}

func (r *DelayQueue) do(msg amqp.Delivery) {
	lh := log.NewHelper(r.logger)

	var retryMsg Message
	if err := json.Unmarshal(msg.Body, &retryMsg); err != nil {
		lh.Errorw("jsonContent", string(msg.Body), log.DefaultMessageKey, "unmarshal msg: "+err.Error())
		return
	}

	lh = log.NewHelper(log.With(r.logger, "traceId", retryMsg.TraceID, "times", retryMsg.Times, "totalTimes", retryMsg.TotalTimes))
	lh.Debugw("jsonContent", string(msg.Body), log.DefaultMessageKey, "处理重试消息")

	if err := r.handler(retryMsg); err != nil {
		lh.Debug("重试消息处理失败: " + err.Error())

		if retryMsg.TotalTimes == 0 || retryMsg.Times < retryMsg.TotalTimes {
			// 重新入队
			retryMsg.LastPublishAt = time.Now()
			retryMsg.Times++
			if err := r.publish(&retryMsg); err != nil {
				lh.Error("publish: " + err.Error())
			}
			return
		}

		lh.Debug("达到最大重试次数，结束重试")

		return
	}

	lh.Debug("重试处理成功，结束重试")
}
