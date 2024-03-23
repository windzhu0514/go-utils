package amqpx

import (
	"os"
	"strconv"
	"sync/atomic"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer interface {
	HandleDelivery(delivery amqp.Delivery)
}

var consumerSeq uint64

const consumerTagLengthMax = 0xFF // see writeShortstr

func uniqueConsumerTag() string {
	return commandNameBasedUniqueConsumerTag(os.Args[0])
}

func commandNameBasedUniqueConsumerTag(commandName string) string {
	tagPrefix := "ctag-"
	tagInfix := commandName
	tagSuffix := "-" + strconv.FormatUint(atomic.AddUint64(&consumerSeq, 1), 10)

	if len(tagPrefix)+len(tagInfix)+len(tagSuffix) > consumerTagLengthMax {
		tagInfix = "rabbitmq/amqp"
	}

	return tagPrefix + tagInfix + tagSuffix
}
