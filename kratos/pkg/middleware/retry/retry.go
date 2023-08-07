package retry

import (
	"context"
	"crypto/x509"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
)

type (
	Checker       func(ctx context.Context, resp *http.Response, err error) (bool, error)
	BackoffPolicy func(times, maxTimes int, retryWaitMin, retryWaitMax time.Duration) time.Duration
)

type Option func(*options)

func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func WithChecker(checker Checker) Option {
	return func(o *options) {
		o.retryChecker = checker
	}
}

func WithBackoffPolicy(policy BackoffPolicy) Option {
	return func(o *options) {
		o.backoffPolicy = policy
	}
}

func WithMaxTimes(maxTimes int) Option {
	return func(o *options) {
		o.retryMaxTimes = maxTimes
	}
}

func WithWaitMin(wait time.Duration) Option {
	return func(o *options) {
		o.retryWaitMin = wait
	}
}

func WithWaitMax(wait time.Duration) Option {
	return func(o *options) {
		o.retryWaitMax = wait
	}
}

type options struct {
	logger        log.Logger
	retryChecker  Checker
	backoffPolicy BackoffPolicy
	retryMaxTimes int
	retryWaitMin  time.Duration // Minimum time to wait
	retryWaitMax  time.Duration // Maximum time to wait
}

func New(opts ...Option) middleware.Middleware {
	options := &options{
		retryChecker:  defaultRetryChecker,
		backoffPolicy: DefaultBackoff,
		retryMaxTimes: 3,
		retryWaitMin:  1 * time.Second,
		retryWaitMax:  30 * time.Second,
	}

	for _, o := range opts {
		o(options)
	}

	if options.retryWaitMax <= options.retryWaitMin {
		options.retryWaitMax = options.retryWaitMin
	}

	return func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			for i := 0; ; i++ {
				if i >= options.retryMaxTimes {
					break
				}

				reply, err = h(ctx, req)
				if err == nil {
					return reply, nil
				}

				shouldRetry, err := options.retryChecker(ctx, nil, err)
				if err != nil || !shouldRetry {
					return nil, err
				}

				if options.logger != nil {
					_ = options.logger.Log(log.LevelInfo, log.DefaultMessageKey, "retry.retry", "times", i+1)
				}

				wait := options.backoffPolicy(i, options.retryMaxTimes, options.retryWaitMin, options.retryWaitMax)
				time.Sleep(wait)
			}

			return
		}
	}
}

var (
	// 匹配 net/http 返回的达到最大重定向次数返回的错误
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)
	// 匹配 net/http 返回的协议无效错误
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
	// 匹配 net/http 返回的 TLS certificate 不信任错误
	notTrustedErrorRe = regexp.MustCompile(`certificate is not trusted`)
)

func defaultRetryChecker(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, v
			}

			if schemeErrorRe.MatchString(v.Error()) {
				return false, v
			}

			if notTrustedErrorRe.MatchString(v.Error()) {
				return false, v
			}

			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, v
			}
		}

		return true, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented) {
		return true, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	return false, nil
}

func DefaultBackoff(times, maxTimes int, retryWaitMin, retryWaitMax time.Duration) time.Duration {
	return retryWaitMin
}

func ExponentialBackoff(times, maxTimes int, retryWaitMin, retryWaitMax time.Duration) time.Duration {
	if times >= maxTimes {
		times = maxTimes
	}

	exp := math.Pow(2, float64(times))
	return retryWaitMin * time.Duration(exp)
}

func LinearJitterBackoff(times, maxTimes int, retryWaitMin, retryWaitMax time.Duration) time.Duration {
	times++

	random := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	jitter := random.Float64() * float64(retryWaitMax-retryWaitMin)
	jitterMin := int64(jitter) + int64(retryWaitMin)
	return time.Duration(jitterMin * int64(times))
}
