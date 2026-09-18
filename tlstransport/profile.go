package tlstransport

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"

	"github.com/imroc/req/v3/http2"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http/httpguts"
	xhttp2 "golang.org/x/net/http2"
)

// Profile 将 TLS、HTTP/2 和默认请求头绑定为一个浏览器配置。
// Headers 只应用于请求中未设置的字段；业务相关的 Cookie、认证和 Fetch Metadata 应由请求设置。
// Settings、PriorityFrames 和顺序切片均按给定顺序发送，SpecFactory 优先于 ClientHelloID。
type Profile struct {
	Name              string
	ClientHelloID     utls.ClientHelloID
	SpecFactory       SpecFactory
	Settings          []http2.Setting
	ConnectionFlow    uint32
	HeaderPriority    http2.PriorityParam
	PriorityFrames    []http2.PriorityFrame
	PseudoHeaderOrder []string
	HeaderOrder       []string
	Headers           http.Header
}

// WithProfile 同时设置 TLS 和 HTTP 指纹。数据在构造时复制，工厂仍须并发安全。
func WithProfile(profile Profile) Option {
	return func(c *config) error {
		p := profile.clone()
		if err := p.validate(); err != nil {
			return err
		}
		var err error
		if p.SpecFactory != nil {
			err = WithClientHelloSpec(p.SpecFactory)(c)
		} else {
			err = WithClientHelloID(p.ClientHelloID)(c)
		}
		if err != nil {
			return err
		}
		c.profile = &p
		return nil
	}
}

// WithRandomProfile 在创建时随机选择完整 Profile，使 TLS、UA、HTTP/2 保持配套。
func WithRandomProfile(profiles ...Profile) Option {
	profiles = slices.Clone(profiles)
	if len(profiles) == 0 {
		profiles = []Profile{Chrome133(), Firefox120(), Safari160()}
	}
	return func(c *config) error {
		for _, p := range profiles {
			var candidate config
			if err := WithProfile(p)(&candidate); err != nil {
				return err
			}
		}
		return WithProfile(profiles[rand.IntN(len(profiles))])(c)
	}
}

func (p Profile) clone() Profile {
	p.Settings = slices.Clone(p.Settings)
	p.PriorityFrames = slices.Clone(p.PriorityFrames)
	p.PseudoHeaderOrder = slices.Clone(p.PseudoHeaderOrder)
	p.HeaderOrder = slices.Clone(p.HeaderOrder)
	p.Headers = p.Headers.Clone()
	return p
}

func (p Profile) validate() error {
	seen := make(map[http2.SettingID]bool)
	for _, setting := range p.Settings {
		if seen[setting.ID] {
			return errors.New("tlstransport: HTTP/2 SETTINGS 不能重复")
		}
		seen[setting.ID] = true
		if err := (xhttp2.Setting{ID: xhttp2.SettingID(setting.ID), Val: setting.Val}).Valid(); err != nil {
			return fmt.Errorf("tlstransport: HTTP/2 SETTINGS 无效: %w", err)
		}
	}
	// req 用 int32 表示接收窗口；保留协议初始窗口的空间。
	if p.ConnectionFlow > (1<<31-1)-65535 || p.HeaderPriority.StreamDep > 1<<31-1 {
		return errors.New("tlstransport: HTTP/2 窗口或流依赖超出范围")
	}
	var previous uint32
	for _, frame := range p.PriorityFrames {
		if frame.StreamID <= previous || frame.StreamID%2 == 0 || frame.StreamID > (1<<31)-3 ||
			frame.PriorityParam.StreamDep > 1<<31-1 || frame.StreamID == frame.PriorityParam.StreamDep {
			return errors.New("tlstransport: PRIORITY 帧必须使用递增的有效奇数流 ID")
		}
		previous = frame.StreamID
	}
	if len(p.PseudoHeaderOrder) != 0 {
		order := slices.Clone(p.PseudoHeaderOrder)
		slices.Sort(order)
		if !slices.Equal(order, []string{":authority", ":method", ":path", ":scheme"}) {
			return errors.New("tlstransport: 伪头顺序必须恰好包含四个 HTTP/2 请求伪头")
		}
	}
	seenHeaders := make(map[string]bool)
	for i, name := range p.HeaderOrder {
		name = strings.ToLower(name)
		if !httpguts.ValidHeaderFieldName(name) || seenHeaders[name] {
			return errors.New("tlstransport: 请求头顺序包含无效或重复字段")
		}
		seenHeaders[name] = true
		p.HeaderOrder[i] = name
	}
	for name, values := range p.Headers {
		if !httpguts.ValidHeaderFieldName(name) {
			return errors.New("tlstransport: 默认请求头名称无效")
		}
		switch strings.ToLower(name) {
		case "authorization", "proxy-authorization", "cookie", "host", "content-length", "transfer-encoding":
			return errors.New("tlstransport: 认证、Cookie 和请求路由字段不能放入 Profile")
		}
		for _, value := range values {
			if !httpguts.ValidHeaderFieldValue(value) {
				return errors.New("tlstransport: 默认请求头值无效")
			}
		}
	}
	return nil
}

