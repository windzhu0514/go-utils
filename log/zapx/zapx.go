package zapx

import (
	"context"

	"go.uber.org/zap"
)

var Default = func() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}()

type loggerKey struct{}

func NewContext(ctx context.Context, logger *zap.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if logger == nil {
		return ctx
	}

	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return Default
	}

	if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return logger
	}

	return Default
}
