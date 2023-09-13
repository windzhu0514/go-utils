package httpclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	stdurl "net/url"
	"time"

	"golang.org/x/net/http2"

	"github.com/windzhu0514/go-utils/httpclient/metadata"
)

func Get(url string) (statusCode int, resp []byte, err error) {
	return defaultClient.Get(url)
}

func Post(url, contentType string, body interface{}) (statusCode int, resp []byte, err error) {
	return defaultClient.Post(url, contentType, body)
}

func AddCookie(cookie *http.Cookie) {
	defaultClient.AddCookie(cookie)
}

func AddCookies(cookies []*http.Cookie) {
	defaultClient.AddCookies(cookies)
}

func DelCookie(cookie *http.Cookie) {
	for i, cc := range defaultClient.cookies {
		if cookieID(cc) == cookieID(cookie) {
			defaultClient.cookies = append(defaultClient.cookies[:i], defaultClient.cookies[i+1:]...)
		}
	}
}

// get cookies from client cookie jar
func Cookies(url string) ([]*http.Cookie, error) {
	return defaultClient.Cookies(url)
}

func SetTransport(rt http.RoundTripper) *Client {
	defaultClient.client.Transport = rt
	return defaultClient
}

func SetCheckRedirect(checkRedirect func(req *http.Request, via []*http.Request) error) *Client {
	defaultClient.client.CheckRedirect = checkRedirect
	return defaultClient
}

func SetJar(jar http.CookieJar) *Client {
	if jar == nil {
		jar, _ = cookiejar.New(nil)
	}

	defaultClient.client.Jar = jar
	return defaultClient
}

func SetTimeout(timeout time.Duration) *Client {
	defaultClient.client.Timeout = timeout
	return defaultClient
}

func SetMetadata(mds ...map[string]string) *Client {
	defaultClient.metadata = metadata.New(mds...)
	return defaultClient
}

func SetMaxIdleConns(n int) *Client {
	defaultClient.transport().MaxIdleConns = n
	return defaultClient
}

func SetMaxIdleConnsPerHost(n int) *Client {
	defaultClient.transport().MaxIdleConnsPerHost = n
	return defaultClient
}

func SetMaxConnsPerHost(n int) *Client {
	defaultClient.transport().MaxConnsPerHost = n
	return defaultClient
}

func SetIdleConnTimeout(timeout time.Duration) *Client {
	defaultClient.transport().IdleConnTimeout = timeout
	return defaultClient
}

func SetProxySelector(selector ProxySelector) *Client {
	defaultClient.transport().Proxy = selector.ProxyFunc
	return defaultClient
}

// SetProxyURL
// Proxy：http://127.0.0.1:8888
// Proxy：http://username:password@127.0.0.1:8888
func SetProxyURL(proxyURL string) *Client {
	defaultClient.transport().Proxy = func(req *http.Request) (*stdurl.URL, error) {
		return stdurl.Parse(proxyURL)
	}

	return defaultClient
}

// SetProxy
// http 127.0.0.1 8888
// http 127.0.0.1 8888 username password
// socks5 127.0.0.1 8888 username password
func SetProxy(scheme, ip, port, username, password string) *Client {
	defaultClient.transport().Proxy = func(req *http.Request) (*stdurl.URL, error) {
		u := &stdurl.URL{
			Scheme: scheme,
			Host:   ip + ":" + port,
		}

		if username != "" && password != "" {
			u.User = stdurl.UserPassword(username, password)
		}

		return u, nil
	}

	return defaultClient
}

func SetCheckProxy(checkProxy func(response *Response) bool) *Client {
	defaultClient.checkProxy = checkProxy
	return defaultClient
}

// Content-Type
// https://www.iana.org/assignments/media-types/media-types.xhtml
const (
	MIMEJSON              = "application/json"
	MIMEHTML              = "text/html"
	MIMEXML               = "application/xml"
	MIMETextXML           = "text/xml"
	MIMEPlain             = "text/plain"
	MIMEPOSTForm          = "application/x-www-form-urlencoded"
	MIMEMultipartPOSTForm = "multipart/form-data"
	MIMEXPROTOBUF         = "application/x-protobuf"
	MIMEXMSGPACK          = "application/x-msgpack"
	MIMEMSGPACK           = "application/msgpack"
	MIMEYAML              = "application/x-yaml"
)

var defaultClient = NewDefaultClient()

var defaultHttp2Client = NewClientHttp2()

type Client struct {
	client        *http.Client
	metadata      metadata.Metadata
	proxySelector ProxySelector
	checkProxy    func(response *Response) bool

	cookies []*http.Cookie

	keepParamAddOrder                 bool
	jsonEscapeHTML                    bool
	jsonIndentPrefix, jsonIndentValue string
}

type ClientOption func(*Client)

func WithTransport(rt http.RoundTripper) ClientOption {
	return func(client *Client) {
		client.client.Transport = rt
	}
}

func WithCheckRedirect(checkRedirect func(req *http.Request, via []*http.Request) error) ClientOption {
	return func(client *Client) {
		client.client.CheckRedirect = checkRedirect
	}
}