// Chrome133 返回固定版本的 Chrome 配置；TLS 模板来自 uTLS，HTTP/2 参数参考 tls-client。
func Chrome133() Profile {
	return Profile{
		Name: "chrome_133", ClientHelloID: utls.HelloChrome_133,
		Settings:          []http2.Setting{{ID: 1, Val: 65536}, {ID: 2, Val: 0}, {ID: 4, Val: 6291456}, {ID: 6, Val: 262144}},
		ConnectionFlow:    15663105,
		PseudoHeaderOrder: []string{":method", ":authority", ":scheme", ":path"},
		HeaderOrder: []string{"host", "sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "user-agent", "accept",
			"sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest", "referer", "accept-encoding", "accept-language", "cookie", "priority"},
		Headers: http.Header{
			"User-Agent":       {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"},
			"Sec-Ch-Ua":        {`"Not(A:Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`},
			"Sec-Ch-Ua-Mobile": {"?0"}, "Sec-Ch-Ua-Platform": {`"macOS"`},
			"Accept": {"*/*"}, "Accept-Encoding": {"gzip, deflate, br, zstd"},
		},
	}
}

// Firefox120 返回 Firefox 120 的 TLS、HTTP/2 优先级树和请求头配置。
func Firefox120() Profile {
	return Profile{
		Name: "firefox_120", ClientHelloID: utls.HelloFirefox_120,
		Settings:       []http2.Setting{{ID: 1, Val: 65536}, {ID: 4, Val: 131072}, {ID: 5, Val: 16384}},
		ConnectionFlow: 12517377,
		HeaderPriority: http2.PriorityParam{StreamDep: 13, Weight: 41},
		PriorityFrames: []http2.PriorityFrame{
			{StreamID: 3, PriorityParam: http2.PriorityParam{Weight: 200}},
			{StreamID: 5, PriorityParam: http2.PriorityParam{Weight: 100}},
			{StreamID: 7}, {StreamID: 9, PriorityParam: http2.PriorityParam{StreamDep: 7}},
			{StreamID: 11, PriorityParam: http2.PriorityParam{StreamDep: 3}},
			{StreamID: 13, PriorityParam: http2.PriorityParam{Weight: 240}},
		},
		PseudoHeaderOrder: []string{":method", ":path", ":authority", ":scheme"},
		HeaderOrder:       []string{"host", "user-agent", "accept", "accept-language", "accept-encoding", "referer", "cookie", "sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site", "te"},
		Headers: http.Header{
			"User-Agent": {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0"},
			"Accept":     {"*/*"}, "Accept-Encoding": {"gzip, deflate, br"},
		},
	}
}

// Safari160 返回 Safari 16.0 的 TLS、HTTP/2 和请求头配置。
func Safari160() Profile {
	return Profile{
		Name: "safari_16_0", ClientHelloID: utls.HelloSafari_16_0,
		Settings:          []http2.Setting{{ID: 4, Val: 4194304}, {ID: 3, Val: 100}},
		ConnectionFlow:    10485760,
		PseudoHeaderOrder: []string{":method", ":scheme", ":path", ":authority"},
		HeaderOrder:       []string{"host", "accept", "sec-fetch-site", "cookie", "sec-fetch-dest", "accept-language", "sec-fetch-mode", "user-agent", "referer", "accept-encoding"},
		Headers: http.Header{
			"User-Agent": {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Safari/605.1.15"},
			"Accept":     {"*/*"}, "Accept-Encoding": {"gzip, deflate, br"},
		},
	}
}
