# 浏览器与 TLS 指纹

## 先选择正确的层级

大多数业务优先使用浏览器 profile：

```go
client := req.C().ImpersonateChromeWithOS(req.BrowserOSWindows)
```

Chrome 和 Firefox 可选 `BrowserOSWindows`、`BrowserOSMacOS`、`BrowserOSLinux`、`BrowserOSAndroid`、`BrowserOSIOS`、`BrowserOSRandom`；Safari 使用 `ImpersonateSafari`。

profile 会组合 Header、Header 顺序、HTTP/2 参数、HTTP/1.1/2 的 uTLS ClientHello，以及该 profile 明确提供的 HTTP/3 设置。Chrome/Firefox 的 Header 会按请求方法选择，Safari 当前使用一组静态 common headers。它不是浏览器运行时，也不能复制 JavaScript、Canvas、IP、Cookie 或行为特征。

### 当前 profile 边界

| Profile | Header / UA | HTTP/1.1、HTTP/2 TLS | HTTP/2 | HTTP/3 |
| --- | --- | --- | --- | --- |
| Chrome | 固定 Chrome 133；OS 主要改变 UA、Client Hints 与 Header | 固定 uTLS `HelloChrome_133` | Chrome-like SETTINGS、flow、priority | Go `crypto/tls` + quic-go 的 Chrome-like 配置 |
| Firefox | 固定 Firefox 120；OS 主要改变 UA 与 Header | 固定 uTLS `HelloFirefox_120` | Firefox-like SETTINGS、flow、stream ID、priority | Go `crypto/tls` + quic-go 的 Firefox-like 配置 |
| Safari | Safari 16.6 风格 Header/UA | uTLS `HelloSafari_16_0` | Safari-like SETTINGS、flow、priority | 没有专用 Safari H3 profile；需要时在其后显式配置 H3 |

指定 Android 或 iOS 不会自动把 Chrome/Firefox 的 TLS preset 变成另一套移动端 ClientHello；目前主要改变 UA、Client Hints 和 Header。`Auto` preset 可能随 uTLS 升级漂移，因此内置 Chrome、Firefox、Safari 已固定明确版本。

在发送第一个请求前完成 profile 配置。切换 profile 会清理它拥有的 Header/H2/H3 状态，但已经建立的连接无法被改写；需要切换身份时优先新建或 `Clone` 一个尚未使用的 client。

## 低层 TLS 指纹模式

| 目标 | API |
| --- | --- |
| 内置 preset | `SetTLSFingerprintChrome`、`SetTLSFingerprintFirefox`、`SetTLSFingerprintSafari`、Edge/QQ/360/iOS/Android 对应方法；前三者固定明确版本，其余按所用 uTLS 常量定义 |
| 兼容旧随机行为 | `SetTLSFingerprintRandomized` |
| 明确允许 ALPN/H2 | `SetTLSFingerprintRandomizedALPN` |
| 明确只模拟无 ALPN 的 H1 对端 | `SetTLSFingerprintRandomizedNoALPN` |
| 可复现随机指纹 | 两种 `SetTLSFingerprintRandomized*WithSeed` |
| uTLS ID | `SetTLSFingerprint` |
| 自定义 spec | `SetTLSFingerprintSpecFactory` |
| 捕获的 ClientHello | `ParseTLSClientHello` 后接 `SetTLSFingerprintSpecFactory` |

### 稳定随机指纹

uTLS 的随机指纹在没有 seed 时可为每条新连接重新生成。需要同一 client 内稳定复用时显式传 seed：

```go
seed, err := utls.NewPRNGSeed()
if err != nil {
	log.Fatal(err)
}

client := req.C().
	SetTLSFingerprintRandomizedALPNWithSeed(seed)
```

req 会复制 seed，调用后继续修改原值不会影响已经配置的 client。ALPN 版本适合允许 HTTP/2 协商的场景；NoALPN 版本用于明确只走 HTTP/1.1 的对端。

### 自定义 fresh spec

```go
client.SetTLSFingerprintSpecFactory(func() *utls.ClientHelloSpec {
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_133)
	if err != nil {
		panic(err)
	}
	return &spec
})
```

factory 必须每次返回全新且非 nil 的 spec。Transport 可并发建连，因此自定义 factory 也必须自行保护共享状态。nil factory 或 nil spec 会让握手返回明确错误，不会 panic。

### 从捕获的 ClientHello 创建 profile

```go
factory, err := req.ParseTLSClientHello(rawTLSRecord)
if err != nil {
	log.Fatal(err)
}

client := req.C().SetTLSFingerprintSpecFactory(factory)
```

