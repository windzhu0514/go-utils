package recovery

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"

	"github.com/windzhu0514/go-utils/stdx/middleware"
)

type Option func(*options)

type options struct {
	logger     *zap.Logger
	printStack bool
}

func WithLogger(logger *zap.Logger) Option {
	return func(opts *options) {
		opts.logger = logger
	}
}

func WithStack(printStack bool) Option {
	return func(opts *options) {
		opts.printStack = printStack
	}
}

func Recovery(opts ...Option) middleware.Middleware {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if o.logger != nil {
						fields := make([]zap.Field, 0, 1)
						if o.printStack {
							fields = append(fields, zap.String("stack", string(debug.Stack())))
						}
						o.logger.Error(fmt.Sprintf("%v", err), fields...)
					}

					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte("Internal Server Error"))
				}
			}()

			h.ServeHTTP(w, r)
		})
	}
}
