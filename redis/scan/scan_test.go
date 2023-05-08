package scan

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestScan(t *testing.T) {
	addrs := []string{"10.189.6.70:8019", "10.189.6.71:8019"}
	password := "z6KxNbyHxFRjokTc7mzWzHEp7m"
	s := NewScanner(addrs, password, true)
	if err := s.Scan(context.Background(), "air:booking:oi:*", func(client *redis.Client, key string) {
		t.Log(key)
	}); err != nil {
		t.Log(err)
	}
}
