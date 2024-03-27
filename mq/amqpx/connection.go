package amqpx

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	url    string
	logger *slog.Logger

	*amqp.Connection
	close           chan struct{}
	closed          int32
	chanNotifyClose chan *amqp.Error
	recoveryDelay   time.Duration

	mux      sync.Mutex
	channels []*Channel
}

func (c *Connection) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

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

	c.channels = append(c.channels, ch)

	return ch, nil
}

func (c *Connection) Close() error {
	// atomic.StoreInt32(&c.closed, 1)
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

// TODO:
// 重试次数 -1 无限重试 0 不重试 默认为-1
// 重试间隔策略 默认为指数退避策略 也可以自定义
func (c *Connection) handleReconnect() {
	for {
		select {
		case errClose := <-c.chanNotifyClose:
			// if c.conn.IsClosed() {
			// 	return
			// }
			c.logger.Error(fmt.Sprintf("connection closed: %v", errClose))

			for {
				conn, err := amqp.Dial(c.url)
				if err != nil {
					c.logger.Error("reconnect failed: " + err.Error())

					select {
					case <-c.close:
						return
					case <-time.After(5 * time.Second):
					}
					continue
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
