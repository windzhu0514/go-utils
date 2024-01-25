// 基于rabbitmq的重试

package retry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/go-kratos/kratos/v2/log"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/windzhu0514/go-utils/utils"
)

const (
	delayedExchangeName = "qtrain.delayed.retry"
	delayedQueueName    = "qtrain.delayed.retry" // 延迟队列
)

type KeyValue struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

// CallbackMsg
// URL 消息发送地址
// Heads 请求头
// PostForm
// post请求表单值 Body字段不为空才使用PostForm
// Content-Type 默认使用 head 里 Content-Type 的值，如果为空，默认值为：application/x-www-form-urlencoded
// Body
// Content-Type 默认使用 head 里 Content-Type 的值，如果为空，按以下规则设置
// Struct Map Slice Content-Type 自动使用 "application/json"
// String Content-Type 自动使用 "text/plain; charset=utf-8"
// []byte Content-Type 使用 http.DetectContentType 来探测 Content-Type 的值
// FirstDelayed 首次发送延迟时间
// Delayed 每次发送的延迟时间
// Times 当前发送次数
// TotalTimes 总共发送次数，默认和 Times 相同，0 表示未收到正确的回复时，一直重试
type CallbackMsg struct {
	Host         string        `json:"host"`
	Path         string        `json:"path"`
	Heads        http.Header   `json:"heads"`
	PostForm     url.Values    `json:"params"`
	Body         []byte        `json:"body"`
	FirstDelayed time.Duration `json:"firstDelayed"` // 首次发送延迟时间
	Delayed      time.Duration `json:"delayed"`      // 延迟时间
	TotalTimes   int           `json:"totalTimes"`   // 重试次数
	OrderID      string        `json:"orderId"`
	Method       string        `json:"method"` // 发送到同程的请求method
	Times        int           `json:"times"`  // 重试次数
	TraceId      string        `json:"traceId"`
}

type Retry struct {
	amqpUrl string
	checker func(msg CallbackMsg, v []byte) bool
	debug   bool

	log                 *log.Helper
	client              *resty.Client
	amqpConn            *amqp.Connection
	amqpChannel         *amqp.Channel
	closeChan           chan *amqp.Error
	quitChan            chan struct{}
	tag                 string
	delayedExchangeName string
	delayedQueueName    string
}

func New(client *resty.Client, amqpUrl string, logger log.Logger, tag string, checker func(msg CallbackMsg, v []byte) bool, debug bool) (*Retry, error) {
	r := &Retry{
		amqpUrl: amqpUrl,
		log:     log.NewHelper(logger, log.WithMessageKey("message")),
		client:  client,
		checker: checker,
		tag:     tag,
		debug:   debug,
	}

	if r.checker == nil {
		return nil, errors.New("checker is nil")
	}

	r.quitChan = make(chan struct{})

	if r.tag == "" {
		r.log.Errorf("tag is empty")
		return nil, errors.New("tag is empty")
	}

	r.delayedExchangeName = delayedExchangeName + "." + r.tag
	if r.debug {
		r.delayedExchangeName += ".t"
	}
	r.delayedQueueName = delayedQueueName + "." + r.tag
	if r.debug {
		r.delayedQueueName += ".t"
	}

	if err := r.connect(); err != nil {
		r.log.Errorf("connect amqp error: %s", err.Error())
		return nil, err
	}

	return r, nil
}

func (r *Retry) Shutdown() {
	r.quitChan <- struct{}{}

	r.log.Info("shutting down rabbitMQ's connection...")

	<-r.quitChan
}

func (r *Retry) Send(msg CallbackMsg) error {
	return r.send(msg, true)
}

func (r *Retry) send(msg CallbackMsg, first bool) error {
	msg.TraceId = uuid.NewV4().String()

	headers := make(amqp.Table)
	if first {
		if msg.FirstDelayed != 0 {
			headers["x-delay"] = msg.FirstDelayed.Milliseconds()
		}
	} else {
		if msg.Delayed != 0 {
			headers["x-delay"] = msg.Delayed.Milliseconds()
		}
	}

	body, _ := json.Marshal(msg)
	err := r.amqpChannel.Publish(
		r.delayedExchangeName, // exchange
		"",                    // routing key
		false,                 // mandatory
		false,                 // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "application/json",
			Headers:      headers,
			Body:         body,
		})

	return err
}