func WithJar(jar http.CookieJar) ClientOption {
	return func(client *Client) {
		if jar == nil {
			jar, _ = cookiejar.New(nil)
		}
		client.client.Jar = jar
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(client *Client) {
		client.client.Timeout = timeout
	}
}

func WithMetadata(mds ...map[string]string) ClientOption {
	return func(client *Client) {
		client.metadata = metadata.New(mds...)
	}
}

func WithMaxIdleConns(n int) ClientOption {
	return func(client *Client) {
		client.transport().MaxIdleConns = n
	}
}

func WithMaxIdleConnsPerHost(n int) ClientOption {
	return func(client *Client) {
		client.transport().MaxIdleConnsPerHost = n
	}
}

func WithMaxConnsPerHost(n int) ClientOption {
	return func(client *Client) {
		client.transport().MaxConnsPerHost = n
	}
}

func WithIdleConnTimeout(timeout time.Duration) ClientOption {
	return func(client *Client) {
		client.transport().IdleConnTimeout = timeout
	}
}

func WithProxySelector(selector ProxySelector) ClientOption {
	return func(client *Client) {
		client.proxySelector = selector
		client.transport().Proxy = selector.ProxyFunc
	}
}

func WithCheckProxy(checkProxy func(response *Response) bool) ClientOption {
	return func(client *Client) {
		client.checkProxy = checkProxy
	}
}

func NewDefaultClient(opts ...ClientOption) *Client {
	c := &Client{client: &http.Client{Transport: http.DefaultTransport}}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func NewClient(opts ...ClientOption) *Client {
	c := &Client{client: &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}}}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func NewClientHttp2(opts ...ClientOption) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	http2.ConfigureTransport(transport)
	c := &Client{client: &http.Client{Transport: transport}}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func NewWithClient(hc *http.Client, opts ...ClientOption) *Client {
	c := &Client{client: hc}
	if c.client == nil {
		c.client = &http.Client{Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}}
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) NewRequest(method, url string) *Request {
	return c.NewRequestWithContext(context.Background(), method, url)
}

func (c *Client) NewRequestWithContext(ctx context.Context, method, url string) *Request {
	var req Request
	req.client = c
	req.method = method
	req.url = url
	req.heads = make(http.Header)
	req.queryParam = make(stdurl.Values)
	req.formData = make(stdurl.Values)
	req.ctx = ctx

	return &req
}

func (c *Client) Get(url string) (statusCode int, resp []byte, err error) {
	req := c.NewRequest(http.MethodGet, url)
	var response *Response
	response, err = req.Do()
	if err != nil {
		return
	}

	statusCode = response.StatusCode()
	resp, err = response.Body()

	return
}

func (c *Client) Post(url, contentType string, body interface{}) (statusCode int, resp []byte, err error) {
	req := c.NewRequest(http.MethodPost, url)

	req.SetBody(contentType, body)

	var response *Response
	response, err = req.Do()
	if err != nil {
		return
	}

	statusCode = response.StatusCode()
	resp, err = response.Body()

	return
}

func (c *Client) SetTransport(transport http.RoundTripper) *Client {
	if transport != nil {
		c.client.Transport = transport
	}

	return c
}

func (c *Client) transport() *http.Transport {
	if c.client.Transport != nil {
		return c.client.Transport.(*http.Transport)
	}
	return http.DefaultTransport.(*http.Transport)
}

func (c *Client) AddCookie(cookie *http.Cookie) *Client {
	if c.client.Jar == nil {
		c.client.Jar, _ = cookiejar.New(nil)
	}

	c.cookies = append(c.cookies, cookie)
	return c
}

func (c *Client) AddCookies(cookies []*http.Cookie) *Client {
	if c.client.Jar == nil {
		c.client.Jar, _ = cookiejar.New(nil)
	}
	c.cookies = append(c.cookies, cookies...)
	return c
}

// SetCookie 避免重复的cookie
func (c *Client) SetCookie(cookie *http.Cookie) *Client {
	if c.client.Jar == nil {
		c.client.Jar, _ = cookiejar.New(nil)
	}
	c.DelCookie(cookie)
	c.AddCookie(cookie)
	return c
}

func (c *Client) SetCookies(cookies []*http.Cookie) *Client {
	for _, cc := range cookies {
		c.SetCookie(cc)
	}
	return c
}

func cookieID(c *http.Cookie) string {
	return fmt.Sprintf("%s;%s;%s", c.Domain, c.Path, c.Name)
}

func (c *Client) DelCookie(cookie *http.Cookie) *Client {
	for i, cc := range c.cookies {
		if cookieID(cc) == cookieID(cookie) {
			c.cookies = append(c.cookies[:i], c.cookies[i+1:]...)
		}
	}
	return c
}

func (c *Client) Cookies(url string) ([]*http.Cookie, error) {
	if c.client.Jar == nil {
		return nil, errors.New("client not enable cookie jar")
	}

	URL, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	cookies := c.client.Jar.Cookies(URL)
	if len(cookies) == 0 {
		return nil, errors.New("cookies is empty")
	}

	return cookies, nil
}

func (c *Client) SetCookieJar(cookieJar http.CookieJar) {
	c.client.Jar = cookieJar
}

func (c *Client) GetCookieJar() http.CookieJar {
	return c.client.Jar
}

func (c *Client) SetJsonEscapeHTML(jsonEscapeHTML bool) *Client {
	c.jsonEscapeHTML = jsonEscapeHTML
	return c
}

func (c *Client) SetJsonIndent(prefix, indent string) *Client {
	c.jsonIndentPrefix = prefix
	c.jsonIndentValue = indent
	return c
}

func (c *Client) KeepParamAddOrder(keepParamAddOrder bool) *Client {
	c.keepParamAddOrder = keepParamAddOrder
	return c
}

// SetProxy 设置代理时保证用底层共用一个client
func (c *Client) SetProxy(ps ProxySelector) *Client {
	c.proxySelector = ps
	c.transport().Proxy = ps.ProxyFunc
	return c
}

// GetMetaDataByKey 通过key获取metadata
func (c *Client) GetMetaDataByKey(key string) string {
	return c.metadata.Get(key)
}

func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.client.Timeout = timeout
	return c
}
