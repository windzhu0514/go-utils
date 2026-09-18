package workpool

import (
	"fmt"
	"testing"
	"time"
)

func TestWorkerPool_Run(t *testing.T) {
	wp := New(10, func(item int) int {
		time.Sleep(time.Second * time.Duration(item))
		return item * 2
	})

	results := wp.Run([]int{1, 2, 3, 4, 5})

	fmt.Println(results)
}

func TestWorkerPool_RunAsync(t *testing.T) {
	wp := New(10, func(item int) int {
		time.Sleep(time.Second * time.Duration(item))
		return item * 2
	})

	results := wp.RunAsync([]int{1, 2, 3, 4, 5})

	for result := range results {
		fmt.Println(result)
	}
}

func TestWorkerPool_RunWithChannel(t *testing.T) {
	wp := New(10, func(item int) int {
		time.Sleep(time.Second * time.Duration(item))
		return item * 2
	})

	items := []int{1, 2, 3, 4, 5}

	ch := make(chan int, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)

	results := wp.RunWithChannel(ch)

	fmt.Println(results)
}

func TestWorkerPool_RunWithChannelAsync(t *testing.T) {
	wp := New(10, func(item int) int {
		time.Sleep(time.Second * time.Duration(item))
		return item * 2
	})

	items := []int{1, 2, 3, 4, 5}

	ch := make(chan int, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)

	results := wp.RunWithChannelAsync(ch)

	for result := range results {
		fmt.Println(result)
	}
}
