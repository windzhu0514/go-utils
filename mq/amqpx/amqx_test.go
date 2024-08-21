package amqpx

import (
	"fmt"
	"testing"
	"time"
)

func TestExponentialRandomBackoff(t *testing.T) {
	bo := NewExponentialRandomBackoff(time.Second, 2)
	fmt.Println(bo.Delay(1))
	fmt.Println(bo.Delay(2))
	fmt.Println(bo.Delay(3))
}
