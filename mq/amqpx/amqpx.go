package amqpx

import (
	"log"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Dial(url string) (*Connection, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		url:        url,
		logger:     slog.Default(),
		Connection: conn,
		close:      make(chan struct{}),
	}

	c.chanNotifyClose = make(chan *amqp.Error)
	c.Connection.NotifyClose(c.chanNotifyClose)

	go c.handleReconnect()

	return c, nil
}
