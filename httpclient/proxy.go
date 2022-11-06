package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type ProxySelector interface {
	ProxyFunc(req *http.Request) (*url.URL, error)
	ProxyInvalid(ctx context.Context)
}

type ProxyURLSelector struct {
	proxy        *url.URL
	proxyInvalid func(ctx context.Context)
}

func NewProxyURLSelector(proxy *url.URL, proxyInvalid func(ctx context.Context)) (*ProxyURLSelector, error) {
	return &ProxyURLSelector{proxy: proxy, proxyInvalid: proxyInvalid}, nil
}

func (s *ProxyURLSelector) ProxyFunc(req *http.Request) (*url.URL, error) {
	return s.proxy, nil
}

func (s *ProxyURLSelector) ProxyInvalid(ctx context.Context) {
	s.proxyInvalid(ctx)
}

// HostnameProxySelector 特定 Hostname 使用特定代理
type HostnameProxySelector struct {
	proxys       map[string]*url.URL
	proxyInvalid func(ctx context.Context)
}

func NewHostNameProxySelector(proxyInvalid func(ctx context.Context)) *HostnameProxySelector {
	return &HostnameProxySelector{proxys: make(map[string]*url.URL), proxyInvalid: proxyInvalid}
}

func (s *HostnameProxySelector) SetProxyURL(proxy *url.URL, urls ...string) error {
	if proxy == nil {
		return errors.New("proxy is nil")
	}

	for _, rawURL := range urls {
		URL, err := url.Parse(rawURL)
		if err != nil {
			return err
		}

		s.proxys[URL.Hostname()] = proxy
	}

	return nil
}

// ProxyFunc 实现ProxySelector接口
func (s *HostnameProxySelector) ProxyFunc(req *http.Request) (*url.URL, error) {
	if req == nil || req.URL == nil || len(s.proxys) == 0 {
		return nil, nil
	}

	proxy, ok := s.proxys[req.URL.Hostname()]
	if !ok {
		return nil, nil
	}

	return proxy, nil
}

func (s *HostnameProxySelector) ProxyInvalid(ctx context.Context) {
	s.proxyInvalid(ctx)
}
