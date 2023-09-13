package httpclient

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	tls "github.com/refraction-networking/utls"
)

// // var proxyAddr = "socks5://127.0.0.1:8889" // "http://127.0.0.1:8888"
var (
	// proxyAddr = "socks5://127.0.0.1:8889" // "http://127.0.0.1:8888"
	// proxyAddr = "socks5://proxyUser:JMUiGRENbn2wVZYj@49.72.114.169:9096"
	proxyAddr = "http://127.0.0.1:8888" // "http://127.0.0.1:8888"
)

func TestTransport_default(t *testing.T) {
	tr := NewTransport()
	client := NewClient()
	client.SetTransport(tr)
	// test http
	url := "http://www.baidu.com/"
	req := client.NewRequest(http.MethodGet, url)
	code, resp, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_default test http code:%d, resp:%s", code, resp[0:100])
	if code != 200 {
		t.Fatal("TestTransport_default test http failed")
	}

	// test https
	url = "https://www.baidu.com/"
	req = client.NewRequest(http.MethodGet, url)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_default test https code:%d, resp:%s", code, resp[0:100])
	if code != 200 {
		t.Fatal("TestTransport_default test https failed")
	}
}

func TestTransport_no_proxy_mutil_host_default(t *testing.T) {
	tr := NewTransport()
	client := NewClient()
	client.SetTransport(tr)

	urlOne := "https://www.baidu.com/"
	urlTwo := "https://cn.bing.com/"
	req := client.NewRequest(http.MethodGet, urlOne)
	code, resp, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_no_proxy_mutil_host_default test https baidu code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_no_proxy_mutil_host_default test https failed")
	}

	req = client.NewRequest(http.MethodGet, urlTwo)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_no_proxy_mutil_host_default test https bing code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_no_proxy_mutil_host_default test https failed")
	}
	count := 0
	for host, _ := range tr.hostToInnerTransportMap {
		count++
		t.Logf("TestTransport_no_proxy_mutil_host_default transport host: %s", host)
	}

	if count != 2 {
		t.Fatal("TestTransport_no_proxy_mutil_host_default count is not right")
	}
}
func TestTransport_proxy_mutil_host(t *testing.T) {
	t.Logf("TestTransport_proxy_mutil_host use proxy_addr:%s", proxyAddr)
	opt := WithTlsConnOptProxyAddr(proxyAddr)
	tr := NewTransport(opt)
	client := NewClient()
	client.SetTransport(tr)

	urlOne := "https://www.baidu.com/"
	urlTwo := "https://cn.bing.com/"
	urlThree := "https://www.koreanair.com/"
	req := client.NewRequest(http.MethodGet, urlOne)
	code, resp, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_proxy_mutil_host test https baidu code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_proxy_mutil_host test https baidu failed")
	}

	req = client.NewRequest(http.MethodGet, urlTwo)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_proxy_mutil_host test https bing code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_proxy_mutil_host test https bing failed")
	}

	req = client.NewRequest(http.MethodGet, urlThree)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_proxy_mutil_host test https koreanair code:%d, resp:%s", code, resp[0:50])
	if code != 403 {
		t.Fatal("TestTransport_proxy_mutil_host test https koreanair failed")
	}
	count := 0
	for host, _ := range tr.hostToInnerTransportMap {
		count++
		t.Logf("TestTransport_proxy_mutil_host transport host: %s", host)
	}

	if count != 3 {
		t.Fatal("TestTransport_proxy_mutil_host count is not right")
	}
}

func TestTransport_default_tls_mutil_host(t *testing.T) {
	clientHelloID := tls.HelloChrome_96
	t.Logf("TestTransport_default_tls_mutil_host use clientHelloID:%s", clientHelloID.Client)
	opt := WithTlsConnOptClientHelloID(clientHelloID)
	tr := NewTransport(opt)
	client := NewClient()
	client.SetTransport(tr)
	ja3, ja3Md5 := tr.JA3()
	t.Logf("TestTransport_default_tls_mutil_host tls ja3:%s, ja3Md5:%s", ja3, ja3Md5)

	urlOne := "https://www.baidu.com/"
	urlTwo := "https://cn.bing.com/"
	urlThree := "https://www.koreanair.com/"
	req := client.NewRequest(http.MethodGet, urlOne)
	code, resp, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_default_tls_mutil_host test https baidu code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_default_tls_mutil_host test https baidu failed")
	}

	req = client.NewRequest(http.MethodGet, urlTwo)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_default_tls_mutil_host test https bing code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_default_tls_mutil_host test https bing failed")
	}

	req = client.NewRequest(http.MethodGet, urlThree)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_default_tls_mutil_host test https koreanair code:%d, resp:%s", code, resp[0:50])
	if code != 403 {
		t.Fatal("TestTransport_default_tls_mutil_host test https koreanair failed")
	}
	count := 0
	for host, _ := range tr.hostToInnerTransportMap {
		count++
		t.Logf("TestTransport_default_tls_mutil_host transport host: %s", host)
	}

	if count != 3 {
		t.Fatal("TestTransport_default_tls_mutil_host count is not right")
	}
}

