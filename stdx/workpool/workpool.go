// Package workpool provides a simple worker pool implementation.
//
// It is a simple and efficient worker pool implementation that allows you to
// process some items (or a channel of items) in parallel.
//
// Be careful, the results order is not guaranteed when numWorkers > 1.
package workpool

import (
	"sync"
)

type WorkerPool[T any, R any] struct {
	numWorkers int
	process    func(T) R
}

func New[T any, R any](numWorkers int, process func(T) R) *WorkerPool[T, R] {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	return &WorkerPool[T, R]{
		numWorkers: numWorkers,
		process:    process,
	}
}

func (wp *WorkerPool[T, R]) Run(items []T) []R {
	results := wp.RunAsync(items)

	var output []R
	for result := range results {
		output = append(output, result)
	}

	return output
}

func (wp *WorkerPool[T, R]) RunAsync(items []T) <-chan R {
	if len(items) == 0 {
		return nil
	}

	if wp.process == nil {
		return nil
	}

	var wg sync.WaitGroup
	jobs := make(chan T, len(items))
	results := make(chan R, len(items))

	for i := 1; i <= wp.numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				results <- wp.process(item)
			}
		}()
	}

	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}

func (wp *WorkerPool[T, R]) RunWithChannel(jobs <-chan T) []R {
	results := wp.RunWithChannelAsync(jobs)

	var output []R
	for result := range results {
		output = append(output, result)
	}
	return output
}

func (wp *WorkerPool[T, R]) RunWithChannelAsync(jobs <-chan T) <-chan R {
	if jobs == nil {
		return nil
	}

	if wp.process == nil {
		return nil
	}

	var wg sync.WaitGroup
	results := make(chan R, len(jobs))
	for i := 1; i <= wp.numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				results <- wp.process(job)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
