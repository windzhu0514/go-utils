package backoff

import (
	"math"
	"math/rand"
	"time"
)

type BackOffPolicy interface {
	BackOff(retryCount int, maxAttempts int) (delay time.Duration) // 延迟时间
}

type noPolicy struct{}

func NewNoPolicy() BackOffPolicy {
	return &fixedPolicy{}
}

func (p *noPolicy) BackOff(retryCount int, maxAttempts int) time.Duration {
	return 0
}

type fixedPolicy struct {
	initialDelay time.Duration
	fixedDelay   time.Duration
}

func NewFixedPolicy(initialDelay, fixedDelay time.Duration) BackOffPolicy {
	return &fixedPolicy{initialDelay, fixedDelay}
}

func (p *fixedPolicy) BackOff(retryCount int, maxAttempts int) time.Duration {
	if retryCount == 0 {
		return p.initialDelay
	}

	return p.fixedDelay
}

type exponentialPolicy struct {
	initialDelay time.Duration
	interval     time.Duration
	multiplier   float64
	maxInterval  time.Duration
}

func NewExponentialPolicy(initialDelay, interval time.Duration, multiplier float64, maxInterval time.Duration) BackOffPolicy {
	return &exponentialPolicy{
		initialDelay: initialDelay,
		interval:     interval,
		multiplier:   multiplier,
		maxInterval:  maxInterval,
	}
}

func (p *exponentialPolicy) BackOff(retryCount int, maxAttempts int) time.Duration {
	if retryCount == 0 {
		return p.initialDelay
	}

	if retryCount == 1 {
		return p.interval
	}

	delay := time.Duration(float64(p.interval) * math.Pow(p.multiplier, float64(retryCount-1)))
	if p.maxInterval > 0 && delay > p.maxInterval {
		delay = p.maxInterval
	}

	return delay
}

type exponentialRandPolicy struct {
	exponentialPolicy
}

func NewExponentialRandPolicy(initialDelay, interval time.Duration, multiplier float64, maxInterval time.Duration) BackOffPolicy {
	return &exponentialRandPolicy{
		exponentialPolicy: exponentialPolicy{
			initialDelay: initialDelay,
			interval:     interval,
			multiplier:   multiplier,
			maxInterval:  maxInterval,
		},
	}
}

func (p *exponentialRandPolicy) BackOff(retryCount int, maxAttempts int) time.Duration {
	if retryCount == 0 {
		return p.initialDelay
	}

	if retryCount == 1 {
		return p.interval
	}

	r := 1 + rand.Float64()*(p.multiplier-1)
	delay := time.Duration(float64(p.interval) * math.Pow(p.multiplier, float64(retryCount-1)) * r)
	if delay > p.maxInterval {
		delay = p.maxInterval
	}

	return delay
}

type customPolicy struct {
	intervals []time.Duration
}

func NewCustomPolicy(intervals []time.Duration) BackOffPolicy {
	return &customPolicy{
		intervals: intervals,
	}
}

func (p *customPolicy) BackOff(retryCount int, maxAttempts int) time.Duration {
	if retryCount < len(p.intervals) {
		return p.intervals[retryCount]
	}

	if len(p.intervals) == 0 {
		return 0
	}

	return p.intervals[len(p.intervals)-1]
}
