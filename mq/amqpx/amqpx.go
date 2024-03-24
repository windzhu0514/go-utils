package amqpx

import (
	"log"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

func New(url string) (*Connection, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
