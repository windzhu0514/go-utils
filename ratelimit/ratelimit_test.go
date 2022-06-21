package limit

import (
	"fmt"
	"testing"
	"time"
)

func TestFixedWindowRateLimit1(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func() {
			// 限制1秒钟5次
			rs := FixedWindowRateLimit("test1", 1*time.Second, 5)
			fmt.Println("result is:", rs)
		}()
	}

	fmt.Println("end")
	select {}
}

// 从30秒开始
// === RUN   TestFixedWindowRateLimit2
// time range from 0 to 30
// time range from 30 to 35
// result is: true
// time range from 35 to 40
// result is: true
// time range from 40 to 45
// result is: true
// time range from 45 to 50
// result is: true
// time range from 50 to 55
// result is: true
// time range from 55 to 60
// result is: false
// time range from 60 to 65
// result is: false
// time range from 65 to 70
// result is: false
// time range from 70 to 75
// result is: false
// time range from 75 to 80
// result is: false
// end
func TestFixedWindowRateLimit2(t *testing.T) {
	fillInteval := 1 * time.Minute
	var limitNum int64 = 5
	waitTime := 30
	fmt.Printf("time range from 0 to %d\n", waitTime)
	time.Sleep(time.Duration(waitTime) * time.Second)
	for i := 0; i < 20; i++ {
		fmt.Printf("time range from %d to %d\n", i*10+waitTime, (i+1)*10+waitTime)
		rs := FixedWindowRateLimit("test2", fillInteval, limitNum)
		fmt.Println("result is:", rs)
		time.Sleep(10 * time.Second)
	}

	fmt.Println("end")
	select {}
}

func TestSlidingWindowRatelimit(t *testing.T) {
	fillInteval := 1 * time.Minute
	var limitNum int64 = 5
	var segmentNum int64 = 6
	waitTime := 30
	fmt.Printf("time range from 0 to %d\n", waitTime)
	time.Sleep(time.Duration(waitTime) * time.Second)
	for i := 0; i < 10; i++ {
		fmt.Printf("time range from %d to %d\n", i*10+waitTime, (i+1)*10+waitTime)
		for j := 0; j < 8; j++ {
			rs := SlidingWindowRatelimit("test", fillInteval, segmentNum, limitNum)
			fmt.Println("result is:", rs)
		}
		time.Sleep(10 * time.Second)
	}
}
