package rabbitx

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/streadway/amqp"
)

type Connection struct {
	conn *amqp091.Connection
}
type Queue struct {
	channel *amqp.Channel
}

type Message struct{}

// 延迟消息
// 自动重试
// 达到最大次数回调
type DelayedMessage struct{}

type (
	ConnOption  func(*Connection)
	QueueOption func(*Queue)
)

// 初始化
// 建立连接
// 自动重连连接
func NewConnect(opts ...ConnOption) (*Connection, error) {
	conn := &Connection{}
	return conn, nil
}

// 创建queue
// 自动重连
func NewQueue(opts ...QueueOption) (*Queue, error) {
	q := &Queue{}
	return q, nil
}

func (q *Queue) Publish(msg *Message) error {
}
