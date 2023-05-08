package backoff

import (
	"math"
	"math/rand"
	"time"
)

type Policy interface {
	BackOff(attempts int) (delay time.Duration) // 本次延迟时间，attempts 从1开始
}

type noPolicy struct{}

func NewNoPolicy() Policy {
	return &fixedPolicy{}
}

func (p *noPolicy) BackOff(attempts int) time.Duration {
	return 0
}

type fixedPolicy struct {
	InitialDelay time.Duration
	FixedDelay   time.Duration
}

func NewFixedPolicy(initialDelay, fixedDelay time.Duration) Policy {
	return &fixedPolicy{initialDelay, fixedDelay}
}

func (p *fixedPolicy) BackOff(attempts int) time.Duration {
	if attempts == 1 {
		return p.InitialDelay
	}

	return p.FixedDelay
}

type exponentialPolicy struct {
	InitialDelay time.Duration
	Interval     time.Duration
	Multiplier   float64
	MaxInterval  time.Duration
}

func NewExponentialPolicy(initialDelay, interval time.Duration, multiplier float64, maxInterval time.Duration) Policy {
	return &exponentialPolicy{
		InitialDelay: initialDelay,
		Interval:     interval,
		Multiplier:   multiplier,
		MaxInterval:  maxInterval,
	}
}

func (p *exponentialPolicy) BackOff(attempts int) time.Duration {
	if attempts == 1 {
		return p.InitialDelay
	}

	if attempts == 2 {
		return p.Interval
	}

	delay := time.Duration(float64(p.Interval) * math.Pow(p.Multiplier, float64(attempts-2)))
	if p.MaxInterval > 0 && delay > p.MaxInterval {
		delay = p.MaxInterval
	}

	if delay < p.Interval {
		return p.Interval
	}

	return delay
}

type exponentialRandPolicy struct {
	exponentialPolicy
}

func NewExponentialRandPolicy(initialDelay, interval time.Duration, multiplier float64, maxInterval time.Duration) Policy {
	return &exponentialRandPolicy{
		exponentialPolicy: exponentialPolicy{
			InitialDelay: initialDelay,
			Interval:     interval,
			Multiplier:   multiplier,
			MaxInterval:  maxInterval,
		},
	}
}

func (p *exponentialRandPolicy) BackOff(attempts int) time.Duration {
	if attempts == 1 {
		return p.InitialDelay
	}

	if attempts == 2 {
		return p.Interval
	}

	r := 1 + rand.Float64()*(p.Multiplier-1)
	delay := time.Duration(float64(p.Interval) * math.Pow(p.Multiplier, float64(attempts-1)) * r)
	if delay > p.MaxInterval {
		delay = p.MaxInterval
	}

	if delay < p.Interval {
		return p.Interval
	}

	return delay
}

type customPolicy struct {
	Intervals []time.Duration
}

func NewCustomPolicy(intervals []time.Duration) Policy {
	return &customPolicy{
		Intervals: intervals,
	}
}

func (p *customPolicy) BackOff(attempts int) time.Duration {
	if len(p.Intervals) == 0 {
		return 0
	}

	idx := attempts - 1
	if idx < 0 {
		idx = 0
	}

	if idx < len(p.Intervals) {
		return p.Intervals[idx]
	}

	return p.Intervals[len(p.Intervals)-1]
}
