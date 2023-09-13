package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	stdurl "net/url"
	"strings"

	"github.com/windzhu0514/go-utils/httpclient/metadata"
)

type Request struct {
	client *Client

	url                               string
	method                            string
	heads                             http.Header
	queryParam                        stdurl.Values
	keepParamAddOrder                 bool
	queryParamKeys                    []string
	formData                          stdurl.Values
	body                              interface{}
	cookies                           []*http.Cookie
	ctx                               context.Context
	jsonEscapeHTML                    bool
	jsonIndentPrefix, jsonIndentValue string
	checkRedirect                     func(req *http.Request, via []*http.Request) error
	checkProxy                        func(response *Response) bool
}

type DecryptFunc = func(string) (string, error)

func NewRequest(method, url string) *Request {
	return defaultClient.NewRequest(method, url)
}

func NewRequestWithContext(ctx context.Context, method, url string) *Request {
	return defaultClient.NewRequestWithContext(ctx, method, url)
}

func (r *Request) SetClient(cli *Client) *Request {
	r.client = cli
	return r
}

// SetHead 设置head 自动规范化
func (r *Request) SetHead(key, value string) *Request {
	r.heads.Set(key, value)
	return r
}

// AddHead 添加head 自动规范化
func (r *Request) AddHead(key, value string) *Request {
	r.heads.Add(key, value)
	return r
}

// SetHeads 设置head 自动规范化
func (r *Request) SetHeads(headers http.Header) *Request {
	for key, values := range headers {
		for _, value := range values {
			r.heads.Set(key, value)
		}
	}
	return r
}

// AddHeads 添加head 自动规范化
func (r *Request) AddHeads(headers http.Header) *Request {
	for key, values := range headers {
		for _, value := range values {
			r.heads.Add(key, value)
		}
	}
	return r
}

// SetRawHead 设置head 不自动规范化
func (r *Request) SetRawHead(key, value string) *Request {
	r.heads[key] = []string{value}
	return r
}

// SetRawHeads 设置head 不自动规范化
func (r *Request) SetRawHeads(heads map[string]string) *Request {
	for key, value := range heads {
		r.heads[key] = []string{value}
	}
	return r
}

// SetQueryParam 添加URL path参数
func (r *Request) SetQueryParam(key, value string) *Request {
	r.queryParam.Set(key, value)
	r.queryParamKeys = append(r.queryParamKeys, key)
	return r
}

// SetQueryParam 添加URL path参数
func (r *Request) SetQueryParams(params map[string]string) *Request {
	for k, v := range params {
		r.queryParam.Set(k, v)
		r.queryParamKeys = append(r.queryParamKeys, k)
	}
	return r
}

// KeepQueryParamOrder 保持查询参数添加顺序
func (r *Request) KeepQueryParamOrder(keepParamAddOrder bool) *Request {
	r.keepParamAddOrder = keepParamAddOrder
	return r
}

// SetFormData 添加请求参数
func (r *Request) SetFormData(key, value string) *Request {
	r.formData.Set(key, value)
	return r
}

// SetRawFormData 添加请求参数
func (r *Request) SetRawFormData(formData map[string]string) *Request {
	for k, v := range formData {
		r.formData.Set(k, v)
	}
	return r
}

// SetCheckRedirect 设置该请求的重定向函数
func (r *Request) SetCheckRedirect(checkRedirect func(req *http.Request, via []*http.Request) error) *Request {
	r.checkRedirect = checkRedirect
	return r
}

// SetCheckProxy 设置代理检查函数
func (r *Request) SetCheckProxy(checkProxy func(response *Response) bool) *Request {
	r.checkProxy = checkProxy
	return r
}

// Context 获取请求的Context
func (r *Request) Context() context.Context {
	return r.ctx
}

// WithContext 设置请求的Context
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// AddCookie 添加cookie
// client CookieJar 默认不启用，如需使用 cookie ,先设置client 启用 CookieJar
// 只允许通过 client 设置，避免并发创建
func (r *Request) AddCookie(cookie *http.Cookie) *Request {
	r.cookies = append(r.cookies, cookie)
	return r
}

// AddCookies 添加cookie
// client CookieJar 默认不启用，如需使用 cookie ,先设置client 启用 CookieJar
// 只允许通过 client 设置，避免并发创建
func (r *Request) AddCookies(cookies []*http.Cookie) *Request {
	r.cookies = append(r.cookies, cookies...)
	return r
}

// SetBody 设置body
func (r *Request) SetBody(contentType string, body interface{}) *Request {
	r.SetHead("Content-Type", contentType)
	r.body = body
	return r
}

// SetJsonEscapeHTML 设置该请求json编码时是否转义HTML字符
func (r *Request) SetJsonEscapeHTML() *Request {
	r.jsonEscapeHTML = true
	return r
}

