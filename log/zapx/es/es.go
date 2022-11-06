package es

import "go.uber.org/zap/zapcore"

// https://github.com/sohlich/elogrus
// https://github.com/go-extras/elogrus with the official client

type es struct{}

type Option struct{}

func (es) New(opts ...Option) zapcore.WriteSyncer {
	return &es{}
}

func (ws *es) Write(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (ws *es) Sync() error {
	panic("not implemented") // TODO: Implement
}