func (r *Retry) connect() (err error) {
	r.amqpConn, err = amqp.Dial(r.amqpUrl)
	if err != nil {
		r.log.Error(err)
		return err
	}

	r.amqpChannel, err = r.amqpConn.Channel()
	if err != nil {
		r.log.Error(err)
		return err
	}

	r.closeChan = make(chan *amqp.Error)
	r.amqpConn.NotifyClose(r.closeChan)
	// r.amqpChannel.NotifyClose()

	err = r.amqpChannel.Qos(100, 0, false)
	if err != nil {
		return err
	}

	args := make(amqp.Table)
	args["x-delayed-type"] = "direct"

	err = r.amqpChannel.ExchangeDeclare(r.delayedExchangeName, "x-delayed-message", true, false, false, false, args)
	if err != nil {
		return fmt.Errorf("ExchangeDeclare:%s error: %s", r.delayedExchangeName, err.Error())
	}

	_, err = r.amqpChannel.QueueDeclare(
		r.delayedQueueName, // name
		true,               // 开启消息持久化
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)

	err = r.amqpChannel.QueueBind(r.delayedQueueName, "", r.delayedExchangeName, false, nil)
	if err != nil {
		return
	}

	chMsgs, err := r.amqpChannel.Consume(
		r.delayedQueueName, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		r.log.Error(err)
		return err
	}

	go r.consume(chMsgs)
	go r.handleDisconnect()

	return nil
}

func (r *Retry) handleDisconnect() {
	for {
		select {
		case errChann := <-r.closeChan:
			if errChann != nil {
				r.log.Errorf("rabbitMQ disconnection: %v", errChann)
			}
		case <-r.quitChan:
			r.amqpConn.Close()
			r.log.Info("rabbitMQ has been shut down")
			r.quitChan <- struct{}{}
			return
		}

		r.log.Info("trying to reconnect to rabbitMQ")

		time.Sleep(5 * time.Second)
		r.amqpChannel.Close()
		if err := r.connect(); err != nil {
			r.log.Errorf("rabbitMQ error: %v", err)
		}
	}
}

func (r *Retry) consume(chMsgs <-chan amqp.Delivery) {
	r.log.Debug("beging waiting for mq messages")
	for d := range chMsgs {
		d := d
		// TODO：限制协程数量
		go func() {
			defer func() {
				if err := recover(); err != nil {
					buf := make([]byte, 64<<10) //nolint:gomnd
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
		}()
	}
}

func (r *Retry) do(msg amqp.Delivery) {
	var callbackMsg CallbackMsg
	if err := json.Unmarshal(msg.Body, &callbackMsg); err != nil {
		r.log.Error(err)
		return
	}

	req := r.client.R()

	req.Header = callbackMsg.Heads
	if callbackMsg.Body != nil {
		req.SetBody(callbackMsg.Body)
	}

	r.log.Infow("orderId", callbackMsg.OrderID, "traceId", callbackMsg.TraceId,
		"request", string(callbackMsg.Body), "mqMsg", utils.JsonMarshalString(callbackMsg),
		"msg", "发送重试消息")

	URL := callbackMsg.Host
	if callbackMsg.Path != "" {
		URL = path.Join(([]string{URL, callbackMsg.Path})...)
	}

	resp, err := req.Post(URL)
	if err != nil {
		r.log.Errorw("orderId", callbackMsg.OrderID, "traceId", callbackMsg.TraceId,
			"request", utils.JsonMarshalString(callbackMsg),
			"msg", "发送重试消息失败:"+err.Error())
		return
	}

	r.log.Infow("orderId", callbackMsg.OrderID, "traceId", callbackMsg.TraceId,
		"request", string(callbackMsg.Body), "mqMsg", utils.JsonMarshalString(callbackMsg),
		"response", resp.String(), "msg", "收到重试消息结果")

	if r.checker(callbackMsg, resp.Body()) {
		return
	}

	callbackMsg.Times++
	if callbackMsg.TotalTimes > 0 && callbackMsg.Times < callbackMsg.TotalTimes {
		// 重新入队
		if err := r.send(callbackMsg, false); err != nil {
			r.log.Error(err)
		}
	} else {
		r.log.Infow("orderId", callbackMsg.OrderID, "traceId", callbackMsg.TraceId,
			"request", utils.JsonMarshalString(callbackMsg), "mqMsg", utils.JsonMarshalString(callbackMsg),
			"response", resp.String(),
			"msg", fmt.Sprintf("结束重试 times:%d totalTimes:%d", callbackMsg.Times, callbackMsg.TotalTimes))
	}
}

func (r *Retry) decodeBody(msg *CallbackMsg) string {
	dest := make([]byte, base64.StdEncoding.DecodedLen(len(msg.Body)))
	n, err := base64.StdEncoding.Decode(dest, msg.Body)
	if err != nil {
		r.log.Errorw("orderId", msg.OrderID, "msg", err.Error())
		return string(msg.Body)
	}

	dest = dest[:n]

	fields := bytes.Split(dest, []byte("="))
	switch len(fields) {
	case 1:
		s, err := url.QueryUnescape(string(fields[0]))
		if err != nil {
			r.log.Errorw("orderId", msg.OrderID, "msg", "QueryUnescape:"+err.Error())
			return string(msg.Body)
		}

		return s
	case 2:
		s, err := url.QueryUnescape(string(fields[1]))
		if err != nil {
			r.log.Errorw("orderId", msg.OrderID, "msg", "QueryUnescape:"+err.Error())
			return string(msg.Body)
		}

		return s

	default:
		return string(msg.Body)
	}
}
