package httpclient

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	stdurl "net/url"
	"os"
)

// Response 请求结果
type Response struct {
	resp *http.Response
	body []byte
}

// StatusCode 返回状态码
func (r *Response) StatusCode() int {
	if r == nil || r.resp == nil {
		return 0
	}

	return r.resp.StatusCode
}

// Headers 返回请求结果的heads
func (r *Response) Headers() http.Header {
	if r == nil || r.resp == nil {
		return http.Header{}
	}

	return r.resp.Header
}

// Cookies 返回请求结果的Cookie
func (r *Response) Cookies() []*http.Cookie {
	if r == nil || r.resp == nil {
		return nil
	}

	return r.resp.Cookies()
}

// Location 返回重定向地址
func (r *Response) Location() (*stdurl.URL, error) {
	if r == nil || r.resp == nil {
		return nil, errors.New("response is nil")
	}

	return r.resp.Location()
}

// Body 返回请求结果的body 超时时间包括body的读取 请求结束后要尽快读取
func (r *Response) Body() (body []byte, err error) {
	if r == nil || r.resp == nil {
		return nil, errors.New("response is nil")
	}

	if r.body != nil {
		return r.body, nil
	}

	if r.resp.Body == nil {
		return nil, errors.New("response is nil")
	}

	defer r.resp.Body.Close()

	if r.resp.Header.Get("Content-Encoding") == "gzip" {
		reader, readerErr := gzip.NewReader(r.resp.Body)
		if readerErr != nil {
			return nil, readerErr
		}
		r.body, err = io.ReadAll(reader)
	} else {
		r.body, err = io.ReadAll(r.resp.Body)
	}

	return r.body, nil
}

// FromJSON 解析请求结果到 v
func (r *Response) FromJSON(v interface{}) error {
	resp, err := r.Body()
	if err != nil {
		return err
	}

	return json.Unmarshal(resp, v)
}

// ToFile 保存请求结果到文件
func (r *Response) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := r.Body()
	if err != nil {
		return err
	}

	_, err = io.Copy(f, bytes.NewReader(resp))
	return err
}

func (r *Response) Response() *http.Response {
	return r.resp
}
