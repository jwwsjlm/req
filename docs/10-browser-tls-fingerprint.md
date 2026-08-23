# 浏览器与 TLS 指纹

## 优先使用完整浏览器 profile

```go
client := req.C().ImpersonateChromeWithOS(req.BrowserOSWindows)
```

可选 OS：`BrowserOSWindows`、`BrowserOSMacOS`、`BrowserOSLinux`、`BrowserOSAndroid`、`BrowserOSIOS`、`BrowserOSRandom`。Chrome 和 Firefox 支持指定或随机 OS；Safari 使用 `ImpersonateSafari`。

完整 profile 会组合：

- User-Agent 和 method-aware 常见 Header。
- Header / pseudo Header 顺序。
- HTTP/2 SETTINGS、flow、priority 等配置。
- HTTP/1.1 和 HTTP/2 的 uTLS ClientHello。
- 相应的 HTTP/3 TLS/QUIC profile。

只设置 `User-Agent` 不会改变 TLS 或 HTTP/2 指纹。

## 低层 TLS 指纹 API

内置入口包括：

- `SetTLSFingerprintChrome`
- `SetTLSFingerprintFirefox`
- `SetTLSFingerprintSafari`
- `SetTLSFingerprintEdge`
- `SetTLSFingerprintIOS`、`SetTLSFingerprintAndroid`、`SetTLSFingerprintRandomized`
- `SetTLSFingerprintJA3`
- `SetTLSFingerprint`

自定义 uTLS spec 时优先使用 factory：

```go
client.SetTLSFingerprintSpecFactory(func() *utls.ClientHelloSpec {
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
	if err != nil {
		panic(err)
	}
	return &spec
})
```

factory 每次返回全新 spec。旧的 `SetTLSFingerprintSpec` 保留兼容，但 uTLS 可能在应用 preset 时修改对象，跨多个新连接复用同一 spec 容易产生问题。

## HTTP/3 的 TLS 边界

uTLS 指纹只作用于 HTTP/1.1 和 HTTP/2。HTTP/3 基于 quic-go 和 Go `crypto/tls`，使用：

- `SetHTTP3TLSChromeProfile`
- `SetHTTP3TLSFirefoxProfile`
- `SetHTTP3TLSClientConfig`
- `SetHTTP3QUICChromeProfile`

因此不能声称 HTTP/3 发出了与浏览器完全相同的 uTLS QUIC ClientHello。

## 证书与 TLS 配置

```go
client.SetTLSClientConfig(&tls.Config{
	MinVersion: tls.VersionTLS12,
})
```

根证书和客户端证书使用 `SetRootCertFromString`、`SetRootCertsFromFile`、`SetCertFromFile`、`SetCerts`。`EnableInsecureSkipVerify` 仅用于明确受控的测试环境，生产环境应验证证书和目标主机。

## 读取服务端 TLS 信息

```go
info := resp.TLSInfo()
if info != nil {
	fmt.Println(info.Version)
	fmt.Println(info.FingerprintSHA256)
}
```

`TLSInfo` 返回证书主体、签发者、DNSNames、TLS 版本和 SHA-256 指纹。HTTP 明文响应返回 nil。

## 正确认识“伪装”

指纹 profile 只能让协议与 Header 组合更接近目标浏览器，不能保证通过所有风控。服务端还可能结合 IP、Cookie、行为序列、时间特征、JavaScript 环境和账号信誉。应使用合法授权的目标，并以抓包或服务端观测验证实际结果。

HTTP/3 组合示例：[browser_http3_test.go](examples/browser_http3_test.go)。
