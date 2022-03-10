package scan

import (
	"context"
	"fmt"
	"regexp"
	"time"

	redisinfo "github.com/geoffreybauduin/redis-info"
	"github.com/go-redis/redis/v8"
)

type rdbScanner struct {
	addr     string
	password string
	verbose  bool
}

func NewScanner(addr string, password string, verbose bool) *rdbScanner {
	s := &rdbScanner{
		addr:     addr,
		password: password,
		verbose:  verbose,
	}
	return s
}

func (s *rdbScanner) Scan(match string) error {
	if s.verbose {
		fmt.Printf("Scan addr:%s password:%s verbose:%t pattern:%s\n", s.addr, s.password, s.verbose, match)
	}

	var rdb redis.Cmdable
	rdb = redis.NewClient(&redis.Options{
		Addr:        s.addr,
		PoolSize:    10,
		Password:    s.password,
		DialTimeout: time.Second,
		DB:          0,
	})

	rdbInfo, err := rdb.Info(context.TODO(), "cluster").Result()
	if err != nil {
		return err
	}

	info, err := redisinfo.Parse(rdbInfo)
	if err != nil {
		return err
	}

	var nodes []string
	if info.Cluster.ClusterEnabled {
		resp, err := rdb.ClusterNodes(context.TODO()).Result()
		if err != nil {
			fmt.Println(err)
			return err
		}

		reg, err := regexp.Compile("(((2(5[0-5]|[0-4]\\d))|[0-1]?\\d{1,2})(\\.((2(5[0-5]|[0-4]\\d))|[0-1]?\\d{1,2})){3}:[0-9]*).*master")
		if err != nil {
			return err
		}

		resAll := reg.FindAllStringSubmatch(resp, -1)
		for _, r := range resAll {
			if len(r) < 2 {
				continue
			}
			nodes = append(nodes, r[1])
		}
	} else {
		nodes = append(nodes, s.addr)
	}

	for i, addr := range nodes {
		fmt.Println(i, addr)
		err := s.scan(addr, match)
		if err != nil {
			fmt.Println(err)
		}

	}

	return nil
}

func (s *rdbScanner) scan(addr, match string) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:        addr,
		PoolSize:    10,
		Password:    s.password,
		DialTimeout: time.Second,
		DB:          0,
	})

	_, err := rdb.Ping(context.TODO()).Result()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//查找匹配的key
	var cursor uint64 = 0
	var amount int
	for {
		if s.verbose {
			fmt.Printf("Scan cursor:%d match:%s count:%d\n", cursor, match, 200)
		}
		var keys []string
		keys, cursor, err = rdb.Scan(context.TODO(), cursor, match, 200).Result()
		if err != nil {
			fmt.Println("Scan cursor:%d count:200 err:" + err.Error())
			return err
		}

		if len(keys) > 0 {
			for _, key := range keys {
				amount++
				fmt.Println(key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	fmt.Printf("%s keys amount:%d\n", addr, amount)
	return nil
}
