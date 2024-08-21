package amqpx

import (
	"maps"
	"slices"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RecordedExchange struct {
	Kind       string
	Durable    bool
	AutoDelete bool
	Args       amqp.Table
}

type RecordedQueue struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	Args       amqp.Table
}

type RecordedBinding struct {
	QueueName    string
	ExchangeName string
	RoutingKey   string
	Args         amqp.Table
}

func (rb *RecordedBinding) Equal(another *RecordedBinding) bool {
	return rb.QueueName == another.QueueName && rb.ExchangeName == another.ExchangeName && rb.RoutingKey == another.RoutingKey && maps.Equal(rb.Args, another.Args)
}

type RecordedConsumer struct {
	QueueName   string
	ConsumerTag string
	AutoAck     bool
	Exclusive   bool
	NoLocal     bool
	NoWait      bool
	Args        amqp.Table
	Consumer    Consumer
}

func (c *Channel) recordExchange(exchangeName string, x *RecordedExchange) {
	c.recordedExchanges[exchangeName] = x
}

func (c *Channel) deleteRecordedExchange(exchangeName string) {
	delete(c.recordedExchanges, exchangeName)
}

func (c *Channel) recordedQueue(queueName string, x *RecordedQueue) {
	c.recordedQueues[queueName] = x
}

func (c *Channel) deleteRecordedQueue(queueName string) {
	delete(c.recordedQueues, queueName)
}

func (c *Channel) recordQueueBinding(x *RecordedBinding) {
	c.recordedBindings = slices.DeleteFunc(c.recordedBindings, func(i *RecordedBinding) bool {
		return i.Equal(x)
	})

	c.recordedBindings = append(c.recordedBindings, x)
}

func (c *Channel) deleteRecordedQueueBinding(x *RecordedBinding) {
	c.recordedBindings = slices.DeleteFunc(c.recordedBindings, func(i *RecordedBinding) bool {
		return i.Equal(x)
	})
}

func (c *Channel) recordConsumer(x *RecordedConsumer) {
	c.recordedConsumers[x.ConsumerTag] = x
}

func (c *Channel) deleteRecordedConsumer(consumerTag string) {
	delete(c.recordedConsumers, consumerTag)
}

func (c *Channel) recover() {
	if err := c.Qos(c.recordedPrefetchCount, c.recordedPrefetchSize, c.recordedPrefetchGlobal); err != nil {
		c.logger.Error("failed to set Qos", "error", err)
	}

	for name, v := range c.recordedExchanges {
		if err := c.ExchangeDeclare(name, v.Kind, v.Durable, v.AutoDelete, false, v.Args); err != nil {
			c.logger.Error("failed to declare exchange", "name", name, "error", err)
		}
	}

	for name, v := range c.recordedQueues {
		if _, err := c.QueueDeclare(name, v.Durable, v.AutoDelete, v.Exclusive, v.Args); err != nil {
			c.logger.Error("failed to declare queue", "name", name, "error", err)
		}
	}

	for _, v := range c.recordedBindings {
		if err := c.QueueBind(v.QueueName, v.RoutingKey, v.ExchangeName, v.Args); err != nil {
			c.logger.Error("failed to bind queue", "queue", v.QueueName, "exchange", v.ExchangeName, "routingKey", v.RoutingKey, "error", err)
		}
	}

	for _, v := range c.recordedConsumers {
		if err := c.consume(v.QueueName, v.ConsumerTag, v.AutoAck, v.Exclusive, v.NoLocal, v.NoWait, v.Args, v.Consumer); err != nil {
			c.logger.Error("failed to consume", "queue", v.QueueName, "consumerTag", v.ConsumerTag, "error", err)
		}
	}
}
