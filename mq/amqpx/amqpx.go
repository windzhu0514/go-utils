package amqpx

import (
	"context"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	url    string
	logger *slog.Logger

	conn            *amqp.Connection
	close           chan struct{}
	chanNotifyClose chan *amqp.Error
	recoveryDelay   time.Duration

	mux      sync.Mutex
	channels []*Channel
}

func New(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		url:    url,
		logger: slog.Default(),
		conn:   conn,
		close:  make(chan struct{}),
	}

	c.chanNotifyClose = make(chan *amqp.Error)
	c.conn.NotifyClose(c.chanNotifyClose)

	go c.handleReconnect()

	return c, nil
}

func (c *Connection) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

func (c *Connection) Close() error {
	close(c.close)
	if err := c.conn.Close(); err != nil {
		return err
	}

	for _, channel := range c.channels {
		channel.Close()
	}

	return nil
}

func (c *Connection) IsClosed() bool {
	return c.conn.IsClosed()
}

func (c *Connection) handleReconnect() {
	for {
		select {
		case <-c.chanNotifyClose:
			if c.conn.IsClosed() {
				return
			}

			if err := func() error {
				conn, err := amqp.Dial(c.url)
				if err != nil {
					return err
				}

				c.conn = conn
				c.chanNotifyClose = make(chan *amqp.Error)
				c.conn.NotifyClose(c.chanNotifyClose)

				for _, ch := range c.channels {
					ch.Close()

					ch.channel, err = conn.Channel()
					if err != nil {
						return err
					}

					ch.recover()
				}

				return nil
			}(); err != nil {
				time.Sleep(5 * time.Second)
			}

		case <-c.close:
			return
		}
	}
}

type Channel struct {
	conn             *Connection
	channel          *amqp.Channel
	close            chan struct{}
	chanNotifyClose  chan *amqp.Error
	chanNotifyCancel chan string
	consumers        map[string]chan struct{}

	recordedExchanges      map[string]*RecordedExchange
	recordedQueues         map[string]*RecordedQueue
	recordedBindings       []*RecordedBinding
	recordedConsumers      map[string]*RecordedConsumer
	recordedPrefetchCount  int
	recordedPrefetchSize   int
	recordedPrefetchGlobal bool
}

func (c *Connection) Channel() (*Channel, error) {
	channel, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}

	ch := &Channel{
		conn:             c,
		channel:          channel,
		close:            make(chan struct{}),
		chanNotifyClose:  make(chan *amqp.Error),
		chanNotifyCancel: make(chan string),
		consumers:        make(map[string]chan struct{}),
	}

	go ch.watch()

	channel.NotifyClose(ch.chanNotifyClose)
	channel.NotifyCancel(ch.chanNotifyCancel)

	ch.recordedExchanges = make(map[string]*RecordedExchange)
	ch.recordedQueues = make(map[string]*RecordedQueue)
	ch.recordedConsumers = make(map[string]*RecordedConsumer)

	c.channels = append(c.channels, ch)

	return ch, nil
}

func (c *Channel) watch() {
	for {
		select {
		case <-c.chanNotifyClose:
		case <-c.chanNotifyCancel:

		case <-c.close:
			return
		}
	}
}

func (c *Channel) Close() error {
	if !c.channel.IsClosed() {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}

	close(c.close)

	return nil
}

func (c *Channel) Qos(prefetchCount, prefetchSize int, global bool) error {
	if err := c.channel.Qos(prefetchCount, prefetchSize, global); err != nil {
		return err
	}

	c.recordedPrefetchCount = prefetchCount
	c.recordedPrefetchSize = prefetchSize
	c.recordedPrefetchGlobal = global

	return nil
}

func (c *Channel) ExchangeDeclare(name string, kind string, durable bool, autoDelete bool, internal bool, args amqp.Table) error {
	err := c.channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, false, args)
	c.recordExchange(name, &RecordedExchange{Kind: kind, Durable: durable, AutoDelete: autoDelete, Args: args})
	return err
}

func (c *Channel) ExchangeDeclareNoWait(name string, kind string, durable bool, autoDelete bool, internal bool, args amqp.Table) error {
	err := c.channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, true, args)
	c.recordExchange(name, &RecordedExchange{Kind: kind, Durable: durable, AutoDelete: autoDelete, Args: args})
	return err
}

func (c *Channel) QueueDeclare(name string, durable bool, autoDelete bool, exclusive bool, args amqp.Table) (amqp.Queue, error) {
	queue, err := c.channel.QueueDeclare(name, durable, autoDelete, exclusive, false, args)
	c.recordedQueue(queue.Name, &RecordedQueue{Durable: durable, AutoDelete: autoDelete, Exclusive: exclusive, Args: args})
	return queue, err
}

func (c *Channel) QueueDeclareNoWait(name string, durable bool, autoDelete bool, exclusive bool, args amqp.Table) (amqp.Queue, error) {
	queue, err := c.channel.QueueDeclare(name, durable, autoDelete, exclusive, true, args)
	c.recordedQueue(queue.Name, &RecordedQueue{Durable: durable, AutoDelete: autoDelete, Exclusive: exclusive, Args: args})
	return queue, err
}

func (c *Channel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	err := c.channel.QueueBind(name, key, exchange, noWait, args)
	c.recordQueueBinding(&RecordedBinding{QueueName: name, ExchangeName: exchange, RoutingKey: key, Args: args})
	return err
}

func (c *Channel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return c.channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

func (c *Channel) Consume(queue, consumerTag string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table, consumer Consumer) error {
	if consumerTag == "" {
		consumerTag = uniqueConsumerTag()
	}

	delivery, err := c.channel.Consume(queue, consumerTag, autoAck, exclusive, noLocal, noWait, args)
	if err != nil {
		return err
	}

	closeConsumer := make(chan struct{})
	c.consumers[consumerTag] = closeConsumer

	go func() {
		for {
			select {
			case d, ok := <-delivery:
				if !ok {
					// consumer cancelled
					return
				}

				consumer.HandleDelivery(d)

			case errClose := <-c.chanNotifyClose:
				_ = errClose

				return
			case <-closeConsumer:
				return
			case <-c.close:
				return
			}
		}
	}()

	c.recordConsumer(&RecordedConsumer{QueueName: queue, ConsumerTag: consumerTag, AutoAck: autoAck, Exclusive: exclusive, NoWait: noWait, Args: args, Consumer: consumer})

	return nil
}

func (c *Channel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return c.channel.Publish(exchange, key, mandatory, immediate, msg)
}

func (c *Channel) CancelConsumer(consumerTag string, noWait bool) error {
	c.deleteRecordedConsumer(consumerTag)

	return c.channel.Cancel(consumerTag, noWait)
}
