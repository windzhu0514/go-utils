package delaypolicy

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestNewExponentialRandDelayPolicy(t *testing.T) {
	rand.Seed(time.Now().UnixMilli())
	p := NewExponentialRandDelayPolicy(2*time.Second, 3, 10*time.Second, 5)
	fmt.Println(p.Delay())
	fmt.Println(p.Delay())
	fmt.Println(p.Delay())
	fmt.Println(p.Delay())
}