func TestTransport_custom_tls_mutil_host(t *testing.T) {
	clientHelloID := tls.HelloCustom
	t.Logf("TestTransport_custom_tls_mutil_host use clientHelloID:%s", clientHelloID.Client)
	opts := make([]TlsConnOption, 0)
	opts = append(opts, WithTlsConnOptClientHelloID(clientHelloID))
	opts = append(opts, WithTlsConnOptClientHelloSpec(CustomClientHelloShuffleCipherSuitesSpec02))
	tr := NewTransport(opts...)
	client := NewClient()
	client.SetTransport(tr)
	ja3, ja3Md5 := tr.JA3()
	t.Logf("TestTransport_custom_tls_mutil_host tls ja3:%s, ja3Md5:%s", ja3, ja3Md5)

	urlOne := "https://www.baidu.com/"
	urlTwo := "https://cn.bing.com/"
	urlThree := "https://www.koreanair.com/"
	req := client.NewRequest(http.MethodGet, urlOne)
	code, resp, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_custom_tls_mutil_host test https baidu code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_custom_tls_mutil_host test https baidu failed")
	}

	req = client.NewRequest(http.MethodGet, urlTwo)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("TestTransport_custom_tls_mutil_host test https bing code:%d, resp:%s", code, resp[0:50])
	if code != 200 {
		t.Fatal("TestTransport_custom_tls_mutil_host test https bing failed")
	}

	req = client.NewRequest(http.MethodGet, urlThree)
	code, resp, err = req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TestTransport_custom_tls_mutil_host test https koreanair code:%d, resp:%s", code, resp[0:50])
	if code != 403 {
		t.Fatal("TestTransport_custom_tls_mutil_host test https koreanair failed")
	}
	count := 0
	for host, _ := range tr.hostToInnerTransportMap {
		count++
		t.Logf("TestTransport_custom_tls_mutil_host transport host: %s", host)
	}

	if count != 3 {
		t.Fatal("TestTransport_custom_tls_mutil_host count is not right")
	}
}

var urls = []string{
	"http://www.baidu.com/",
	"https://cn.bing.com/",
	"https://www.koreanair.com/",
	"https://fanyi.baidu.com/",
	"https://zhuanlan.zhihu.com/",
	"http://www.urlencode.com.cn/",
	"https://blog.csdn.net/",
	"https://mp.weixin.qq.com/",
	"https://juejin.cn/",
	"https://studygolang.com/",
	"https://blog.51cto.com/",
	"https://www.ping.cn",
	"https://www.itdog.cn/http/",
	"http://m.baidu.com/",
}

// test mutil goroutine
func TestTransport_mutil_goroutine(t *testing.T) {
	clientHelloID := tls.HelloChrome_70
	opts := make([]TlsConnOption, 0)
	opts = append(opts, WithTlsConnOptClientHelloID(clientHelloID))
	opts = append(opts, WithTlsConnOptProxyAddr(proxyAddr))
	tr := NewTransport(opts...)
	client := NewClient()
	client.SetTransport(tr)
	ja3, ja3Md5 := tr.JA3()
	t.Logf("TestTransport_mutil_goroutine tls client:%s version:%s ja3:%s, ja3Md5:%s",
		clientHelloID.Client, clientHelloID.Version, ja3, ja3Md5)

	worker := 4
	ch := make(chan string, 3)
	wg := sync.WaitGroup{}
	go func() {
		defer close(ch)
		for _, u := range urls {
			ch <- u
		}
	}()
	for i := 0; i < worker; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			for {
				select {
				case u, ok := <-ch:
					if !ok {
						return
					}
					req := client.NewRequest(http.MethodGet, u)
					code, _, err := req.String()
					if err != nil {
						fmt.Printf("TestTransport_mutil_goroutine num:%d host:%s, err:%s \n", num, u, err)
					} else {
						fmt.Printf("TestTransport_mutil_goroutine num:%d host:%s,  code:%d\n", num, u, code)
					}
				}
			}
		}(i)
	}
	wg.Wait()
	count := 0
	for host, _ := range tr.hostToInnerTransportMap {
		count++
		t.Logf("TestTransport_mutil_goroutine transport host: %s", host)
	}

	if count != len(urls) {
		t.Fatalf("TestTransport_mutil_goroutine count is not right, count:%d, urls length:%d", count, len(urls))
	}
}

