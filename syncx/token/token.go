package token

type token struct {
	token chan struct{}
}

// New 新建一个令牌分发
func New(n int) *token {
	var x token
	x.token = make(chan struct{}, n)
	return &x
}

func (x *token) Add() {
	x.token <- struct{}{}
}

func (x *token) Done() {
	<-x.token
}
