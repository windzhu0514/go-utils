// Package tlstransport 提供可注入 net/http.Client 的 TLS 指纹 Transport。
// uTLS 负责 ClientHello，req.Transport 负责代理、连接池及 HTTP/1.1、HTTP/2 协商。
package tlstransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/imroc/req/v3"
	utls "github.com/refraction-networking/utls"
)

// Transport 实现 http.RoundTripper，可并发复用；必须通过 NewTransport 创建。
// 配置在构造后固定，切换代理或指纹应创建新实例并关闭旧实例的空闲连接。
type Transport struct {
	inner *req.Transport
	cfg   config
	mu    sync.Mutex
	hosts map[string]*req.Transport
}

var _ http.RoundTripper = (*Transport)(nil)

// NewTransport 创建 Transport，默认 Chrome133 Profile、证书校验开启、直连。
// 构造时离线验证指纹与选项，不建立网络连接。
func NewTransport(opts ...Option) (*Transport, error) {
	cfg := config{
		tlsConfig:        &utls.Config{},
		autoDecompress:   true,
		dialTimeout:      10 * time.Second,
		handshakeTimeout: 10 * time.Second,
	}
	if err := WithProfile(Chrome133())(&cfg); err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, errors.New("tlstransport: 选项不能为空")
		}
		if err := opt(&cfg); err != nil {
			return nil, fmt.Errorf("tlstransport: 配置失败: %w", err)
		}
	}
	t := &Transport{cfg: cfg, hosts: make(map[string]*req.Transport)}
	if _, _, err := t.JA3(); err != nil {
		return nil, fmt.Errorf("tlstransport: 指纹校验失败: %w", err)
	}
	t.inner = req.NewTransport()
	t.inner.SetProxy(nil)
	if cfg.proxyURL != nil {
		t.inner.SetProxy(http.ProxyURL(cfg.proxyURL))
	}
	t.inner.SetDial((&net.Dialer{Timeout: cfg.dialTimeout, KeepAlive: 30 * time.Second}).DialContext)
	// 解压在本层处理，统一 HTTP/1.1 与 HTTP/2 的错误及多层编码行为。
	t.inner.DisableCompression = true
	if p := cfg.profile; p != nil {
		t.inner.SetHTTP2SettingsFrame(p.Settings...)
		t.inner.SetHTTP2ConnectionFlow(p.ConnectionFlow)
		t.inner.SetHTTP2HeaderPriority(p.HeaderPriority)
		t.inner.SetHTTP2PriorityFrames(p.PriorityFrames...)
	}
	// 自定义握手自行管理超时，避免 req 的外层定时器提前返回后仍写连接状态。
	t.inner.SetTLSHandshakeTimeout(0)
	t.inner.SetTLSHandshake(t.handshake)
	return t, nil
}

// RoundTrip 执行一次请求；重定向、Cookie 和整次请求超时由上层 Client 管理。
func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil {
		return nil, errors.New("tlstransport: 请求不能为空")
	}
	if request.URL == nil {
		if request.Body != nil {
			request.Body.Close()
		}
		return nil, errors.New("tlstransport: 请求 URL 不能为空")
	}
	r := request.Clone(request.Context())
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if p := t.cfg.profile; p != nil {
		for key, values := range p.Headers {
			if !hasHeader(r.Header, key) {
				r.Header[key] = slices.Clone(values)
			}
		}
		r.Header[req.HeaderOderKey] = completeHeaderOrder(p.HeaderOrder, r.Header)
		r.Header[req.PseudoHeaderOderKey] = slices.Clone(p.PseudoHeaderOrder)
	}
	resp, err := t.transportFor(r.URL.Scheme + "://" + r.URL.Host).RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if t.cfg.autoDecompress && r.Method != http.MethodHead && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		if err := decompress(resp); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (t *Transport) transportFor(origin string) *req.Transport {
	t.mu.Lock()
	defer t.mu.Unlock()
	if tr := t.hosts[origin]; tr != nil {
		return tr
	}
	// req v3.54 在创建 HTTP/2 连接时会改写 MaxHeaderListSize。
	// 按 origin 隔离连接池，避免不同目标的连接初始化并发写同一字段。
	tr := t.inner.Clone()
	t.hosts[origin] = tr
	return tr
}

