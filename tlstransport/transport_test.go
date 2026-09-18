package tlstransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	utls "github.com/refraction-networking/utls"
)

func newTestTransport(t *testing.T, opts ...Option) *Transport {
	t.Helper()
	tr, err := NewTransport(opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.CloseIdleConnections)
	return tr
}

func testTLSConfig(server *httptest.Server) *utls.Config {
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	return &utls.Config{RootCAs: roots}
}

func readReply(resp *http.Response, err error) error {
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		return fmt.Errorf("响应异常: %s %q", resp.Status, body)
	}
	return nil
}

func TestUnitTransport_ProtocolsAndReuse(t *testing.T) {
	for _, profile := range []utls.ClientHelloID{utls.HelloChrome_Auto, utls.HelloFirefox_Auto, utls.HelloSafari_Auto} {
		for _, h2 := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/h2=%v", profile.Str(), h2), func(t *testing.T) {
				var hellos atomic.Int32
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.Header.Get("X-Test") != "preserved" {
						t.Error("请求方法或请求头未保留")
					}
					body, err := io.ReadAll(r.Body)
					if err != nil || string(body) != "payload" {
						t.Errorf("请求体错误: %q %v", body, err)
					}
					io.WriteString(w, "ok")
				}))
				server.EnableHTTP2 = h2
				server.TLS = &tls.Config{GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
					hellos.Add(1)
					if hello.ServerName != "example.com" || len(hello.CipherSuites) < 5 {
						t.Errorf("ClientHello 异常: SNI=%q, ciphers=%v", hello.ServerName, hello.CipherSuites)
					}
					return nil, nil
				}}
				server.StartTLS()
				defer server.Close()
				cfg := testTLSConfig(server)
				cfg.ServerName = "example.com"
				tr := newTestTransport(t, WithClientHelloID(profile), WithTLSConfig(cfg))
				// 构造后修改调用者的值字段，不应污染已创建的 Transport。
				cfg.ServerName = "invalid.example"
				client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
				for i := range 3 {
					var reused bool
					request, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("payload"))
					if err != nil {
						t.Fatal(err)
					}
					request.Header.Set("X-Test", "preserved")
					request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{
						GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused },
					}))
					resp, err := client.Do(request)
					if err != nil {
						t.Fatal(err)
					}
					wantProto := 1
					if h2 {
						wantProto = 2
					}
					if resp.ProtoMajor != wantProto || resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 {
						t.Errorf("协议或证书状态异常: %s %+v", resp.Proto, resp.TLS)
					}
					if err := readReply(resp, nil); err != nil {
						t.Fatal(err)
					}
					if i == 1 && !reused {
						t.Error("第二次请求没有复用连接")
					}
					if i == 1 {
						tr.CloseIdleConnections()
					}
				}
				if hellos.Load() != 2 {
					t.Errorf("预期两次握手（含关闭空闲连接后的重连），实际 %d", hellos.Load())
				}
			})
		}
	}
}

func TestUnitTransport_CertificateValidation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
	defer server.Close()
	for _, wrongHost := range []bool{false, true} {
		var opts []Option
		if wrongHost {
			cfg := testTLSConfig(server)
			cfg.ServerName = "invalid.example"
			opts = append(opts, WithTLSConfig(cfg))
		}
		client := &http.Client{Transport: newTestTransport(t, opts...), Timeout: 3 * time.Second}
		resp, err := client.Get(server.URL)
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Fatal("未拒绝不可信证书或错误主机名")
		}
		var unknown x509.UnknownAuthorityError
		var hostname x509.HostnameError
		if wrongHost && !errors.As(err, &hostname) || !wrongHost && !errors.As(err, &unknown) {
			t.Errorf("证书错误类型未保留: %v", err)
		}
	}
}