// SetJsonIndent 设置该请求json编码时的缩进格式 都为空不进行缩进
func (r *Request) SetJsonIndent(prefix, indent string) *Request {
	r.jsonIndentPrefix = prefix
	r.jsonIndentValue = indent
	return r
}

// CustomRequest 自定义Request
func (r *Request) CustomRequest(f func(request *Request)) *Request {
	f(r)
	return r
}

// Do TODO: retry
func (r *Request) Do() (*Response, error) {
	if r.checkRedirect != nil {
		r.client.client.CheckRedirect = r.checkRedirect
	}

	var body io.Reader
	if len(r.formData) > 0 {
		body = bytes.NewBuffer([]byte(r.formData.Encode()))
	} else if r.body != nil {
		switch data := r.body.(type) {
		case io.Reader:
			body = data
		case []byte:
			body = bytes.NewReader(data)
		case string:
			body = strings.NewReader(data)
		default:
			buf := bytes.NewBuffer(nil)
			enc := json.NewEncoder(buf)
			if r.jsonEscapeHTML || r.client.jsonEscapeHTML {
				enc.SetEscapeHTML(true)
			}

			jsonIndentPrefix := r.client.jsonIndentPrefix
			jsonIndentValue := r.client.jsonIndentValue

			if r.jsonIndentPrefix != "" {
				jsonIndentPrefix = r.jsonIndentPrefix
			}

			if r.jsonIndentValue != "" {
				jsonIndentValue = r.jsonIndentValue
			}

			enc.SetIndent(jsonIndentPrefix, jsonIndentValue)

			if err := enc.Encode(data); err != nil {
				return nil, err
			}
			body = buf
		}
	}

	reqURL, err := stdurl.Parse(r.url)
	if err != nil {
		return nil, err
	}

	var queryParam string
	if r.client.keepParamAddOrder || r.keepParamAddOrder {
		var buf strings.Builder
		for i := 0; i < len(r.queryParamKeys); i++ {
			vs := r.queryParam[r.queryParamKeys[i]]
			keyEscaped := stdurl.QueryEscape(r.queryParamKeys[i])
			for _, v := range vs {
				if buf.Len() > 0 {
					buf.WriteByte('&')
				}
				buf.WriteString(keyEscaped)
				buf.WriteByte('=')
				buf.WriteString(stdurl.QueryEscape(v))
			}
		}
		queryParam = buf.String()
	} else {
		queryParam = r.queryParam.Encode()
	}

	if len(queryParam) > 0 {
		if reqURL.RawQuery == "" {
			reqURL.RawQuery = queryParam
		} else {
			reqURL.RawQuery = reqURL.RawQuery + "&" + queryParam
		}
	}

	r.url = reqURL.String()

	ctx := metadata.NewRequestContext(r.ctx, r.client.metadata)
	req, err := http.NewRequestWithContext(ctx, r.method, r.url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range r.heads {
		req.Header[key] = value
	}

	if len(r.client.cookies) > 0 {
		if r.client.client.Jar != nil {
			r.client.client.Jar.SetCookies(req.URL, r.client.cookies)
		}
	}

	if len(r.cookies) > 0 {
		if r.client.client.Jar != nil {
			r.client.client.Jar.SetCookies(req.URL, r.cookies)
		}
	}

	var resp Response
	resp.resp, err = r.client.client.Do(req)

	checkProxy := r.client.checkProxy
	if r.checkProxy != nil {
		checkProxy = r.checkProxy
	}
	if checkProxy != nil && !checkProxy(&resp) {
		r.client.proxySelector.ProxyInvalid(ctx)
	}

	return &resp, err
}

func (r *Request) Unmarshal(val interface{}) (err error) {
	_, resp, err := r.String()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(resp), val)
}

func (r *Request) UnmarshalWithDecrypt(decrypt DecryptFunc, val interface{}) (err error) {
	_, resp, err := r.String()
	if err != nil {
		return err
	}
	if decrypt != nil {
		resp, err = decrypt(resp)
		if err != nil {
			return err
		}
	}
	return json.Unmarshal([]byte(resp), val)
}

func (r *Request) StringWithDecrypt(decrypt DecryptFunc) (statusCode int, resp string, err error) {
	statusCode, resp, err = r.String()
	if decrypt != nil {
		resp, err = decrypt(resp)
	}
	return
}

func (r *Request) String() (statusCode int, resp string, err error) {
	var response *Response
	response, err = r.Do()
	if err != nil {
		return
	}
	statusCode = response.resp.StatusCode
	respByte, err := response.Body()
	resp = string(respByte)
	return
}

func (r *Request) Byte() (statusCode int, resp []byte, err error) {
	var response *Response
	response, err = r.Do()
	if err != nil {
		return
	}

	statusCode = response.resp.StatusCode
	resp, err = response.Body()
	return
}

func (r *Request) GetUrl() string {
	return r.url
}