// test reuse tranport
func TestTransport_reuse_transport(t *testing.T) {
	clientHelloID := tls.HelloCustom
	opts := make([]TlsConnOption, 0)
	opts = append(opts, WithTlsConnOptClientHelloID(clientHelloID))
	opts = append(opts, WithTlsConnOptClientHelloSpec(CustomClientHelloShuffleCipherSuitesSpec02))
	opts = append(opts, WithTlsConnOptProxyAddr(proxyAddr))
	opts = append(opts, WithTcpDialTimeout(6*time.Second))
	tr := NewTransport(opts...)
	client := NewClient()
	client.SetTransport(tr)
	ja3, ja3Md5 := tr.JA3()
	t.Logf("TestTransport_reuse_transport tls client:%s version:%s ja3:%s, ja3Md5:%s",
		clientHelloID.Client, clientHelloID.Version, ja3, ja3Md5)

	urlAddrs := []string{
		"https://www.koreanair.com/",
		"https://studygolang.com/",
		"https://zhuanlan.zhihu.com/",
		"https://www.baidu.com/",
	}
	for _, urlAddr := range urlAddrs {
		num := 10
		wg := sync.WaitGroup{}
		wg.Add(num)
		for i := 0; i < num; i++ {
			go func(n int) {
				defer wg.Done()
				req := client.NewRequest(http.MethodGet, urlAddr)
				code, _, err := req.String()
				if err != nil {
					t.Fatalf("TestTransport_reuse_transport num:%d err:%s \n", n, err)
				} else {
					t.Logf("TestTransport_reuse_transport num:%d code:%d\n", n, code)
				}
			}(i)
		}
		u, err := url.Parse(urlAddr)
		if err != nil {
			t.Fatalf("TestTransport_reuse_transport err:%s \n", err)
		}
		wg.Wait()
		t.Logf("TestTransport_reuse_transport host count is %d host:%s handshakeinfo:%d",
			len(tr.hostToInnerTransportMap), urlAddr, tr.Debug.GetTlsConnCount(u.Host))

	}
	t.Logf(" tr.Debug: %v", tr.Debug.TlsConnCount)

}

// test redirect http to https
func TestTransport_redirect_transport(t *testing.T) {
	clientHelloID := tls.HelloChrome_70
	opts := make([]TlsConnOption, 0)
	opts = append(opts, WithTlsConnOptClientHelloID(clientHelloID))
	opts = append(opts, WithTlsConnOptProxyAddr(proxyAddr))
	opts = append(opts, WithTcpDialTimeout(6*time.Second))
	tr := NewTransport(opts...)
	client := NewClient()
	client.SetTransport(tr)
	ja3, ja3Md5 := tr.JA3()
	t.Logf("TestTransport_redirect_transport tls client:%s version:%s ja3:%s, ja3Md5:%s",
		clientHelloID.Client, clientHelloID.Version, ja3, ja3Md5)

	num := 10
	urlAddr := "https://www.studygolang.com/"
	for i := 0; i < num; i++ {
		req := client.NewRequest(http.MethodGet, urlAddr)
		code, _, err := req.String()
		if err != nil {
			t.Logf("TestTransport_redirect_transport num:%d err:%s \n", i, err)
			t.Fatal(err)
		} else {
			t.Logf("TestTransport_redirect_transport num:%d code:%d\n", i, code)
		}
	}
	if len(tr.hostToInnerTransportMap) != 2 {
		t.Fatalf("TestTransport_redirect_transport host count should be 1, but %d", len(tr.hostToInnerTransportMap))
	}
}

// 已知的浏览器指纹
var TlsClientHelloArray = []tls.ClientHelloID{
	// fireFox
	tls.HelloFirefox_55, tls.HelloFirefox_56, tls.HelloFirefox_63, tls.HelloFirefox_65,
	tls.HelloFirefox_99, tls.HelloFirefox_102, tls.HelloFirefox_105,
	// chrome
	tls.HelloChrome_58, tls.HelloChrome_70, tls.HelloChrome_72, tls.HelloChrome_83,
	tls.HelloChrome_87, tls.HelloChrome_96, tls.HelloChrome_100, tls.HelloChrome_102,
	// edge
	tls.HelloEdge_85, tls.HelloEdge_106,
	// sarari
	tls.HelloSafari_16_0,
	// ios
	tls.HelloIOS_14,
}

