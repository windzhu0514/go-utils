package delaypolicy

import (
	"math/rand"
	"time"
)

type DelayPolicy interface {
	MaxAttempts() int
	InitialDelay() time.Duration // 首次延迟时间
	Delay() time.Duration        // 延迟时间
}

type FixedDelayPolicy struct {
	initialDelay time.Duration
	fixedDelay   time.Duration
	maxAttempts  int
}

func NewFixedDelayPolicy(initialDelay, fixedDelay time.Duration, maxAttempts int) DelayPolicy {
	return &FixedDelayPolicy{initialDelay, fixedDelay, maxAttempts}
}

func (p *FixedDelayPolicy) MaxAttempts() int {
	return p.maxAttempts
}

func (p *FixedDelayPolicy) InitialDelay() time.Duration {
	return p.initialDelay
}

func (p *FixedDelayPolicy) Delay() time.Duration {
	return p.fixedDelay
}

type ExponentialDelayPolicy struct {
	interval    time.Duration
	multiplier  float64
	maxInterval time.Duration
	maxAttempts int
	attempt     int
}

func NewExponentialDelayPolicy(interval time.Duration, multiplier float64, maxInterval time.Duration, maxAttempts int) DelayPolicy {
	return &ExponentialDelayPolicy{
		interval:    interval,
		multiplier:  multiplier,
		maxInterval: maxInterval,
		maxAttempts: maxAttempts,
	}
}

func (p *ExponentialDelayPolicy) MaxAttempts() int {
	return p.maxAttempts
}

func (p *ExponentialDelayPolicy) InitialDelay() time.Duration {
	return p.interval
}

func (p *ExponentialDelayPolicy) Delay() time.Duration {
	if p.attempt == 0 {
		return p.interval
	}

	p.interval = time.Duration(float64(p.interval) * p.multiplier)
	if p.interval > p.maxInterval {
		p.interval = p.maxInterval
	}

	return p.interval
}

type ExponentialRandDelayPolicy struct {
	ExponentialDelayPolicy
}

func NewExponentialRandDelayPolicy(interval time.Duration, multiplier float64, maxInterval time.Duration, maxAttempts int) DelayPolicy {
	return &ExponentialRandDelayPolicy{
		ExponentialDelayPolicy: ExponentialDelayPolicy{
			interval:    interval,
			multiplier:  multiplier,
			maxInterval: maxInterval,
			maxAttempts: maxAttempts,
		},
	}
}

func (p *ExponentialRandDelayPolicy) MaxAttempts() int {
	return p.maxAttempts
}

func (p *ExponentialRandDelayPolicy) InitialDelay() time.Duration {
	return p.interval
}

func (p *ExponentialRandDelayPolicy) Delay() time.Duration {
	if p.attempt == 0 {
		p.attempt++
		return p.interval
	}

	p.attempt++

	p.interval = time.Duration(float64(p.interval) * (1 + rand.Float64()*(p.multiplier-1)))
	if p.interval > p.maxInterval {
		p.interval = p.maxInterval
	}

	return p.interval
}
