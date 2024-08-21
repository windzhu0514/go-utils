package amqpx

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	*amqp.Connection

	url                 string
	logger              *slog.Logger
	recoveryMaxAttempts int // 0 一直重试
	recoveryBackoff     RecoveryBackoff

	close           chan struct{}
	chanNotifyClose chan *amqp.Error
	mux             sync.Mutex
	channels        []*Channel
}

// Channel 创建一个新的Channel
// 一个Connection可以创建多个Channel
// Channel是一个轻量级的Connection，可以用来发送和接收消息
// 不建议在多个goroutine中共享一个Channel
// https://www.rabbitmq.com/docs/channels#basics
func (c *Connection) Channel() (*Channel, error) {
	channel, err := c.Connection.Channel()
	if err != nil {
		return nil, err
	}

	ch := &Channel{
		logger:           c.logger.With("module", "channel"),
		conn:             c,
		Channel:          channel,
		close:            make(chan struct{}),
		chanNotifyClose:  make(chan *amqp.Error),
		chanNotifyCancel: make(chan string),
		// consumers:        make(map[string]chan struct{}),
	}

	// go ch.watch()

	// channel.NotifyClose(ch.chanNotifyClose)
	// channel.NotifyCancel(ch.chanNotifyCancel)

	ch.recordedExchanges = make(map[string]*RecordedExchange)
	ch.recordedQueues = make(map[string]*RecordedQueue)
	ch.recordedConsumers = make(map[string]*RecordedConsumer)

	c.mux.Lock()
	c.channels = append(c.channels, ch)
	c.mux.Unlock()

	return ch, nil
}

func (c *Connection) Close() error {
	close(c.close)
	if err := c.Connection.Close(); err != nil {
		return err
	}

	for _, channel := range c.channels {
		channel.Close()
	}

	return nil
}

func (c *Connection) IsClosed() bool {
	return c.Connection.IsClosed()
}

// 重试次数 -1 无限重试 0 不重试 默认为-1
// 重试间隔策略 默认为指数退避策略 也可以自定义
func (c *Connection) handleReconnect() {
	for {
		select {
		case errClose := <-c.chanNotifyClose:
			c.logger.Error(fmt.Sprintf("connection closed: %v", errClose))

			if errClose != nil && !errClose.Recover {
				select {
				case <-time.After(5 * time.Second): // ReconnectWait
				case <-c.close:
					return
				}
			}

			attempt := 0
			for {
				conn, err := amqp.Dial(c.url)
				if err != nil {
					c.logger.Error("reconnect failed: " + err.Error())

					if c.recoveryMaxAttempts > 0 && attempt >= c.recoveryMaxAttempts {
						c.logger.Error("recovery attempts exceed the limit")
						return
					}

					select {
					case <-c.close:
						return
					case <-time.After(c.recoveryBackoff.Delay(attempt)):
						attempt++
						continue
					}
				}

				c.Connection = conn
				c.chanNotifyClose = make(chan *amqp.Error)
				c.Connection.NotifyClose(c.chanNotifyClose)

				for _, ch := range c.channels {
					// ch.Close()

					ch.Channel, err = conn.Channel()
					if err != nil {
						c.logger.Error("recreate channel failed: " + err.Error())
						return
					}

					ch.recover()
				}

				break
			}

		case <-c.close:
			return
		}
	}
}
