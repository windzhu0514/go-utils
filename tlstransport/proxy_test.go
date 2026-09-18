package tlstransport

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func relayTestTunnel(client, upstream net.Conn) {
	defer client.Close()
	defer upstream.Close()
	done := make(chan struct{})
	go func() {
		io.Copy(upstream, client)
		upstream.Close()
		close(done)
	}()
	io.Copy(client, upstream)
	client.Close()
	<-done
}

func TestUnitProxy_HTTPAndCONNECT(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(fmt.Sprint(secure), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Proxy-Authorization") != "" {
					t.Error("代理认证泄露到目标服务器")
				}
				io.WriteString(w, "ok")
			})
			target := httptest.NewUnstartedServer(handler)
			target.EnableHTTP2 = secure
			if secure {
				target.StartTLS()
			} else {
				target.Start()
			}
			defer target.Close()
			upstream := &http.Transport{}
			defer upstream.CloseIdleConnections()
			var requests atomic.Int32
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Header.Get("Proxy-Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte("user:pass")) {
					http.Error(w, "proxy auth required", http.StatusProxyAuthRequired)
					return
				}
				if r.Host != strings.TrimPrefix(strings.TrimPrefix(target.URL, "https://"), "http://") {
					http.Error(w, "wrong target", http.StatusBadRequest)
					return
				}
				if r.Method == http.MethodConnect {
					remote, err := net.DialTimeout("tcp", r.Host, time.Second)
					if err != nil {
						t.Error(err)
						return
					}
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						remote.Close()
						t.Error(err)
						return
					}
					io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
					relayTestTunnel(conn, remote)
					return
				}
				request := r.Clone(r.Context())
				request.RequestURI = ""
				request.Header.Del("Proxy-Authorization")
				resp, err := upstream.RoundTrip(request)
				if err != nil {
					t.Error(err)
					return
				}
				defer resp.Body.Close()
				w.WriteHeader(resp.StatusCode)
				io.Copy(w, resp.Body)
			}))
			defer proxy.Close()
			proxyURL, err := url.Parse(proxy.URL)
			if err != nil {
				t.Fatal(err)
			}
			proxyURL.User = url.UserPassword("user", "pass")
			opts := []Option{WithProxyURL(proxyURL.String())}
			if secure {
				opts = append(opts, WithTLSConfig(testTLSConfig(target)))
			}
			tr := newTestTransport(t, opts...)
			defer tr.CloseIdleConnections()
			client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
			resp, err := client.Get(target.URL)
			if secure && err == nil && resp.ProtoMajor != 2 {
				t.Error("CONNECT 隧道内未协商 HTTP/2")
			}
			if err := readReply(resp, err); err != nil {
				t.Fatal(err)
			}
			badOpts := append([]Option{}, opts...)
			badOpts = append(badOpts, WithProxyURL(proxy.URL))
			badClient := &http.Client{Transport: newTestTransport(t, badOpts...), Timeout: 3 * time.Second}
			resp, err = badClient.Get(target.URL)
			if resp != nil {
				defer resp.Body.Close()
			}
			if secure && err == nil || !secure && (err != nil || resp.StatusCode != 407) {
				t.Fatalf("代理拒绝认证未正确传递: %v %v", resp, err)
			}
			if requests.Load() != 2 {
				t.Fatalf("请求未经过代理: %d", requests.Load())
			}
		})
	}
}

func TestUnitProxy_SOCKS5(t *testing.T) {
	for _, scheme := range []string{"socks5", "socks5h"} {
		t.Run(scheme, func(t *testing.T) {
			target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
			defer target.Close()
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(3 * time.Second))
				remote, err := acceptTestSOCKS(conn, strings.TrimPrefix(target.URL, "https://"))
				done <- err
				if err == nil {
					relayTestTunnel(conn, remote)
				}
			}()
			tr := newTestTransport(t, WithProxyURL(scheme+"://user:pass@"+listener.Addr().String()), WithTLSConfig(testTLSConfig(target)))
			defer tr.CloseIdleConnections()
			client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
			// 这个域名只能由测试代理解析；如果在客户端解析或直连，请求会失败。
			_, port, _ := net.SplitHostPort(strings.TrimPrefix(target.URL, "https://"))
			if err := readReply(client.Get("https://example.com:" + port)); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func acceptTestSOCKS(conn net.Conn, target string) (net.Conn, error) {
	read := func(n int) ([]byte, error) {
		data := make([]byte, n)
		_, err := io.ReadFull(conn, data)
		return data, err
	}
	greeting, err := read(2)
	if err != nil || greeting[0] != 5 {
		return nil, fmt.Errorf("SOCKS greeting: %v", err)
	}
	if _, err := read(int(greeting[1])); err != nil {
		return nil, err
	}
	if _, err := conn.Write([]byte{5, 2}); err != nil {
		return nil, err
	}
	auth, err := read(2)
	if err != nil || auth[0] != 1 {
		return nil, fmt.Errorf("SOCKS auth: %v", err)
	}
	user, err := read(int(auth[1]))
	if err != nil {
		return nil, err
	}
	length, err := read(1)
	if err != nil {
		return nil, err
	}
	password, err := read(int(length[0]))
	if err != nil || string(user) != "user" || string(password) != "pass" {
		return nil, fmt.Errorf("SOCKS 认证不匹配")
	}
	if _, err := conn.Write([]byte{1, 0}); err != nil {
		return nil, err
	}
	header, err := read(5)
	if err != nil || header[0] != 5 || header[1] != 1 || header[3] != 3 {
		return nil, fmt.Errorf("SOCKS CONNECT 不是远端 DNS 模式: %v", err)
	}
	host, err := read(int(header[4]))
	if err != nil || string(host) != "example.com" {
		return nil, fmt.Errorf("SOCKS 目标主机错误: %v", err)
	}
	port, err := read(2)
	if err != nil || !strings.HasSuffix(target, fmt.Sprintf(":%d", binary.BigEndian.Uint16(port))) {
		return nil, fmt.Errorf("SOCKS 目标端口错误: %v", err)
	}
	remote, err := net.DialTimeout("tcp", target, time.Second)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0}); err != nil {
		remote.Close()
		return nil, err
	}
	return remote, nil
}

func TestUnitProxy_CancelCONNECT(t *testing.T) {
	done := make(chan struct{})
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		io.Copy(io.Discard, conn)
	}))
	defer proxy.Close()
	tr := newTestTransport(t, WithProxyURL(proxy.URL))
	client := &http.Client{Transport: tr, Timeout: 100 * time.Millisecond}
	start := time.Now()
	if err := readReply(client.Get("https://example.com")); err == nil {
		t.Fatal("未返回 CONNECT 超时")
	}
	if time.Since(start) > time.Second {
		t.Fatal("CONNECT 超时未及时返回")
	}
	<-done
}
