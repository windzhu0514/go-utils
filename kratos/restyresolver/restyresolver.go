// 基于 kratos 服务发现的 resty 中间件
package restyresolver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/random"
	"github.com/go-resty/resty/v2"
)

type ServerConfig struct {
	Name      string        `json:"name,omitempty"`
	Addr      string        `json:"addr,omitempty"`
	AuthUser  string        `json:"auth_user,omitempty"`
	PartnerId string        `json:"partner_id,omitempty"`
	SignKey   string        `json:"sign_key,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	Prefix    string        `json:"prefix,omitempty"`  // etcd prefix
	Service   string        `json:"service,omitempty"` // etcd service name
}

type Resolver struct {
	discovery registry.Discovery
	targets   []ServerConfig
	selectors map[string]selector.Selector
	watcher   registry.Watcher
	logger    *log.Helper
}

func New(ctx context.Context, logger log.Logger, discovery registry.Discovery, serverCfgs []ServerConfig, block bool) (*Resolver, error) {
	r := &Resolver{
		discovery: discovery,
		targets:   serverCfgs,
		selectors: make(map[string]selector.Selector),
		logger:    log.NewHelper(logger, log.WithMessageKey("message")),
	}

	for _, cfg := range serverCfgs {
		if cfg.Addr == "" {
			return nil, fmt.Errorf("target:%s addr is empty", cfg.Name)
		}

		target, err := parseTarget(cfg.Addr, true)
		if err != nil {
			return nil, err
		}

		r.selectors[target.Endpoint] = random.New()
		watcher, err := discovery.Watch(ctx, target.Endpoint)
		if err != nil {
			return nil, err
		}

		if block {
			done := make(chan error, 1)
			go func() {
				services, err := watcher.Next()
				if err != nil {
					done <- fmt.Errorf("http client watch service:%v got unexpected error:%s", target, err.Error())
					return
				}

				if len(services) == 0 {
					done <- fmt.Errorf("http client:%v watch service got empty", target)
					return
				}

				nodes := make([]selector.Node, 0)
				for _, ins := range services {
					nodes = append(nodes, selector.NewNode(target.Scheme, ins.Endpoints[0], ins))
				}

				r.selectors[target.Endpoint].Apply(nodes)
			}()

			select {
			case err := <-done:
				if err != nil {
					err := watcher.Stop()
					if err != nil {
						r.logger.Errorf("failed to http client watch stop:%v error:%s", target, err.Error())
					}
					return nil, err
				}
			case <-ctx.Done():
				r.logger.Errorf("http client watch service %v reaching context deadline!", target)
				err := watcher.Stop()
				if err != nil {
					r.logger.Errorf("failed to http client watch stop: %v", target)
				}
				return nil, ctx.Err()
			}
		}

		go func(serverCfgs ServerConfig) {
			for {
				services, err := watcher.Next()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					r.logger.Errorf("http client watch service:%s got unexpected error:%s", target.Endpoint, err.Error())
					time.Sleep(time.Second)
					continue
				}

				// 服务结点为0 不更新原来的Selector 暂时缓存一下
				if len(services) == 0 {
					r.logger.Errorf("http client:%s watch service got empty", serverCfgs.Name)
					continue
				}

				nodes := make([]selector.Node, 0)
				for _, ins := range services {
					nodes = append(nodes, selector.NewNode(target.Scheme, ins.Endpoints[0], ins))
				}

				r.selectors[target.Endpoint].Apply(nodes)
			}
		}(cfg)
	}

	return r, nil
}

func (r *Resolver) OnBeforeRequest(client *resty.Client, request *resty.Request) error {
	target, err := parseTarget(request.URL, true)
	if err != nil {
		return err
	}

	var (
		done func(context.Context, selector.DoneInfo)
		node selector.Node
	)

	for endpoint := range r.selectors {
		if strings.HasPrefix(target.Endpoint, endpoint) {
			target.Endpoint = strings.TrimPrefix(target.Endpoint, endpoint)
			s, ok := r.selectors[endpoint]
			if !ok {
				continue
			}
			node, done, err = s.Select(context.Background())
			if err != nil {
				return err
			}
			break
		}
	}

	if node == nil {
		return fmt.Errorf("%s:http client select node is nil", request.URL)
	}

	request.URL = node.Address() + target.Endpoint
	if done != nil {
		done(context.Background(), selector.DoneInfo{Err: err})
	}
	return nil
}

func (r *Resolver) Close() error {
	return r.watcher.Stop()
}

type Target struct {
	Scheme    string
	Authority string
	Endpoint  string
}

func parseTarget(endpoint string, insecure bool) (*Target, error) {
	if !strings.Contains(endpoint, "://") {
		if insecure {
			endpoint = "http://" + endpoint
		} else {
			endpoint = "https://" + endpoint
		}
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	target := &Target{Scheme: u.Scheme, Authority: u.Host}
	if len(u.Path) > 1 {
		target.Endpoint = u.Path[1:]
	}
	return target, nil
}
