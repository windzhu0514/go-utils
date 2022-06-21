package limit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// Go：分布式高并发服务限流实现方案
// https://mp.weixin.qq.com/s/GkEAVQ6AWGRadB4fEBlepg?utm_source=wechat_session&utm_medium=social&utm_oi=43120666345472
// https://github.com/skyhackvip/ratelimit
var client *redis.Client

func init() {
	client = redis.NewClient(&redis.Options{
		Addr:     "10.181.20.68:6379",
		Password: "TkFXc2Ju7GbRCyDH",
		DB:       0,
	})
}

func FixedWindowRateLimit(key string, fillInterval time.Duration, limitNum int64) bool {
	// current tick time window
	tick := int64(time.Now().Unix() / int64(fillInterval.Seconds()))
	currentKey := fmt.Sprintf("%s_%d_%d_%d", key, fillInterval, limitNum, tick)
	fmt.Println(currentKey)

	startCount := 0
	_, err := client.SetNX(context.Background(), currentKey, startCount, fillInterval).Result()
	if err != nil {
		panic(err)
	}
	// number in current time window
	quantum, err := client.Incr(context.Background(), currentKey).Result()
	if err != nil {
		panic(err)
	}
	if quantum > limitNum {
		return false
	}
	return true
}

// segmentNum split inteval time into smaller segments
func SlidingWindowRatelimit(key string, fillInteval time.Duration, segmentNum int64, limitNum int64) bool {
	segmentInteval := fillInteval.Seconds() / float64(segmentNum)
	tick := float64(time.Now().Unix()) / segmentInteval
	currentKey := fmt.Sprintf("%s_%d_%d_%d_%f", key, fillInteval, segmentNum, limitNum, tick)
	fmt.Println(currentKey)

	startCount := 0
	_, err := client.SetNX(context.Background(), currentKey, startCount, fillInteval).Result()
	if err != nil {
		panic(err)
	}
	quantum, err := client.Incr(context.Background(), currentKey).Result()
	if err != nil {
		panic(err)
	}
	// add in the number of the previous time
	for tickStart := segmentInteval; tickStart < fillInteval.Seconds(); tickStart += segmentInteval {
		tick = tick - 1
		preKey := fmt.Sprintf("%s_%d_%d_%d_%f", key, fillInteval, segmentNum, limitNum, tick)
		val, err := client.Get(context.Background(), preKey).Result()
		if err != nil {
			val = "0"
		}
		num, err := strconv.ParseInt(val, 0, 64)
		quantum = quantum + num
		if quantum > limitNum {
			client.Decr(context.Background(), currentKey).Result()
			return false
		}
	}
	return true
}