func TestUnitTransport_ConcurrentCustomSpec(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		t.Run(fmt.Sprint(h2), func(t *testing.T) {
			var calls atomic.Int32
			factory := func() (utls.ClientHelloSpec, error) {
				calls.Add(1)
				return utls.UTLSIdToSpec(utls.HelloChrome_100)
			}
			var servers []*httptest.Server
			roots := x509.NewCertPool()
			for range 2 {
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
				server.EnableHTTP2 = h2
				server.StartTLS()
				defer server.Close()
				servers = append(servers, server)
				roots.AddCert(server.Certificate())
			}
			profile := Chrome133()
			profile.SpecFactory = factory
			tr := newTestTransport(t, WithProfile(profile), WithTLSConfig(&utls.Config{RootCAs: roots}))
			client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
			var wg sync.WaitGroup
			for i := range 16 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if _, _, err := tr.JA3(); err != nil {
						t.Error(err)
					}
					if err := readReply(client.Get(servers[i%len(servers)].URL)); err != nil {
						t.Error(err)
					}
				}()
			}
			wg.Wait()
			if calls.Load() < 19 { // 构造、16 次预览以及至少两个目标连接。
				t.Errorf("指纹工厂未逐次调用: %d", calls.Load())
			}
		})
	}
}

func TestUnitTransport_TimeoutAndCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var wg sync.WaitGroup
	accepted := make(chan struct{}, 4)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(3 * time.Second))
				accepted <- struct{}{}
				io.Copy(io.Discard, conn) // 接收 ClientHello，但永不返回 ServerHello。
			}()
		}
	}()
	tr := newTestTransport(t, WithTLSHandshakeTimeout(100*time.Millisecond))
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	if err := readReply(client.Get("https://" + listener.Addr().String())); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("握手超时未保留 DeadlineExceeded: %v", err)
	}
	<-accepted
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+listener.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- readReply(client.Do(request)) }()
	<-accepted
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("取消错误未保留: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("取消未及时返回")
	}
	wg.Wait()
}

func TestUnitTransport_HTTPClientAndResty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "test", Path: "/"})
			http.Redirect(w, r, "/end", http.StatusFound)
			return
		}
		if cookie, err := r.Cookie("session"); err != nil || cookie.Value != "test" {
			http.Error(w, "missing cookie", http.StatusBadRequest)
			return
		}
		io.WriteString(w, "ok")
	}))
	defer server.Close()
	// 默认直连不受环境代理干扰。
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	tr := newTestTransport(t)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: tr, Jar: jar, Timeout: 2 * time.Second}
	if err := readReply(client.Get(server.URL + "/start")); err != nil {
		t.Fatal(err)
	}
	res, err := resty.New().SetTransport(tr).SetTimeout(2 * time.Second).R().Get(server.URL + "/start")
	if err != nil || res.StatusCode() != 200 || res.String() != "ok" {
		t.Fatalf("Resty 注入失败: %v %v", res, err)
	}
}

func TestUnitTransport_InvalidOptions(t *testing.T) {
	for _, opt := range []Option{
		nil, WithClientHelloID(utls.HelloCustom), WithClientHelloSpec(nil), WithTLSConfig(nil),
		WithTCPDialTimeout(0), WithTLSHandshakeTimeout(-time.Second), WithRandomBrowser(utls.HelloCustom),
		WithProxyURL("localhost:8080"), WithProxyURL("https://localhost:8080"), WithProxyURL("ftp://localhost"),
		WithProxyURL("http://localhost:70000"), WithProxyURL("http://localhost/path"),
		WithProxyURL("http://test:secret@localhost:bad"),
		WithClientHelloSpec(func() (utls.ClientHelloSpec, error) { return utls.ClientHelloSpec{}, errors.New("factory failure") }),
	} {
		tr, err := NewTransport(opt)
		if err == nil || tr != nil {
			t.Fatalf("无效配置未被拒绝: %v %v", tr, err)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatal("错误泄露代理凭据")
		}
	}
	newTestTransport(t, WithRandomBrowser())
}
