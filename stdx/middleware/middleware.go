package middleware

import (
	"net/http"
)

type Middleware func(http.Handler) http.Handler

type chain struct {
	mws []Middleware
}

func NewChain(mws ...Middleware) *chain {
	c := &chain{}
	c.mws = append(c.mws, mws...)
	return c
}

func (c *chain) Use(mws ...Middleware) {
	c.mws = append(c.mws, mws...)
}

func (c *chain) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := len(c.mws) - 1; i >= 0; i-- {
			next = c.mws[i](next)
		}
		next.ServeHTTP(w, r)
	})
}

func (c *chain) HandleFunc(next http.HandlerFunc) http.HandlerFunc {
	nextHandler := http.Handler(next)
	for i := len(c.mws) - 1; i >= 0; i-- {
		nextHandler = c.mws[i](nextHandler)
	}

	return nextHandler.ServeHTTP
}
