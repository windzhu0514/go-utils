package delayqueue

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/windzhu0514/go-utils/utils"
)

var r *DelayQueue

func TestMain(m *testing.M) {
	url := "amqp://admin:admintc123@10.100.173.206:9073"
	logger := log.DefaultLogger
	var err error
	r, err = New(logger, url, "test.delayed.exchange", "test.delayed.queue", nil, func(msg *Message) error {
		return nil
	})
	if err != nil {
		panic(err)
	}

	m.Run()
}

func TestPush(t *testing.T) {
	for {
		msg := &DelayMessage{
			TotalTimes: 3,
			Body:       []byte("test"),
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
