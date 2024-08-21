package amqpx

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Channel struct {
	logger *slog.Logger
	conn   *Connection
	*amqp.Channel
	close            chan struct{}
	chanNotifyClose  chan *amqp.Error
	chanNotifyCancel chan string
	// consumers        map[string]chan struct{}

	recordedExchanges      map[string]*RecordedExchange
	recordedQueues         map[string]*RecordedQueue
	recordedBindings       []*RecordedBinding
	recordedConsumers      map[string]*RecordedConsumer
	recordedPrefetchCount  int
	recordedPrefetchSize   int
	recordedPrefetchGlobal bool
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
	if !c.Channel.IsClosed() {
		if err := c.Channel.Close(); err != nil {
			return err
		}
	}

	close(c.close)

	return nil
}

func (c *Channel) Qos(prefetchCount, prefetchSize int, global bool) error {
	if err := c.Channel.Qos(prefetchCount, prefetchSize, global); err != nil {
		return err
	}

	c.recordedPrefetchCount = prefetchCount
	c.recordedPrefetchSize = prefetchSize
	c.recordedPrefetchGlobal = global

	return nil
}

func (c *Channel) ExchangeDeclare(name string, kind string, durable bool, autoDelete bool, internal bool, args amqp.Table) error {
	err := c.Channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, false, args)
	if err != nil {
		return err
	}

	c.recordExchange(name, &RecordedExchange{Kind: kind, Durable: durable, AutoDelete: autoDelete, Args: args})
	return nil
}

func (c *Channel) ExchangeDeclareNoWait(name string, kind string, durable bool, autoDelete bool, internal bool, args amqp.Table) error {
	err := c.Channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, true, args)
	if err != nil {
		return err
	}

	c.recordExchange(name, &RecordedExchange{Kind: kind, Durable: durable, AutoDelete: autoDelete, Args: args})
	return nil
}

func (c *Channel) ExchangeDelete(name string, ifUnused bool) error {
	c.deleteRecordedExchange(name)

	return c.Channel.ExchangeDelete(name, ifUnused, false)
}

func (c *Channel) ExchangeDeleteNoWait(name string, ifUnused bool) error {
	c.deleteRecordedExchange(name)

	return c.Channel.ExchangeDelete(name, ifUnused, true)
}

func (c *Channel) QueueDeclare(name string, durable bool, autoDelete bool, exclusive bool, args amqp.Table) (amqp.Queue, error) {
	queue, err := c.Channel.QueueDeclare(name, durable, autoDelete, exclusive, false, args)
	if err != nil {
		return amqp.Queue{}, err
	}

	c.recordedQueue(queue.Name, &RecordedQueue{Durable: durable, AutoDelete: autoDelete, Exclusive: exclusive, Args: args})
	return queue, nil
}

func (c *Channel) QueueDeclareNoWait(name string, durable bool, autoDelete bool, exclusive bool, args amqp.Table) (amqp.Queue, error) {
	queue, err := c.Channel.QueueDeclare(name, durable, autoDelete, exclusive, true, args)
	if err != nil {
		return amqp.Queue{}, err
	}

	c.recordedQueue(queue.Name, &RecordedQueue{Durable: durable, AutoDelete: autoDelete, Exclusive: exclusive, Args: args})
	return queue, nil
}

func (c *Channel) QueueDelete(name string, ifUnused, ifEmpty bool) (int, error) {
	c.deleteRecordedQueue(name)

	return c.Channel.QueueDelete(name, ifUnused, ifEmpty, false)
}

func (c *Channel) QueueDeleteNoWait(name string, ifUnused, ifEmpty bool) (int, error) {
	c.deleteRecordedQueue(name)

	return c.Channel.QueueDelete(name, ifUnused, ifEmpty, true)
}

func (c *Channel) QueueBind(name, key, exchange string, args amqp.Table) error {
	err := c.Channel.QueueBind(name, key, exchange, false, args)
	if err != nil {
		return err
	}

	c.recordQueueBinding(&RecordedBinding{QueueName: name, ExchangeName: exchange, RoutingKey: key, Args: args})
	return nil
}

func (c *Channel) QueueBindNoWait(name, key, exchange string, args amqp.Table) error {
	err := c.Channel.QueueBind(name, key, exchange, true, args)
	if err != nil {
		return err
	}

	c.recordQueueBinding(&RecordedBinding{QueueName: name, ExchangeName: exchange, RoutingKey: key, Args: args})
	return nil
}

func (c *Channel) QueueUnbind(name, key, exchange string, args amqp.Table) error {
	c.deleteRecordedQueueBinding(&RecordedBinding{QueueName: name, ExchangeName: exchange, RoutingKey: key, Args: args})
	return c.Channel.QueueUnbind(name, key, exchange, args)
}

func (c *Channel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return c.Channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

func (c *Channel) PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (*amqp.DeferredConfirmation, error) {
	return c.Channel.PublishWithDeferredConfirmWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

func (c *Channel) Consume(queue, consumerTag string, autoAck, exclusive, noLocal bool, args amqp.Table, consumer Consumer) error {
	return c.consume(queue, consumerTag, autoAck, exclusive, noLocal, false, args, consumer)
}

func (c *Channel) ConsumeNowait(queue, consumerTag string, autoAck, exclusive, noLocal bool, args amqp.Table, consumer Consumer) error {
	return c.consume(queue, consumerTag, autoAck, exclusive, noLocal, true, args, consumer)
}

func (c *Channel) consume(queue, consumerTag string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table, consumer Consumer) error {
	if consumerTag == "" {
		consumerTag = uniqueConsumerTag()
	}

	delivery, err := c.Channel.Consume(queue, consumerTag, autoAck, exclusive, noLocal, noWait, args)
	if err != nil {
		return err
	}

	// closeConsumer := make(chan struct{})
	// c.consumers[consumerTag] = closeConsumer

	// TODO:重连时先关闭之前的消费者和协程
	go func() {
		for {
			select {
			case d, ok := <-delivery:
				if !ok {
					// consumer cancelled
					return
				}

				consumer.HandleDelivery(d)

			case errClose, ok := <-c.chanNotifyClose:
				c.logger.Error(fmt.Sprintf("channel closed: %v ok:%t", errClose, ok))
				return
			// case <-closeConsumer:
			// 	return
			case <-c.close:
				c.logger.Error("channel closed")
				return
			}
		}
	}()

	c.recordConsumer(&RecordedConsumer{QueueName: queue, ConsumerTag: consumerTag, AutoAck: autoAck, Exclusive: exclusive, NoLocal: noLocal, NoWait: noWait, Args: args, Consumer: consumer})

	return nil
}

func (c *Channel) Cancel(consumerTag string, noWait bool) error {
	c.deleteRecordedConsumer(consumerTag)

	return c.Channel.Cancel(consumerTag, noWait)
}
