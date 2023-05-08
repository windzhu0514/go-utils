package scan

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type rdbScanner struct {
	addrs    []string
	password string
	verbose  bool
}

func NewScanner(addrs []string, password string, verbose bool) *rdbScanner {
	s := &rdbScanner{
		addrs:    addrs,
		password: password,
		verbose:  verbose,
	}
	return s
}

func (s *rdbScanner) Scan(ctx context.Context, match string, handler func(client *redis.Client, key string)) error {
	if s.verbose {
		log.Printf("Scan addr:%v password:%s pattern:%s\n", s.addrs, s.password, match)
	}

	if len(s.addrs) == 0 {
		return errors.New("redis addrs is empty")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:        s.addrs[0],
		PoolSize:    10,
		Password:    s.password,
		DialTimeout: time.Second,
	})

	var totalAmount int64
	if len(s.addrs) > 1 {
		rdbCluster := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:       s.addrs,
			PoolSize:    10,
			Password:    s.password,
			DialTimeout: 3 * time.Second,
		})

		clusterInfo, err := rdb.ClusterInfo(context.TODO()).Result()
		if err != nil {
			return err
		}

		log.Println("\n" + clusterInfo)

		err = rdbCluster.ForEachMaster(ctx, func(ctx context.Context, rdb *redis.Client) error {
			iter := rdb.Scan(ctx, 0, match, 10000).Iterator()
			for iter.Next(ctx) {
				totalAmount++

				key := iter.Val()
				handler(rdb, key)
			}
			return iter.Err()
		})

		log.Printf("scan over,total keys: %d", totalAmount)

		return err
	}

	iter := rdb.Scan(ctx, 0, match, 10000).Iterator()
	for iter.Next(ctx) {
		totalAmount++

		key := iter.Val()
		handler(rdb, key)
	}

	log.Printf("scan over,total keys: %d", totalAmount)

	return iter.Err()
}