func completeHeaderOrder(preferred []string, headers http.Header) []string {
	// req v3.54 的排序器要求每个待发送字段都有序号；补齐业务头和自动头，
	// 否则未列出的字段会干扰已指定的相对顺序。
	order := slices.Clone(preferred)
	var extra []string
	for key := range headers {
		key = strings.ToLower(key)
		if key != req.HeaderOderKey && key != req.PseudoHeaderOderKey && !slices.Contains(order, key) {
			extra = append(extra, key)
		}
	}
	for _, key := range []string{"host", "user-agent", "content-length", "transfer-encoding", "connection", "trailer", "proxy-authorization"} {
		if !slices.Contains(order, key) {
			extra = append(extra, key)
		}
	}
	slices.Sort(extra)
	return append(order, slices.Compact(extra)...)
}

func hasHeader(headers http.Header, name string) bool {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}

// CloseIdleConnections 释放空闲连接，不中断正在处理的请求。
func (t *Transport) CloseIdleConnections() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, tr := range t.hosts {
		tr.CloseIdleConnections()
	}
}

func (t *Transport) newConn(plain net.Conn, serverName string) (*tlsConn, error) {
	cfg := t.cfg.tlsConfig.Clone()
	if cfg.ServerName == "" {
		cfg.ServerName = serverName
	}
	spec, err := t.cfg.spec()
	if err != nil {
		return nil, fmt.Errorf("生成 ClientHello: %w", err)
	}
	uconn := utls.UClient(plain, cfg, utls.HelloCustom)
	if err := uconn.ApplyPreset(&spec); err != nil {
		return nil, fmt.Errorf("应用 ClientHello: %w", err)
	}
	conn := &tlsConn{UConn: uconn}
	if p := t.cfg.profile; p != nil {
		for _, setting := range p.Settings {
			if setting.ID == 4 && setting.Val > 4<<20 {
				conn.limitHTTP2Window = true
				conn.filterSettingsACK = true
			}
		}
	}
	return conn, nil
}

func (t *Transport) handshake(ctx context.Context, serverName string, plain net.Conn) (net.Conn, *tls.ConnectionState, error) {
	ctx, cancel := context.WithTimeout(ctx, t.cfg.handshakeTimeout)
	defer cancel()
	conn, err := t.newConn(plain, serverName)
	if err != nil {
		plain.Close()
		return nil, nil, err
	}
	if err := conn.HandshakeContext(ctx); err != nil {
		plain.Close()
		return nil, nil, fmt.Errorf("tlstransport: TLS 握手失败: %w", err)
	}
	state := conn.ConnectionState()
	if state.NegotiatedProtocol != "h2" {
		conn.limitHTTP2Window = false
		conn.filterSettingsACK = false
	}
	if state.NegotiatedProtocol != "" && state.NegotiatedProtocol != "http/1.1" && state.NegotiatedProtocol != "h2" {
		plain.Close()
		return nil, nil, fmt.Errorf("tlstransport: 不支持的 ALPN 协议 %q", state.NegotiatedProtocol)
	}
	return conn, &state, nil
}

// req 的 HTTP/2 transport 通过标准 TLS ConnectionState 接口读取协商结果。
type tlsConn struct {
	*utls.UConn
	writeMu           sync.Mutex
	limitHTTP2Window  bool
	initialWrite      []byte
	readMu            sync.Mutex
	filterSettingsACK bool
	settingsACKs      int
	frameHeader       []byte
	frameRemaining    int
}

func (c *tlsConn) ConnectionState() tls.ConnectionState {
	s := c.UConn.ConnectionState()
	return tls.ConnectionState{
		Version: s.Version, HandshakeComplete: s.HandshakeComplete, DidResume: s.DidResume,
		CipherSuite: s.CipherSuite, NegotiatedProtocol: s.NegotiatedProtocol,
		NegotiatedProtocolIsMutual: s.NegotiatedProtocolIsMutual, ServerName: s.ServerName,
		PeerCertificates: s.PeerCertificates, VerifiedChains: s.VerifiedChains,
		SignedCertificateTimestamps: s.SignedCertificateTimestamps,
		OCSPResponse:                s.OCSPResponse, TLSUnique: s.TLSUnique, ECHAccepted: s.ECHAccepted,
	}
}
