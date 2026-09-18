package tlstransport

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	reqhttp2 "github.com/imroc/req/v3/http2"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

type wireFingerprint struct {
	settings      string
	extraSettings []string
	window        uint32
	priority      []uint32
	streamID      uint32
	fields        []hpack.HeaderField
	err           error
}

func readHTTP2Fingerprint(conn *tls.Conn) (result wireFingerprint) {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	preface := make([]byte, len(http2.ClientPreface))
	if _, result.err = io.ReadFull(conn, preface); result.err != nil {
		return
	}
	if string(preface) != http2.ClientPreface {
		result.err = fmt.Errorf("HTTP/2 preface 无效")
		return
	}
	framer := http2.NewFramer(conn, conn)
	framer.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	if result.err = framer.WriteSettings(); result.err != nil {
		return
	}
	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			result.err = err
			return
		}
		switch frame := frame.(type) {
		case *http2.SettingsFrame:
			if !frame.IsAck() {
				var settings []string
				result.err = frame.ForeachSetting(func(s http2.Setting) error {
					settings = append(settings, fmt.Sprintf("%d:%d", s.ID, s.Val))
					return nil
				})
				if result.settings == "" {
					result.settings = strings.Join(settings, ",")
				} else {
					result.extraSettings = append(result.extraSettings, strings.Join(settings, ","))
				}
				if result.err != nil {
					return
				}
				if result.err = framer.WriteSettingsAck(); result.err != nil {
					return
				}
			}
		case *http2.WindowUpdateFrame:
			if frame.StreamID == 0 {
				result.window = frame.Increment
			}
		case *http2.PriorityFrame:
			result.priority = append(result.priority, frame.StreamID)
		case *http2.MetaHeadersFrame:
			result.streamID = frame.StreamID
			result.fields = frame.Fields
			var block bytes.Buffer
			encoder := hpack.NewEncoder(&block)
			if result.err = encoder.WriteField(hpack.HeaderField{Name: ":status", Value: "200"}); result.err != nil {
				return
			}
			if result.err = framer.WriteHeaders(http2.HeadersFrameParam{StreamID: frame.StreamID, BlockFragment: block.Bytes(), EndHeaders: true}); result.err != nil {
				return
			}
			result.err = framer.WriteData(frame.StreamID, true, []byte("ok"))
			return
		}
	}
}

func TestUnitProfile_HTTP2Wire(t *testing.T) {
	tests := []struct {
		profile  Profile
		settings string
		window   uint32
		priority []uint32
		streamID uint32
		pseudo   []string
	}{
		{Chrome133(), "1:65536,2:0,4:6291456,6:262144", 15663105, nil, 1, []string{":method", ":authority", ":scheme", ":path"}},
		{Firefox120(), "1:65536,4:131072,5:16384", 12517377, []uint32{3, 5, 7, 9, 11, 13}, 15, []string{":method", ":path", ":authority", ":scheme"}},
		{Safari160(), "4:4194304,3:100", 10485760, nil, 1, []string{":method", ":scheme", ":path", ":authority"}},
	}
	for _, tt := range tests {
		t.Run(tt.profile.Name, func(t *testing.T) {
			captured := make(chan wireFingerprint, 1)
			server := httptest.NewUnstartedServer(nil)
			server.EnableHTTP2 = true
			server.Config.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){
				"h2": func(_ *http.Server, conn *tls.Conn, _ http.Handler) { captured <- readHTTP2Fingerprint(conn) },
			}
			server.StartTLS()
			defer server.Close()
			tr := newTestTransport(t, WithProfile(tt.profile), WithTLSConfig(testTLSConfig(server)))
			client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
			if err := readReply(client.Get(server.URL)); err != nil {
				t.Fatal(err)
			}
			got := <-captured
			if got.err != nil {
				t.Fatal(got.err)
			}
			if got.settings != tt.settings || got.window != tt.window || got.streamID != tt.streamID || !slices.Equal(got.priority, tt.priority) {
				t.Fatalf("HTTP/2 线上配置不一致: %+v", got)
			}
			var extra []string
			if tt.profile.Name == "chrome_133" {
				extra = []string{"4:4194304"}
			}
			if !slices.Equal(got.extraSettings, extra) {
				t.Fatalf("有效接收窗口调整错误: %v", got.extraSettings)
			}
			var names []string
			for _, field := range got.fields {
				names = append(names, field.Name)
			}
			if len(names) < 4 || !slices.Equal(names[:4], tt.pseudo) {
				t.Fatalf("伪头顺序错误: %v", names)
			}
			last := -1
			for _, name := range names[4:] {
				index := slices.Index(tt.profile.HeaderOrder, name)
				if index < last || strings.Contains(name, "__") {
					t.Fatalf("请求头顺序错误或控制字段泄露: %v", names)
				}
				last = index
			}
		})
	}
}

