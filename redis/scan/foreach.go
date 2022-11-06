package scan

import (
	"context"
	"hash/crc32"

	"github.com/go-redis/redis/v8"
)

func foreach() {
	realname := "1014615840@qq.com"
	if crc32.ChecksumIEEE([]byte(realname))%100 < uint32(50) {
		println("ok")
	}
	cli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{},
		Password: "",
	})
	ctx := context.Background()
	// cli.Scan(ctx, 0, "test:test:*", 10000)
	var (
		keys   []string
		cursor uint64
		err    error
	)
	for _, key := range []string{
		"",
		"",
		"",
	} {
		cli.ForEachShard(ctx, func(ctx context.Context, client *redis.Client) error {
			for {
				keys, cursor, err = client.Scan(ctx, 0, key, 1000).Result()
				if err != nil {
					return err
				}
				for _, k := range keys {
					client.Unlink(ctx, k)
				}
				if cursor == 0 {
					return nil
				}
			}
		})
	}
}
