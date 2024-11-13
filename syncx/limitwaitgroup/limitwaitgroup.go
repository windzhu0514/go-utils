package limitwaitgroup

import (
	"sync"
)

type LimitWaitGroup struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

func New(n int) *LimitWaitGroup {
	return &LimitWaitGroup{
		sem: make(chan struct{}, n),
	}
}

func (l *LimitWaitGroup) Add() {
	l.sem <- struct{}{}
	l.wg.Add(1)
}

func (l *LimitWaitGroup) Done() {
	<-l.sem
	l.wg.Done()
}

func (l *LimitWaitGroup) Wait() {
	l.wg.Wait()
}