func TestTransport_common_finger_transport(t *testing.T) {
	for _, clientHelloID := range TlsClientHelloArray {
		clientVer := clientHelloID.Client + "_" + clientHelloID.Version
		opts := make([]TlsConnOption, 0)
		opts = append(opts, WithTlsConnOptClientHelloID(clientHelloID))
		opts = append(opts, WithTlsConnOptProxyAddr(proxyAddr))
		tr := NewTransport(opts...)
		client := NewClient()
		client.SetTransport(tr)
		ja3, ja3Md5 := tr.JA3()
		t.Logf("TestTransport_common_finger_transport tls client:%s  JA3:%s, JA3Md5:%s",
			clientVer, ja3, ja3Md5)

		urlAddr := "https://www.koreanair.com/"

		req := client.NewRequest(http.MethodGet, urlAddr)
		code, _, err := req.String()
		if err != nil {
			t.Fatalf("TestTransport_common_finger_transport clientHelloID:%s, err:%s \n",
				clientVer, err)
		} else {
			t.Logf("TestTransport_common_finger_transport code:%d\n", code)
		}

		if len(tr.hostToInnerTransportMap) != 1 {
			t.Fatalf("TestTransport_common_finger_transport host count should be 1, but %d", len(tr.hostToInnerTransportMap))
		}
	}
}

// 测试使用wireshark抓的原始client hello报文来握手
func TestFingerprintClientHello(t *testing.T) {
	byteString := []byte("160303010701000103030358f01419f441b6e3d81f04f83608414e1ba8b76f064cc9bfab8d8cdb9058b14e00001ac02bc02fc02cc030cca9cca8c013c014009c009d002f0035000a010000c00000001600140000117777772e6b6f7265616e6169722e636f6d000500050100000000000a000c000a001d001700180019001e000b00020100000d002800260403050306030804080508060809080a080b04010501060104020303030103020203020102020032002800260403050306030804080508060809080a080b04010501060104020303030103020203020102020010000e000c02683208687474702f312e310011000900070200040000000000170000002b0003020303ff01000100")
	addr := "36.152.44.95:443"
	defaultConfig := &tls.Config{ServerName: "www.baidu.com"}
	dialConn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("TestFingerprintClientHello dail failed: %v", err)
	}
	uConn := tls.UClient(dialConn, defaultConfig, tls.HelloCustom)
	helloBytes := make([]byte, hex.DecodedLen(len(byteString)))
	_, err = hex.Decode(helloBytes, byteString)
	if err != nil {
		t.Fatalf("TestFingerprintClientHello fingerprinting Decode failed: %v", err)
	}
	fingerprinter := &tls.Fingerprinter{
		AllowBluntMimicry: true,
	}
	generatedSpec, err := fingerprinter.FingerprintClientHello(helloBytes)
	if err != nil {
		t.Fatalf("TestFingerprintClientHello fingerprinting failed: %v", err)
	}
	generatedSpec.TLSVersMax = tls.VersionTLS13
	generatedSpec.TLSVersMin = tls.VersionTLS10
	if err := uConn.ApplyPreset(generatedSpec); err != nil {
		t.Fatalf("TestFingerprintClientHello applying generated spec failed: %v", err)
	}
	err = uConn.BuildHandshakeState()
	if err != nil {
		t.Fatalf("TestFingerprintClientHello BuildHandshakeState spec failed: %v", err)
	}
	err = uConn.Handshake()
	if err != nil {
		t.Fatalf("TestFingerprintClientHello Handshake spec failed: %v", err)
	}
	t.Log("TestFingerprintClientHello success")
}

// test handshake timeouot
func TestTransport_timeout_handshake(t *testing.T) {
	// t.Skip()

	clientHelloID := tls.HelloChrome_96
	t.Logf("TestTransport_timeout_handshake use clientHelloID:%s", clientHelloID.Client)
	opts := make([]TlsConnOption, 0)
	opts = append(opts, WithTlsConnOptClientHelloID(clientHelloID))
	opts = append(opts, WithTlsConnOptProxyAddr(proxyAddr))
	opts = append(opts, WithHandShakeTimeout(10000*time.Millisecond))
	tr := NewTransport(opts...)
	client := NewClient()
	client.SetTransport(tr)
	ja3, ja3Md5 := tr.JA3()
	t.Logf("TestTransport_timeout_handshake tls ja3:%s, ja3Md5:%s", ja3, ja3Md5)

	urlOne := "https://www.baidu.com/"
	req := client.NewRequest(http.MethodGet, urlOne)
	_, _, err := req.String()
	host := "www.baidu.com"
	t.Logf("TestTransport_timeout_handshake host:%s cost:%d", host, tr.GetTlsHandShakeTime(host))
	if err != nil {
		if strings.Contains(err.Error(), "HandshakeContextError") {
			t.Logf("TestTransport_timeout_handshake %s", err.Error())
			return
		}
		t.Fatal(err)
	}

}