func TestUnitProfile_HTTP1OrderAndIsolation(t *testing.T) {
	captured := make(chan []string, 1)
	// 使用新的 TCP listener 接收原始 HTTP/1.1 请求，避免 net/http 解析丢失顺序。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		reader := bufio.NewReader(conn)
		var lines []string
		for {
			line, err := reader.ReadString('\n')
			if err != nil || line == "\r\n" {
				break
			}
			lines = append(lines, strings.TrimSpace(line))
		}
		captured <- lines
		io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nok")
	}()
	p := Chrome133()
	p.HeaderOrder = []string{"host", "x-second", "x-first", "user-agent", "accept"}
	tr := newTestTransport(t, WithProfile(p))
	p.Headers.Set("User-Agent", "mutated")
	p.HeaderOrder[1] = "changed"
	request, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header = http.Header{"X-First": {"one"}, "X-Second": {"two"}, "User-Agent": {"caller"}, "Accept": {"application/json"}}
	before := request.Header.Clone()
	if err := readReply((&http.Client{Transport: tr, Timeout: 3 * time.Second}).Do(request)); err != nil {
		t.Fatal(err)
	}
	lines := <-captured
	if len(lines) < 6 || lines[2] != "X-Second: two" || lines[3] != "X-First: one" || lines[4] != "User-Agent: caller" || lines[5] != "Accept: application/json" {
		t.Fatalf("HTTP/1 请求头顺序错误: %v", lines)
	}
	if !reflect.DeepEqual(before, request.Header) || strings.Contains(strings.Join(lines, "\n"), "__") {
		t.Fatal("修改了调用者请求，或控制头被发送")
	}
}

func TestUnitProfile_InvalidConfig(t *testing.T) {
	for _, p := range []Profile{
		{Settings: []reqhttp2.Setting{{ID: 1}, {ID: 1}}},
		{Settings: []reqhttp2.Setting{{ID: 2, Val: 3}}},
		{PseudoHeaderOrder: []string{":path"}}, {HeaderOrder: []string{"Foo", "foo"}},
		{Headers: http.Header{"Cookie": {"test"}}}, {Headers: http.Header{"X-Test": {"a\r\nb"}}},
		{ConnectionFlow: 1 << 31}, {PriorityFrames: []reqhttp2.PriorityFrame{{StreamID: 2}}},
	} {
		if _, err := NewTransport(WithProfile(p)); err == nil {
			t.Fatal("无效 Profile 未被拒绝")
		}
	}
	newTestTransport(t, WithRandomProfile())
}

func TestUnitProfile_HTTP2SlowLargeResponse(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 5<<20)
	sent := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(sent)
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		w.Write(data)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	tr := newTestTransport(t, WithTLSConfig(testTLSConfig(server)))
	resp, err := (&http.Client{Transport: tr, Timeout: 3 * time.Second}).Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// 暂停业务消费，验证传输层能够限制对端发送并在消费恢复后继续。
	time.Sleep(100 * time.Millisecond)
	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil || n != int64(len(data)) {
		t.Fatalf("慢消费大响应失败: bytes=%d err=%v", n, err)
	}
	<-sent
}
