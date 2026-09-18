package delayqueue

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/windzhu0514/go-utils/delayqueue/backoff"
	"github.com/windzhu0514/go-utils/utils"
)

var r *DelayQueue

func TestMain(m *testing.M) {
	url := "amqp://admin:AYJdpnDqVy5Rp4DL@10.177.9.244:35202"
	logger := log.DefaultLogger
	var err error
	r, err = New(logger, url, "test.delayed.exchange", "test.delayed.queue", nil, func(msg Message) error {
		fmt.Println(string(msg.Body))
		return errors.New("test")
	})
	if err != nil {
		panic(err)
	}

	m.Run()
}

func TestPublish(t *testing.T) {
	r.RegisterBackoff("test", backoff.NewExponentialPolicy(5*time.Second, 5*time.Second, 2, 20*time.Second))

	msg := &DelayMessage{
		TotalTimes:  10,
		Body:        []byte("test"),
		BackOffName: "test",
	}
	fmt.Println(utils.JsonMarshalString(msg))
	err := r.Publish(msg)
	if err != nil {
		t.Error(err)
	}

	select {}
}

func TestNewOrder(t *testing.T) {
	logger := log.DefaultLogger
	r, err := New(logger,
		"amqp://admin:AYJdpnDqVy5Rp4DL@10.177.9.244:35202",
		"hitch.fulfill.querydriverorderinfo.t",
		"hitch.fulfill.querydriverorderinfo.t", nil, func(msg Message) error {
			return nil
		})
	if err != nil {
		panic(err)
	}
	c, err := r.amqpConn.Channel()
	if err != nil {
		t.Log("channel err:", err)
		return
	}
	msg, err := c.Consume("hitch.fulfill.querydriverorderinfo.t", "", true, false, false, false, nil)
	if err != nil {
		t.Log("comsume err:", err)
		return
	}
	for range msg {
	}
}

func TestRegisterBackOff(t *testing.T) {
	r.RegisterBackoff("test", backoff.NewExponentialPolicy(20*time.Millisecond, 50*time.Millisecond, 2, 1200*time.Millisecond))
	backOff := r.backoffs["test"]
	fmt.Println(backOff.BackOff(1))
	fmt.Println(backOff.BackOff(2))
	fmt.Println(backOff.BackOff(3))
	fmt.Println(backOff.BackOff(4))
	fmt.Println(backOff.BackOff(5))
	fmt.Println(backOff.BackOff(6))
	fmt.Println(backOff.BackOff(7))
	fmt.Println(backOff.BackOff(8))
	fmt.Println(backOff.BackOff(9))
	fmt.Println(backOff.BackOff(10))
	// output:
	// 20ms
	// 50ms
	// 100ms
	// 200ms
	// 400ms
	// 800ms
	// 1.2s
	// 1.2s
	// 1.2s
	// 1.2s
}
