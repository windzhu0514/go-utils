package tlstransport

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func decompress(resp *http.Response) error {
	encoding := resp.Header.Get("Content-Encoding")
	if encoding == "" || resp.Body == nil || resp.ContentLength == 0 {
		return nil
	}
	encodings := strings.Split(strings.ToLower(encoding), ",")
	for i, name := range encodings {
		encodings[i] = strings.TrimSpace(name)
		switch encodings[i] {
		case "gzip", "deflate", "br", "zstd":
		default:
			return nil // 未识别的编码完整保留，不能移除头部伪装成已解压。
		}
	}
	body := &decodedBody{Reader: resp.Body, source: resp.Body}
	for i := len(encodings) - 1; i >= 0; i-- {
		var reader io.Reader
		var closer io.Closer
		var err error
		switch encodings[i] {
		case "gzip":
			r, e := gzip.NewReader(body.Reader)
			reader, closer, err = r, r, e
		case "deflate":
			buffered := bufio.NewReader(body.Reader)
			prefix, e := buffered.Peek(2)
			if e != nil {
				err = e
			} else if prefix[0]&0x0f == 8 && (int(prefix[0])*256+int(prefix[1]))%31 == 0 {
				r, e := zlib.NewReader(buffered)
				reader, closer, err = r, r, e
			} else {
				r := flate.NewReader(buffered)
				reader, closer = r, r
			}
		case "br":
			reader = brotli.NewReader(body.Reader)
		case "zstd":
			r, e := zstd.NewReader(body.Reader, zstd.WithDecoderConcurrency(1))
			if e == nil {
				reader, closer = r, r.IOReadCloser()
			}
			err = e
		}
		if err != nil {
			body.Close()
			return fmt.Errorf("tlstransport: 解压 %s 响应失败: %w", encodings[i], err)
		}
		body.Reader = reader
		if closer != nil {
			body.decoders = append(body.decoders, closer)
		}
	}
	resp.Body = body
	resp.Header.Del("Content-Encoding")
	resp.Header.Del("Content-Length")
	resp.ContentLength = -1
	resp.Uncompressed = true
	return nil
}

type decodedBody struct {
	io.Reader
	source   io.ReadCloser
	decoders []io.Closer
}

func (b *decodedBody) Close() error {
	// 先关闭网络响应体，使未完成的读取及时退出。
	err := b.source.Close()
	for i := len(b.decoders) - 1; i >= 0; i-- {
		err = errors.Join(err, b.decoders[i].Close())
	}
	return err
}
