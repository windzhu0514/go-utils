package backoff

import (
	"fmt"
	"testing"
	"time"
)

func TestExponentialRandDelayPolicy(t *testing.T) {
	p := NewExponentialPolicy(20*time.Millisecond, 50*time.Millisecond, 2, 1200*time.Millisecond)
	for i := 1; i < 10; i++ {
		fmt.Println(p.BackOff(i))
	}
}

func TestExponentialDelayPolicy(t *testing.T) {
	p := NewExponentialRandPolicy(20*time.Millisecond, 50*time.Millisecond, 2, 1200*time.Millisecond)
	for i := 1; i < 10; i++ {
		fmt.Println(p.BackOff(i))
	}
}
