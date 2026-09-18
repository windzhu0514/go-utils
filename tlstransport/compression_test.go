package tlstransport

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func TestUnitCompression_Encodings(t *testing.T) {
	for _, encoding := range []string{"gzip", "deflate", "raw-deflate", "br", "zstd", "gzip, br"} {
		t.Run(encoding, func(t *testing.T) {
			var encoded bytes.Buffer
			var writer io.WriteCloser
			var err error
			switch encoding {
			case "gzip", "gzip, br":
				writer = gzip.NewWriter(&encoded)
			case "deflate":
				writer = zlib.NewWriter(&encoded)
			case "raw-deflate":
				writer, err = flate.NewWriter(&encoded, flate.DefaultCompression)
			case "br":
				writer = brotli.NewWriter(&encoded)
			case "zstd":
				writer, err = zstd.NewWriter(&encoded, zstd.WithEncoderConcurrency(1))
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(writer, "ok"); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if encoding == "gzip, br" {
				gzipData := bytes.Clone(encoded.Bytes())
				encoded.Reset()
				writer = brotli.NewWriter(&encoded)
				if _, err := writer.Write(gzipData); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
			}
			for _, h2 := range []bool{false, true} {
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					wireEncoding := encoding
					if wireEncoding == "raw-deflate" {
						wireEncoding = "deflate"
					}
					w.Header().Set("Content-Encoding", wireEncoding)
					w.Write(encoded.Bytes())
				}))
				server.EnableHTTP2 = h2
				server.StartTLS()
				defer server.Close()
				for _, decode := range []bool{false, true} {
					tr := newTestTransport(t, WithTLSConfig(testTLSConfig(server)), WithAutoDecompression(decode))
					resp, err := (&http.Client{Transport: tr, Timeout: 3 * time.Second}).Get(server.URL)
					if err != nil {
						t.Fatal(err)
					}
					body, err := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					if decode && (string(body) != "ok" || !resp.Uncompressed || resp.Header.Get("Content-Encoding") != "" || resp.ContentLength != -1) {
						t.Fatalf("解压或响应元数据错误: %q %+v", body, resp)
					}
					if !decode && (!bytes.Equal(body, encoded.Bytes()) || resp.Uncompressed || resp.Header.Get("Content-Encoding") == "") {
						t.Fatal("禁用解压未保留原始报文")
					}
				}
			}
		})
	}
}

func TestUnitCompression_InvalidAndUnknown(t *testing.T) {
	for _, encoding := range []string{"gzip", "br", "zstd", "unknown"} {
		resp := &http.Response{Header: http.Header{"Content-Encoding": {encoding}}, Body: io.NopCloser(bytes.NewReader([]byte{0xff, 0xff, 0xff})), ContentLength: 3}
		err := decompress(resp)
		if err == nil {
			_, err = io.ReadAll(resp.Body)
		}
		resp.Body.Close()
		if encoding == "unknown" {
			if err != nil || resp.Uncompressed || resp.Header.Get("Content-Encoding") != "unknown" {
				t.Fatal("未知编码未被保留")
			}
		} else if err == nil {
			t.Fatalf("无效 %s 响应未报错", encoding)
		}
	}
}
