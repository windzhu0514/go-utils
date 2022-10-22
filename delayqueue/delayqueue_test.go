package delayqueue

import (
	"fmt"
	"testing"
	"time"

	"git.17usoft.com/GSAirGroup/BookingService/pkg/utils"
	"github.com/go-kratos/kratos/v2/log"
)

var r *DelayQueue

func TestMain(m *testing.M) {
	url := "amqp://admin:admintc123@10.100.173.206:9073"
	logger := log.DefaultLogger
	var err error
	r, err = New(url, logger, "test.delayed.exchange", "test.delayed.queue", 3, func(msg *Message) error {
		return nil
	})
	if err != nil {
		panic(err)
	}

	m.Run()
}

func TestPush(t *testing.T) {
	for {
		msg := &Message{
			InitialDelay: time.Minute,
			Delayed:      time.Minute,
			TotalTimes:   3,
			Body:         []byte("test"),
			CreateAt:     time.Now(),
		}
		fmt.Println(utils.JsonMarshalString(msg))
		err := r.Publish(msg)
		if err != nil {
			t.Error(err)
		}

		time.Sleep(time.Second * 20)
	}
	select {}
}
