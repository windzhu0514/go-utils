// TODO: publish retry
// TODO: 重连是否加锁
package amqpx

import (
	"log"
	"log/slog"
	"math"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Dial(url string, opts ...ConnectionOption) (*Connection, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		url:             url,
		logger:          slog.Default(),
		Connection:      conn,
		close:           make(chan struct{}),
		recoveryBackoff: NewFixedBackoff(5 * time.Second),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.chanNotifyClose = make(chan *amqp.Error)
	c.Connection.NotifyClose(c.chanNotifyClose)

	go c.handleReconnect()

	return c, nil
}

type ConnectionOption func(*Connection)

func WithLogger(logger *slog.Logger) ConnectionOption {
	return func(c *Connection) {
		c.logger = logger
	}
}

func WithRecoveryMaxAttempts(maxAttempts int) ConnectionOption {
	return func(c *Connection) {
		c.recoveryMaxAttempts = maxAttempts
	}
}

func WithRecoveryBackoff(backoff RecoveryBackoff) ConnectionOption {
	return func(c *Connection) {
		c.recoveryBackoff = backoff
	}
}

type RecoveryBackoff interface {
	// attempts 从0开始
	Delay(attempts int) time.Duration
}

func NewFixedBackoff(interval time.Duration) RecoveryBackoff {
	return &FixedBackoff{
		interval: interval,
	}
}

type FixedBackoff struct {
	interval time.Duration
}

func (f *FixedBackoff) Delay(attempts int) time.Duration {
	return f.interval
}

func NewExponentialBackoff(initialInterval time.Duration, multiplier float64) RecoveryBackoff {
	return &ExponentialBackoff{
		initialInterval: initialInterval,
		multiplier:      multiplier,
	}
}

type ExponentialBackoff struct {
	initialInterval time.Duration
	multiplier      float64
}

func (e *ExponentialBackoff) Delay(attempts int) time.Duration {
	return time.Duration(float64(e.initialInterval) * math.Pow(e.multiplier, float64(attempts)))
}

func NewExponentialRandomBackoff(initialInterval time.Duration, multiplier float64) RecoveryBackoff {
	return &ExponentialRandomBackoff{
		initialInterval: initialInterval,
		multiplier:      multiplier,
	}
}

type ExponentialRandomBackoff struct {
	initialInterval time.Duration
	multiplier      float64
}

func (e *ExponentialRandomBackoff) Delay(attempts int) time.Duration {
	delay := float64(e.initialInterval) * math.Pow(e.multiplier, float64(attempts))
	return time.Duration(delay * (1 + rand.Float64()*(e.multiplier-1)))
}

func NewCustomBackoff(intervals ...time.Duration) RecoveryBackoff {
	return &CustomBackoff{
		intervals: intervals,
	}
}

type CustomBackoff struct {
	intervals []time.Duration
}

func (e *CustomBackoff) Delay(attempts int) time.Duration {
	if attempts >= len(e.intervals) {
		return e.intervals[len(e.intervals)-1]
	}

	return e.intervals[attempts]
}
