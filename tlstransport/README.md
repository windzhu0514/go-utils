# TLS 指纹 Transport

`spiderman/pkg/tlstransport` 提供标准 `http.RoundTripper`，可注入 `http.Client`、Resty 等客户端。默认采用 Chrome 133 Profile、开启证书校验、显式直连，TCP 连接和目标 TLS 握手各超时 10 秒。

## 设计与取舍

| 参考库 | 采用的能力 | 本包落点 |
| --- | --- | --- |
| [tls-ja3](https://git.17usoft.com/GS-util/tls-ja3) | Transport 注入、函数选项、JA3、自定义指纹、代理 | `NewTransport`、`With...`、`JA3` |
| [uTLS](https://github.com/refraction-networking/utls) | 控制 ClientHello、扩展及顺序、GREASE、浏览器模板 | 复用项目锁定的 uTLS v1.8.0；每次握手生成独立 Spec |
| [tls-client](https://github.com/bogdanfinn/tls-client) | 将 TLS 和 HTTP/2 参数绑定为完整 Profile | `Profile`、有序 SETTINGS、WINDOW_UPDATE、PRIORITY、伪头顺序 |
| [curl-impersonate](https://github.com/lexiforest/curl-impersonate) | 同时匹配 TLS 与 HTTP 层、浏览器请求头、压缩能力 | 固定版本 Profile、HTTP/1.1 与 HTTP/2 请求头排序、gzip/deflate/br/zstd |

传输层复用项目现有的 `req/v3 v3.54.0`；不增加模块、不调用外部 curl、不引入 CGO。使用现有 `cryptobyte` 解析带长度前缀的 ClientHello，避免手写易越界的二进制解析器。`go.mod` 和 `go.sum` 保持原样。

职责分为 `profile.go`（浏览器配置）、`options.go`（校验选项）、`transport.go`（握手与传输）、`http2_compat.go`（底层窗口兼容）、`ja3.go`（指纹解析）、`compression.go`（流式解压）。本包不处理航司流程、账号、代理获取、业务重试或日志。

### 浏览器配置

| 构造函数 | TLS 模板 | HTTP/2 SETTINGS 顺序与值 | 连接窗口增量 |
| --- | --- | --- | --- |
| `Chrome133()` | `HelloChrome_133` | `1:65536,2:0,4:6291456,6:262144` | `15663105` |
| `Firefox120()` | `HelloFirefox_120` | `1:65536,4:131072,5:16384` | `12517377` |
| `Safari160()` | `HelloSafari_16_0` | `4:4194304,3:100` | `10485760` |

HTTP/2 数值及顺序参考 tls-client 的 [Chrome/Safari 配置](https://github.com/bogdanfinn/tls-client/blob/master/profiles/internal_browser_profiles.go) 和 [Firefox 配置](https://github.com/bogdanfinn/tls-client/blob/master/profiles/contributed_browser_profiles.go)，请求头排序参考项目锁定版本 req 的 `client_impersonate.go`。这些是本包固定的模板，不随上游网页更新自动改变。

**HTTP/2 窗口差异：** 表格描述首个 SETTINGS。req v3.54.0 内部流接收窗口固定为 4 MiB；直接通告更大的窗口会使慢读取的大响应触发 `FLOW_CONTROL_ERROR`。本包在这种情况下保留首帧，并在任何请求 HEADERS 前追加 `SETTINGS_INITIAL_WINDOW_SIZE=4194304`，同时消费这帧的额外 ACK。Chrome133 默认会产生这一附加帧，Firefox120、Safari160 不会。有效窗口因此最多为 4 MiB；该行为可被观察，不能视为完整的浏览器 HTTP/2 指纹复刻。测试覆盖首帧、附加帧和 5 MiB 慢读取响应。

默认头提供通用请求基线；导航、XHR、表单等场景的 `Accept`、`Origin`、`Referer`、`Sec-Fetch-*` 应依据实际报文由调用者设置。Profile 不代表完整浏览器环境，也不保证与某个真实浏览器的全部网络行为一致。

## 标准客户端

```go
package example

import (
    "net/http"
    "net/http/cookiejar"
    "time"

    "spiderman/pkg/tlstransport"
)

func NewClient(proxyURL string) (*http.Client, error) {
    tr, err := tlstransport.NewTransport(
        tlstransport.WithProfile(tlstransport.Chrome133()),
        tlstransport.WithProxyURL(proxyURL),
        tlstransport.WithTCPDialTimeout(6*time.Second),
        tlstransport.WithTLSHandshakeTimeout(6*time.Second),
    )
    if err != nil {
        return nil, err
    }
    jar, err := cookiejar.New(nil)
    if err != nil {
        tr.CloseIdleConnections()
        return nil, err
    }
    return &http.Client{
        Transport: tr,
        Jar: jar,
        Timeout: 30*time.Second,
    }, nil
}
```

请求沿用 `http.NewRequestWithContext` 和 `client.Do`。读完后关闭 `resp.Body`；客户端不用时调用 `client.CloseIdleConnections()`。Cookie、重定向策略及覆盖响应体读取的整次超时由 `http.Client` 管理。

Resty 注入方式：

```go
tr, err := tlstransport.NewTransport(tlstransport.WithProfile(tlstransport.Firefox120()))
if err != nil {
    return err
}
defer tr.CloseIdleConnections()
client := resty.New().SetTransport(tr).SetTimeout(30*time.Second)
```

## 自定义与随机配置

`WithProfile` 同时设置 TLS 与 HTTP；`WithRandomProfile()` 从三个内置 Profile 中选择，也可传入自定义候选。选择发生在创建时，同一 Transport 的连接不会随机切换浏览器身份。

```go
profile := tlstransport.Chrome133()
profile.Name = "custom"
profile.HeaderOrder = []string{
    "host", "user-agent", "accept", "content-type", "x-request-id", "cookie",
}
profile.SpecFactory = func() (utls.ClientHelloSpec, error) {
    spec, err := utls.UTLSIdToSpec(utls.HelloChrome_133)
    if err != nil {
        return spec, err
    }
    // 按协议证据修改 spec；此实例及扩展仅属于本次调用。
    return spec, nil
}
tr, err := tlstransport.NewTransport(tlstransport.WithProfile(profile))
```

`Profile.Settings`、`ConnectionFlow`、`HeaderPriority`、`PriorityFrames`、`PseudoHeaderOrder` 可独立配置。顺序表未覆盖的业务头按名称排序追加；传输控制字段不会发送给服务器。请求已有的头优先于 Profile 默认值；设置空值可抑制默认值，Transport 不修改原始请求。

只需要 TLS 指纹时，使用 `WithClientHelloID(utls.HelloFirefox_120)` 或 `WithClientHelloSpec(factory)`；这两个选项会清除完整 Profile，保留普通 HTTP 行为。`WithRandomBrowser(ids...)` 只随机选择 TLS 模板。多个选项按顺序应用，后面的指纹选项覆盖前面的配置。

指纹工厂必须可并发调用，并返回全新的切片、扩展及内部可变对象。uTLS 会修改 SNI、GREASE、密钥等字段，不能返回同一个共享 `*ClientHelloSpec`。构造校验和 JA3 预览也会调用工厂，不应在其中发起业务请求。

## 代理、证书与响应

- `WithProxyURL` 支持 `http://`、`socks5://`、`socks5h://` 及 URL 用户名密码。HTTP 代理同时支持普通 HTTP 转发和 HTTPS CONNECT；两种 SOCKS5 scheme 均由代理解析目标域名。空值为直连，不读取环境代理。不支持 HTTPS 代理端点。
- 无效代理 URL 在构造时返回错误，错误不包含 URL 中的凭据；不会在代理失败后自动直连。
- `WithTLSConfig(*utls.Config)` 配置目标服务器的 RootCAs、ServerName、证书校验回调等。默认校验证书和主机名；未指定 ServerName 时按目标主机填写，IP 地址不会错误生成 SNI。
- TLS 版本、cipher suites、ALPN 等 ClientHello 参数由 Spec 决定。证书回调、证书对象和会话缓存等引用对象必须满足 uTLS 的并发约定。HTTP 响应中的标准 TLS 状态提供证书和协商信息，但不能使用标准 `ConnectionState.ExportKeyingMaterial`；需要此能力时使用 uTLS 校验回调中的原始状态。
- 默认自动解压 gzip、zlib/raw deflate、Brotli、Zstd 及它们的多层 Content-Encoding；移除对应编码和长度头，设置 `Uncompressed`。未知编码原样保留，损坏数据在请求或读取响应体时返回错误。`WithAutoDecompression(false)` 保留压缩报文。
- Transport 按目标 origin 隔离底层连接池，避免 req 在跨目标 HTTP/2 初始化时改写共享字段的竞态；同一 origin 内仍复用连接并支持并发。配置在创建后固定；换代理或 Profile 时创建新 Transport，并关闭旧实例空闲连接。请求取消及时返回，连接建立遵循底层连接池语义；取消单次请求不意味着强制中断其他请求正在共享的连接。

## JA3

```go
value, md5Hex, err := tr.JA3("example.com")
```

`JA3()` 在本地序列化一次 ClientHello 后计算，默认 SNI 为 `example.com`，不连接网络。解析使用消息的真实 TLS legacy version，过滤 cipher、extension、group 中的 GREASE；不硬编码所有版本为 771。

随机扩展顺序、SNI、padding、会话恢复会改变真实握手，预览不能当作实际请求的固定 JA3。对真实抓包，先重组完整握手消息，再调用 `JA3FromClientHello(raw)`；输入包含 4 字节握手头，不含 TLS record 头。JA3 字符串本身不包含所有扩展内容，因此本包不提供“只凭 JA3 字符串还原完整浏览器握手”的接口。

## 从 tls-ja3 迁移

| 旧用法 | 新用法 |
| --- | --- |
| `tr := tlsja3.NewTransport(...)` | `tr, err := tlstransport.NewTransport(...)`，处理配置错误 |
| `WithTlsConnOptProxyAddr` | `WithProxyURL` |
| `WithTcpDialTimeout` | `WithTCPDialTimeout` |
| `WithHandShakeTimeout` | `WithTLSHandshakeTimeout` |
| `WithTlsConfig` | `WithTLSConfig` |
| `WithTlsConnOptClientHelloID(TlsHello)` | `WithClientHelloID(utls.ClientHelloID)` 或完整 `WithProfile` |
| `TlsHello.ClientCustomHelloSpec` | `WithClientHelloSpec(SpecFactory)`，每次返回独立 Spec |
| `RandSetBrowser()` | 构造时传 `WithRandomProfile()` 或仅 TLS 的 `WithRandomBrowser()` |
| `JA3()` 两个返回值 | 三个返回值，增加 `error` |

这是用法相似的新包，不是仅替换 import 即可编译的兼容层。未迁移任何站点，也没有删除旧库依赖。

## 验证与边界

```sh
go test ./pkg/tlstransport -run '^TestUnit' -count=1
CGO_ENABLED=1 go test -race ./pkg/tlstransport -run '^TestUnit' -count=1
go vet ./pkg/tlstransport
go build ./pkg/tlstransport
git diff --check
```

测试全部在本地：HTTP/1.1、HTTP/2、实发 SETTINGS/WINDOW_UPDATE/PRIORITY/HPACK 头顺序、连接复用、证书、并发自定义指纹、HTTP/CONNECT/SOCKS5 代理、取消超时、Cookie/重定向/Resty、压缩和 JA3。race 测试需要支持 CGO 的工具链；库运行本身不要求 CGO。

未实现 HTTP/3、QUIC 指纹、libcurl 引擎、浏览器 JavaScript 环境和 multipart 浏览器边界生成。当前 Profile 的 HTTP/2 行为仍受 req 实现约束；本地协议测试不等于真实航司验收或完整浏览器指纹一致性验证。
