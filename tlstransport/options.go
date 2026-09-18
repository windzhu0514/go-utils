package tlstransport

import (
	"errors"
	"math/rand/v2"
	"net/url"
	"strconv"
	"time"

	utls "github.com/refraction-networking/utls"
)

// SpecFactory 为每次握手返回独立的指纹及扩展对象；实现必须支持并发调用。
type SpecFactory func() (utls.ClientHelloSpec, error)

// Option 设置 Transport，仅在 NewTransport 中应用。
type Option func(*config) error

type config struct {
	spec             SpecFactory
	profile          *Profile
	autoDecompress   bool
	proxyURL         *url.URL
	tlsConfig        *utls.Config
	dialTimeout      time.Duration
	handshakeTimeout time.Duration
}

// WithClientHelloID 选择 uTLS 内置指纹。HelloCustom 请使用 WithClientHelloSpec。
func WithClientHelloID(id utls.ClientHelloID) Option {
	return func(c *config) error {
		if _, err := utls.UTLSIdToSpec(id); err != nil {
			return err
		}
		c.spec = func() (utls.ClientHelloSpec, error) { return utls.UTLSIdToSpec(id) }
		c.profile = nil
		return nil
	}
}

// WithClientHelloSpec 设置指纹工厂。构造校验、JA3 预览和每次握手均会调用它。
// 返回的切片和扩展不能与其他调用共享，uTLS 会修改其中的 SNI、密钥等状态。
func WithClientHelloSpec(factory SpecFactory) Option {
	return func(c *config) error {
		if factory == nil {
			return errors.New("tlstransport: 指纹工厂不能为空")
		}
		c.spec = factory
		c.profile = nil
		return nil
	}
}

// WithAutoDecompression 控制 gzip、deflate、br、zstd 自动解压，默认开启。
// 关闭后，响应体和 Content-Encoding 保持原样。
func WithAutoDecompression(enabled bool) Option {
	return func(c *config) error {
		c.autoDecompress = enabled
		return nil
	}
}

// WithRandomBrowser 在创建时选择一个浏览器，连接复用期间不切换身份。
// 默认候选为当前 uTLS 的 Chrome、Firefox、Safari；可传入自己的候选列表。
func WithRandomBrowser(ids ...utls.ClientHelloID) Option {
	ids = append([]utls.ClientHelloID(nil), ids...)
	if len(ids) == 0 {
		ids = []utls.ClientHelloID{utls.HelloChrome_Auto, utls.HelloFirefox_Auto, utls.HelloSafari_Auto}
	}
	return func(c *config) error {
		for _, id := range ids {
			if _, err := utls.UTLSIdToSpec(id); err != nil {
				return err
			}
		}
		return WithClientHelloID(ids[rand.IntN(len(ids))])(c)
	}
}

// WithProxyURL 设置显式代理；空字符串表示直连，不读取环境代理。
// 支持 http、socks5、socks5h 及 URL 中的用户名密码。
func WithProxyURL(rawURL string) Option {
	return func(c *config) error {
		if rawURL == "" {
			c.proxyURL = nil
			return nil
		}
		u, err := url.Parse(rawURL)
		// 不透传解析错误，避免错误文本泄露代理凭据。
		if err != nil || u.Hostname() == "" || u.Opaque != "" || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return errors.New("tlstransport: 代理 URL 无效")
		}
		switch u.Scheme {
		case "http", "socks5", "socks5h":
		default:
			return errors.New("tlstransport: 不支持的代理协议")
		}
		if port := u.Port(); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return errors.New("tlstransport: 代理端口无效")
			}
		}
		c.proxyURL = u
		return nil
	}
}

// WithTLSConfig 设置目标服务器的证书信任、校验回调和会话配置。
// ClientHello 参数由指纹决定；创建后不得修改共享的证书、回调等引用对象。
func WithTLSConfig(cfg *utls.Config) Option {
	return func(c *config) error {
		if cfg == nil {
			return errors.New("tlstransport: TLS 配置不能为空")
		}
		c.tlsConfig = cfg.Clone()
		return nil
	}
}

// WithTCPDialTimeout 设置 TCP 连接超时；必须大于零，默认 10 秒。
func WithTCPDialTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		if timeout <= 0 {
			return errors.New("tlstransport: TCP 连接超时必须大于零")
		}
		c.dialTimeout = timeout
		return nil
	}
}

// WithTLSHandshakeTimeout 设置目标 TLS 握手超时；必须大于零，默认 10 秒。
func WithTLSHandshakeTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		if timeout <= 0 {
			return errors.New("tlstransport: TLS 握手超时必须大于零")
		}
		c.handshakeTimeout = timeout
		return nil
	}
}
