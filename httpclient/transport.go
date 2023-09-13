package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	golangTls "crypto/tls"

	"github.com/pkg/errors"
	tls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

// 包装http1和http2的transport
// 1. 通过底层的transport, 支持链接池管理
// 2. tls握手后自动选择http1.1或http2
// 3. 支持多线程访问
type Transport struct {
	h1Transport             *http.Transport
	mu                      sync.Mutex
	hostToInnerTransportMap map[string]http.RoundTripper // 每个host对应的transport，key: https://studygolang.com
	ProxyAddr               string                       // "socks5://tst:ge5@127.0.0.1:8889" or "http://127.0.0.1:8888"
	dialTimeout             time.Duration                // tcp端口拨号目标主机或者链接代理的超时时间
	handShakeTimeout        time.Duration                // tls握手超时时间
	ClientHelloSpec         *tls.ClientHelloSpec         // 仅当clientHelloID为HelloCustom时有用
	ClientHelloID           tls.ClientHelloID

	*Debug // 用于调试
}

type TlsConnOption func(*Transport)

func WithTcpDialTimeout(dialTimeout time.Duration) TlsConnOption {
	return func(tr *Transport) {
		tr.dialTimeout = dialTimeout
	}
}

func WithHandShakeTimeout(handShakeTimeout time.Duration) TlsConnOption {
	return func(tr *Transport) {
		tr.handShakeTimeout = handShakeTimeout
	}
}

func WithTlsConnOptProxyAddr(proxyAddr string) TlsConnOption {
	return func(tr *Transport) {
		tr.ProxyAddr = proxyAddr
	}
}

func WithTlsConnOptClientHelloID(id tls.ClientHelloID) TlsConnOption {
	return func(tr *Transport) {
		tr.ClientHelloID = id
	}
}

// 自定义tls指纹： HelloCustom + ClientHelloSpec
func WithTlsConnOptClientHelloSpec(spec *tls.ClientHelloSpec) TlsConnOption {
	return func(tr *Transport) {
		tr.ClientHelloSpec = spec
	}
}