解析器要求一条完整、未加密且未分片的 TLS ClientHello record，限制在 TLS plaintext 16 KiB 内；默认严格拒绝未知扩展、重复 session 扩展和任何捕获到的 `pre_shared_key` 扩展，并复制输入。返回的 factory 每次从新的私有字节副本解析，因此每次握手获得互不别名的 fresh spec，可安全并发使用。捕获数据不应被当作可信输入。

## `crypto/tls.Config` 兼容桥

启用 uTLS 后，req 会把 client 侧标准 TLS 配置转换到每条新连接的 uTLS config，包括：

- 显式 `ServerName`；未设置时从目标 host、`host:port` 或 IPv6 地址安全推导。
- `Certificates` 与 `GetClientCertificate`，支持 mTLS。
- `VerifyPeerCertificate` 与 `VerifyConnection`。
- `ClientSessionCache`，通过两边公开的 session 序列化 API 适配；不兼容编码安全降级为 cache miss。
- `Renegotiation`、key log 和 ECH 客户端配置。

`MinVersion`、`MaxVersion`、`CipherSuites`、`CurvePreferences`、`NextProtos` 也会先转换，但它们会影响 ClientHello 形状。浏览器、随机或自定义 uTLS spec 会按自身 cipher 列表和扩展重写这些值，以保持所选指纹；因此不能把标准配置里的这些字段视为 preset 下的强约束。需要严格限制版本、cipher、curve 或 ALPN 时，应选择/构造满足策略的 spec，或继续使用标准 TLS 路径。

`Client.Clone` 会把指纹握手重新绑定到 clone 自己的 TLS config，原 client 与 clone 可以使用不同 CA、SNI 和验证回调。

两个验证回调都会执行，且其错误会终止握手；但所有从 uTLS 转换出的标准 `tls.ConnectionState`（包括验证回调、trace、`Response.TLS` 和连接状态）都有同一边界：uTLS v1.8.2 没有公开 Go 1.26 的 `CurveID`、`HelloRetryRequest`，也无法重建 `ExportKeyingMaterial` 的私有 exporter。前两个字段保持零值；验证/ECH 回调内误调 exporter 会转为握手错误，而在响应、trace 或连接状态上直接调用该方法会触发标准库 nil exporter panic，因此不要调用。安全策略若依赖这些信息或 keying material，应继续使用标准 TLS，或通过 `SetTLSHandshake` 提供能无损返回标准状态的实现。

动态客户端证书回调能获得证书选择所需的公开字段；Go 没有公开构造 `CertificateRequestInfo` 私有 Context 的 API，因此该 Context 无法无损跨实现传递。服务端专用的 TLS callback 不属于 req 客户端握手路径。

session cache 适配不会凭空给 preset 增加真实 PSK extension。只有所选 ClientHello 本身支持真实恢复时才可能得到 `DidResume=true`；例如 `HelloGolang` 的 TLS 1.3 恢复已做真实双连接测试，而普通 Chrome 133 parrot preset 缺少真实 TLS 1.3 `PreSharedKeyExtension`，会安全跳过 TLS 1.3 恢复。这里不对 TLS 1.2 session ticket 行为作额外承诺。

`EnableInsecureSkipVerify` 只用于明确受控的测试。若使用它配合 `VerifyConnection` 做证书固定，必须保证回调返回错误时请求失败；本仓库已为 uTLS 路径加入真实握手回归测试。

## HTTP/3 的 TLS 边界

uTLS 指纹只作用于 HTTP/1.1 和 HTTP/2。HTTP/3 基于 quic-go 和 Go `crypto/tls`，使用：

- `SetHTTP3TLSChromeProfile`
- `SetHTTP3TLSFirefoxProfile`
- `SetHTTP3TLSClientConfig`
- `SetHTTP3QUICChromeProfile`

这些 API 可以调整 H3 TLS/QUIC 参数，但不能声称发出了与浏览器完全相同的 uTLS QUIC ClientHello。

## 读取服务端 TLS 信息

```go
info := resp.TLSInfo()
if info != nil {
	fmt.Println(info.Version)
	fmt.Println(info.FingerprintSHA256)
}
```

`TLSInfo` 返回证书主体、签发者、DNSNames、TLS 版本和 SHA-256 指纹；HTTP 明文响应返回 nil。

## 版本与验证

当前仓库使用 Go 1.26.7 与 uTLS v1.8.2。真实指纹仍应在授权环境中通过抓包或服务端观测验证，并同时查看 ALPN、JA4/Peetprint、HTTP/2 SETTINGS、Header 顺序和协议回退，不能只看 JA3 hash。

可编译配置示例见 [tls_fingerprint_test.go](examples/tls_fingerprint_test.go)，浏览器与 HTTP/3 组合见 [browser_http3_test.go](examples/browser_http3_test.go)。上游归属与许可证见 [上游项目、致谢与许可](16-upstream-credits.md)。
