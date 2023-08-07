package retry

import (
	"context"
	"errors"
	"testing"
)

func TestRetry(t *testing.T) {
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response data", nil
	}

	reply, err := New()(next)(context.Background(), "request data")
	if err != nil {
		t.Error("Retry: " + err.Error())
	}

	t.Log(reply)
}

func TestRetryError(t *testing.T) {
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Log("next")
		return "response data", errors.New("connection refused")
	}

	reply, err := New()(next)(context.Background(), "request data")
	if err != nil {
		t.Error("Retry: " + err.Error())
		return
	}

	t.Log(reply)
}
