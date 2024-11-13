package logging

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Server is an server logging middleware.
func Server(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				kind      string
				operation string
			)
			startTime := time.Now()
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}

			// reqData, _ := json.Marshal(req)
			_ = logger.Log(log.LevelInfo,
				"kind", "server",
				"component", kind,
				"operation", operation,
				// "request", string(reqData),
				log.DefaultMessageKey, "request received",
			)

			reply, err = handler(ctx, req)

			// replyData, _ := json.Marshal(reply)
			_ = logger.Log(log.LevelInfo,
				"kind", "server",
				"component", kind,
				"operation", operation,
				// "response", string(replyData),
				"latency", time.Since(startTime).Seconds(),
				"error", err,
				log.DefaultMessageKey, "request processed",
			)
			return
		}
	}
}

// Client is an client logging middleware.
func Client(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				kind      string
				operation string
			)
			startTime := time.Now()
			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}

			// reqData, _ := json.Marshal(req)
			_ = log.WithContext(ctx, logger).Log(log.LevelInfo,
				"kind", "client",
				"component", kind,
				"operation", operation,
				// "request", string(reqData),
				log.DefaultMessageKey, "request received",
			)

			reply, err = handler(ctx, req)

			// replyData, _ := json.Marshal(reply)
			_ = log.WithContext(ctx, logger).Log(log.LevelInfo,
				"kind", "client",
				"component", kind,
				"operation", operation,
				// "response", string(replyData),
				"latency", time.Since(startTime).Seconds(),
				"error", err,
				log.DefaultMessageKey, "request processed",
			)
			return
		}
	}
}
