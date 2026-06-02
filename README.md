# req

这个仓库是我自用的 `req` 增强版，基于 [imroc/req](https://github.com/imroc/req) 做扩展，重点加强浏览器伪装、HTTP/3、TLS/JA3 指纹、HTTP/2/HTTP/3 细节控制和一些日常使用体验。

原版文档仍然可以参考：[https://req.cool](https://req.cool)

## 主要能力

- 简洁链式 API，保留 `req.C().R().Get(...)` 这种写法。
- 支持 HTTP/1.1、HTTP/2、HTTP/3，可以自动协商，也可以强制指定。
- 浏览器伪装增强：Chrome、Firefox、Safari，并支持不同系统 profile。
- 支持 uTLS TLS 指纹、JA3、自定义 `ClientHelloSpec`。
- HTTP/2 可控：SETTINGS、header order、pseudo header order、priority、initial stream id。
- HTTP/3 可控：SETTINGS、GREASE、Datagram、Extended CONNECT、QUICConfig、TLS profile、Alt-Svc 失败回退。
- HTTP/3 QUIC 性能 profile：token reuse、keepalive、窗口大小、初始包大小。
- 支持自定义 CookieJar factory，兼容 `func() http.CookieJar` 和旧的 `func() *cookiejar.Jar`。
- 保留 req 原有的 debug、dump、retry、download、upload、middleware、自动 JSON/XML 等能力。

## 安装

```sh
go get github.com/jwwsjlm/req/v3
```

要求 Go `1.24+`。

## 基础用法

```go
package main

import (
	"fmt"
	"log"

	"github.com/jwwsjlm/req/v3"
)

func main() {
	client := req.C()

	resp, err := client.R().
		SetHeader("Accept", "application/json").
		Get("https://httpbin.org/uuid")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.String())
}
```

## 统一 Client

自用时建议创建一个长期复用的 client，不要每次请求都重新建。

```go
var client = req.C().
	SetUserAgent("my-client").
	SetTimeout(10 * time.Second).
	SetCommonHeader("Accept-Language", "zh-CN,zh;q=0.9").
	EnableDumpEachRequest()
```

## JSON 请求和响应

```go
type Repo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Result struct {
	JSON Repo `json:"json"`
}

var result Result

resp, err := req.C().R().
	SetBody(&Repo{Name: "req", URL: "https://github.com/imroc/req"}).
	SetSuccessResult(&result).
	Post("https://httpbin.org/post")
if err != nil {
	log.Fatal(err)
}
if !resp.IsSuccessState() {
	log.Fatalf("bad status: %s", resp.Status)
}
```

## 错误处理

可以把服务端错误结构自动转换成 Go error。

```go
type ErrorMessage struct {
	Message string `json:"message"`
}

func (e *ErrorMessage) Error() string {
	return e.Message
}

client := req.C().
	SetCommonErrorResult(&ErrorMessage{}).
	OnAfterResponse(func(client *req.Client, resp *req.Response) error {
		if resp.Err != nil {
			return nil
		}
		if errMsg, ok := resp.ErrorResult().(*ErrorMessage); ok {
			resp.Err = errMsg
			return nil
		}
		if !resp.IsSuccessState() {
			resp.Err = fmt.Errorf("bad status: %s\n%s", resp.Status, resp.Dump())
		}
		return nil
	})
```

## DevMode 和 Dump

调试接口时直接开：

```go
client := req.C().DevMode()
resp, err := client.R().Get("https://httpbin.org/get")
```

只在出错时 dump：

```go
resp, err := req.C().R().
	EnableDump().
	Get("https://api.example.com/data")
if err != nil {
	fmt.Println(resp.Dump())
}
```

## 浏览器伪装

Chrome 默认使用 macOS profile，也可以指定系统。

```go
client := req.C().
	ImpersonateChromeWithOS(req.BrowserOSWindows)

resp, err := client.R().Get("https://example.com")
```

支持的系统：

```go
req.BrowserOSWindows
req.BrowserOSMacOS
req.BrowserOSLinux
req.BrowserOSAndroid
req.BrowserOSIOS
```

Firefox：

```go
client := req.C().
	ImpersonateFirefoxWithOS(req.BrowserOSLinux)
```

Safari：

```go
client := req.C().
	ImpersonateSafari()
```

内置 profile 会一起设置：

- TLS 指纹，作用于 HTTP/1.1 和 HTTP/2。
- HTTP/2 SETTINGS、flow、priority、pseudo header order。
- method-aware headers：GET/POST 会使用不同的浏览器请求头。
- HTTP/3 SETTINGS、TLS profile、QUIC profile。

## JA3 和自定义 TLS 指纹

JA3：

```go
ja3 := "771,4865-4866-4867-49195-49199,0-5-10-11-13-16-43-51,29-23-24,0"
ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0"

client := req.C().
	SetTLSFingerprintJA3(ja3, ua, false)
```

自定义 uTLS spec：

```go
spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
if err != nil {
	panic(err)
}

client := req.C().
	SetTLSFingerprintSpec(&spec)
```

注意：`SetTLSFingerprint*`、JA3、自定义 uTLS 只作用于 HTTP/1.1 和 HTTP/2。HTTP/3 使用 quic-go 和 Go 的 `crypto/tls`，不能假装成 uTLS QUIC ClientHello。

## HTTP/3 常用组合

自动启用 HTTP/3，并允许 Alt-Svc 探测到的 H3 失败后回退到 H2/H1：

```go
client := req.C().
	EnableHTTP3().
	EnableHTTP3FallbackOnError().
	SetHTTP3AltSvcFailureCooldown(30 * time.Second)
```

强制 HTTP/3：

```go
client := req.C().
	EnableForceHTTP3()
```

强制 HTTP/3，同时失败回退：

```go
client := req.C().
	EnableHTTP3FallbackOnError().
	EnableForceHTTP3()
```

Chrome 风格 HTTP/3：

```go
client := req.C().
	ImpersonateChromeWithOS(req.BrowserOSWindows).
	SetHTTP3TLSChromeProfile().
	SetHTTP3QUICChromeProfile().
	EnableHTTP3FallbackOnError().
	EnableForceHTTP3()
```

## HTTP/3 高级控制

```go
client := req.C().
	SetHTTP3TLSChromeProfile().
	SetHTTP3QUICChromeProfile().
	SetHTTP3AdditionalSetting(req.HTTP3SettingQpackMaxTableCapacity, 65536).
	SetHTTP3AdditionalSetting(req.HTTP3SettingQpackBlockedStreams, 100).
	SetHTTP3MaxResponseHeaderBytes(262144).
	EnableHTTP3Datagrams().
	EnableHTTP3ExtendedConnect().
	SetHTTP3Grease().
	EnableHTTP3FallbackOnError().
	SetHTTP3AltSvcFailureCooldown(30 * time.Second).
	EnableForceHTTP3()
```

自定义 HTTP/3 TLS：

```go
client := req.C().
	SetHTTP3TLSClientConfig(&tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		NextProtos: []string{"h3"},
	})
```

自定义 QUIC：

```go
client := req.C().
	SetHTTP3QUICConfig(&quic.Config{
		HandshakeIdleTimeout: 5 * time.Second,
		MaxIdleTimeout:       45 * time.Second,
		KeepAlivePeriod:      15 * time.Second,
		TokenStore:           quic.NewLRUTokenStore(256, 4),
	})
```

使用内置性能配置：

```go
client := req.C().
	SetHTTP3QUICPerformanceProfile().
	EnableHTTP3()
```

## HTTP/2 高级控制

```go
client := req.C().
	SetHTTP2SettingsFrame(
		http2.Setting{
			ID:  http2.SettingHeaderTableSize,
			Val: 65536,
		},
		http2.Setting{
			ID:  http2.SettingInitialWindowSize,
			Val: 6291456,
		},
	).
	SetHTTP2ConnectionFlow(15663105).
	SetHTTP2InitialStreamID(3)
```

## CookieJar Factory

支持标准 `http.CookieJar`：

```go
client := req.C().
	SetCookieJarFactory(func() http.CookieJar {
		jar, _ := cookiejar.New(nil)
		return jar
	})
```

也兼容旧写法：

```go
client := req.C().
	SetCookieJarFactory(func() *cookiejar.Jar {
		jar, _ := cookiejar.New(nil)
		return jar
	})
```

## 文件上传

```go
resp, err := req.C().R().
	SetFile("file", "./demo.txt").
	Post("https://httpbin.org/post")
```

## 文件下载

```go
resp, err := req.C().R().
	SetOutputFile("./out.bin").
	Get("https://example.com/file.bin")
```

## 推荐自用模板

```go
func NewHTTPClient() *req.Client {
	return req.C().
		SetTimeout(30 * time.Second).
		ImpersonateChromeWithOS(req.BrowserOSWindows).
		EnableHTTP3().
		EnableHTTP3FallbackOnError().
		SetHTTP3AltSvcFailureCooldown(30 * time.Second).
		SetCommonRetryCount(2).
		EnableDumpEachRequest()
}
```

## 测试说明

本仓库在 Windows 下跑 `go test ./...` 时，目前有两个已知旧问题：

- `TestTraceInfo`
- `TestSetFile`：Windows 错误文案是 `The system cannot find the file specified.`，不是 `no such file`

本次增强相关的定向测试、`internal/http2`、`internal/http3` 都已通过。

## 致谢

- 感谢 [imroc/req](https://github.com/imroc/req)，这个库的基础能力来自原项目。
- 感谢 [enetx/surf](https://github.com/enetx/surf)，HTTP/3 tuning、现代浏览器 profile、TLS impersonation 等思路给了很多参考。

## License

MIT，见 [LICENSE](LICENSE)。
