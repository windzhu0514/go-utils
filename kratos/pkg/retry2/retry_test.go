package retry2

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/windzhu0514/go-utils/utils"
)

var r *Retry

func TestMain(m *testing.M) {
	url := "amqp://admin:admintc123@10.100.173.206:9073"
	logger := log.DefaultLogger
	var err error
	r, err = New(url, logger, "test.delayed.exchange", "test.delayed.queue", 3, func(msg *RetryMsg) error {
		return nil
	})
	if err != nil {
		panic(err)
	}

	m.Run()
}

func TestPush(t *testing.T) {
	for {
		msg := &RetryMsg{
			FirstDelayed: time.Minute,
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