func NewTransport(opts ...TlsConnOption) *Transport {
	tr := &Transport{
		h1Transport: &http.Transport{
			DisableKeepAlives:     false,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		hostToInnerTransportMap: make(map[string]http.RoundTripper),
		ProxyAddr:               "",
		dialTimeout:             10 * time.Second,
		handShakeTimeout:        10 * time.Second,
		ClientHelloSpec:         nil,
		ClientHelloID:           tls.HelloChrome_62,
		Debug: &Debug{
			TlsConnCount:     make(map[string]int64),
			TlsHandShakeTime: make(map[string]int64),
		},
	}
	if len(opts) != 0 {
		for _, opt := range opts {
			opt(tr)
		}
	}
	return tr
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	if req.URL == nil {
		return nil, errors.New("http: nil Request.URL")
	}

	scheme := req.URL.Scheme
	host := req.URL.Host
	connectionKey := fmt.Sprintf("%s://%s", scheme, host)
	// fmt.Println("connectionKey: ", connectionKey)
	innerTransport, err := t.initOrGetCacheTransport(connectionKey, req.URL)
	if err != nil {
		return resp, err
	}
	resp, err = innerTransport.RoundTrip(req)
	if err != nil {
		t.deleteHostToInnerTransport(connectionKey)
		return resp, err
	}

	return resp, nil
}

func (t *Transport) initOrGetCacheTransport(cKey string, u *url.URL) (http.RoundTripper, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var err error
	innerTransport, ok := t.hostToInnerTransportMap[cKey]
	if !ok {
		innerTransport, err = t.getTransport(u)
		if err != nil {
			return nil, err
		}
		t.hostToInnerTransportMap[cKey] = innerTransport
	}
	return innerTransport, nil
}

func (t *Transport) getTransport(u *url.URL) (http.RoundTripper, error) {
	var (
		err       error
		tlsConn   *tls.UConn
		transport http.RoundTripper
	)
	if u.Scheme == "http" {
		// http请求统一都使用http1.1
		transport = t.h1Transport.Clone()
		if t.ProxyAddr != "" {
			transport.(*http.Transport).Proxy = func(r *http.Request) (*url.URL, error) {
				u, err := url.Parse(t.ProxyAddr)
				if err != nil {
					return nil, err
				}
				return u, nil
			}
		}
		return transport, nil
	}

	// 先用tls握一次手用于后面判断使用http2还是http1
	tlsConn, err = t.getTlsConn(u.Host)
	if err != nil {
		return nil, err
	}
	connectionState := tlsConn.ConnectionState()
	if !connectionState.HandshakeComplete {
		return nil, errors.New("handshake is not complete")
	}
	return t.obtainInnerTransport(tlsConn, connectionState.NegotiatedProtocol)
}

func (t *Transport) deleteHostToInnerTransport(connectionKey string) {
	t.mu.Lock()
	delete(t.hostToInnerTransportMap, connectionKey)
	t.mu.Unlock()
}

// 自定义的client_hello_spec有些extention不能重复使用
func (t *Transport) deepCopyClientHelloSpec(dst *tls.ClientHelloSpec, src *tls.ClientHelloSpec) error {
	dst.TLSVersMax = src.TLSVersMax
	dst.TLSVersMin = src.TLSVersMin
	dst.CipherSuites = make([]uint16, len(src.CipherSuites))
	copy(dst.CipherSuites, src.CipherSuites)
	dst.CompressionMethods = make([]uint8, len(src.CompressionMethods))
	copy(dst.CompressionMethods, src.CompressionMethods)
	for _, extention := range src.Extensions {
		var copyExt tls.TLSExtension
		switch ext := extention.(type) {
		case *tls.UtlsPaddingExtension:
			copyExt = &tls.UtlsPaddingExtension{
				GetPaddingLen: ext.GetPaddingLen,
			}
		case *tls.UtlsGREASEExtension:
			copyExt = &tls.UtlsGREASEExtension{}
		case *tls.SNIExtension:
			copyExt = &tls.SNIExtension{}
		case *tls.SessionTicketExtension:
			copyExt = &tls.SessionTicketExtension{}
		case *tls.KeyShareExtension:
			ksExt := &tls.KeyShareExtension{}
			for _, ks := range ext.KeyShares {
				ksShare := tls.KeyShare{Group: ks.Group}
				if isGREASEUint16(uint16(ks.Group)) {
					ksShare.Data = []byte{0}
				}
				ksExt.KeyShares = append(ksExt.KeyShares, ksShare)
			}
			copyExt = ksExt
		default:
			copyExt = extention
		}
		dst.Extensions = append(dst.Extensions, copyExt)
	}
	dst.GetSessionID = src.GetSessionID
	return nil
}

func (t *Transport) getProxyDialer(proxyAddr string) (proxy.Dialer, error) {
	if proxyAddr == "" {
		return nil, errors.New("proxyAddr can't be empty")
	}

	var proxyDialer proxy.Dialer
	proxyURI, err := url.Parse(t.ProxyAddr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: t.dialTimeout}
	switch proxyURI.Scheme {
	case "socks5":
		password, _ := proxyURI.User.Password()
		auth := proxy.Auth{
			User:     proxyURI.User.Username(),
			Password: password,
		}
		proxyDialer, err = proxy.SOCKS5("tcp", proxyURI.Host, &auth, dialer)
	case "http":
		proxyDialer, err = NewConnectproxy(proxyURI, dialer)
	}
	if err != nil {
		return nil, err
	}

	return proxyDialer, nil
}

func (t *Transport) getTlsConn(host string) (*tls.UConn, error) {
	host = strings.ReplaceAll(host, ":443", "") // host必须是域名，不然不能验证证书
	t.Debug.IncrTlsConnCount(host)
	var (
		dialConn net.Conn
		err      error
		addr     = host + ":443"
	)

	if t.ProxyAddr != "" {
		var proxyDialer proxy.Dialer
		proxyDialer, err = t.getProxyDialer(t.ProxyAddr)
		if err != nil {
			return nil, errors.Wrap(err, "getProxyDialer")
		}
		dialConn, err = proxyDialer.Dial("tcp", addr)
		if err != nil {
			return nil, errors.Wrap(err, "proxyDialer.Dial")
		}
	} else {
		dialConn, err = net.DialTimeout("tcp", addr, t.dialTimeout)
		if err != nil {
			return nil, errors.Wrap(err, "net.DialTimeout")
		}
	}

	defaultConfig := tls.Config{ServerName: host}
	tlsConn := tls.UClient(dialConn, &defaultConfig, t.ClientHelloID)
	if t.ClientHelloID == tls.HelloCustom {
		if t.ClientHelloSpec == nil {
			return nil, errors.New("HelloCustom but clientHelloSpec is nil")
		}
		copySpec := new(tls.ClientHelloSpec)
		err = t.deepCopyClientHelloSpec(copySpec, t.ClientHelloSpec)
		if err != nil {
			return nil, err
		}
		err := tlsConn.ApplyPreset(copySpec)
		if err != nil {
			return nil, err
		}
	}
	ctx, cancel := context.WithTimeout(context.TODO(), t.handShakeTimeout)
	defer cancel()
	tlsHandStartTime := time.Now().UnixMilli()
	err = tlsConn.HandshakeContext(ctx)
	tlsHandEndTime := time.Now().UnixMilli()
	cost := tlsHandEndTime - tlsHandStartTime
	t.Debug.SetTlsHandShakeTime(host, cost)
	if err != nil {
		fmt.Println("Handshake error: ", err)
		return nil, errors.Wrap(err, "HandshakeContextError")
	}
	return tlsConn, nil
}

func (t *Transport) obtainInnerTransport(tlsConn *tls.UConn, applicatipnProtocol string) (http.RoundTripper, error) {
	switch applicatipnProtocol {
	case "h2":
		tr := &http2.Transport{
			PingTimeout: 30 * time.Second,
		}
		tr.DialTLSContext = func(ctx context.Context, network, addr string, cfg *golangTls.Config) (net.Conn, error) {
			// 这里用个骚操作，第一次握手后的tls链接可以直接使用
			tempTlsConn := tlsConn
			if tlsConn != nil {
				tlsConn = nil
				return tempTlsConn, nil
			}
			// 重新建立链接
			dialTlsConn, err := t.getTlsConn(addr)
			if err != nil {
				return nil, err
			}
			return dialTlsConn, nil
		}
		return tr, nil
	case "http/1.1", "":
		tr := t.h1Transport.Clone()
		tr.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 这里用个骚操作，第一次握手后的tls链接可以直接使用
			tempTlsConn := tlsConn
			if tlsConn != nil {
				tlsConn = nil
				return tempTlsConn, nil
			}
			// 重新建立链接
			dialTlsConn, err := t.getTlsConn(addr)
			if err != nil {
				return nil, err
			}
			return dialTlsConn, nil
		}
		return tr, nil
	default:
		return nil, fmt.Errorf("unkown protocol: %s", applicatipnProtocol)
	}
}

// 设置http1.1的transport模板，否则使用默认
func (t *Transport) SetH1Transport(h1T *http.Transport) {
	t.h1Transport = h1T
}
