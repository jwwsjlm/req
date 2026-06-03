# req

这个仓库是我自用的 `req` 增强版，基于 [imroc/req](https://github.com/imroc/req) 做扩展，重点加强浏览器伪装、HTTP/3、TLS/JA3 指纹、HTTP/2/HTTP/3 细节控制和一些日常使用体验。

原版文档仍然可以参考：[https://req.cool](https://req.cool)

第一次接触 Go 或第一次用这个库，建议先看：[示例.md](示例.md)。那份文档从 `go mod init`、`go get`、第一个 `GET` 请求开始，一直写到 JSON、表单、Cookie、上传下载、代理、重试、浏览器伪装、HTTP/3 和常见报错。

## 主要能力

- 简洁链式 API，保留 `req.C().R().Get(...)` 这种写法。
- 支持 HTTP/1.1、HTTP/2、HTTP/3，可以自动协商，也可以强制指定。
- 浏览器伪装增强：Chrome、Firefox、Safari，并支持不同系统 profile。
- 支持 uTLS TLS 指纹、JA3、自定义 `ClientHelloSpec`。
- HTTP/2 可控：SETTINGS、header order、pseudo header order、priority、initial stream id。
- HTTP/3 可控：SETTINGS、GREASE、Datagram、Extended CONNECT、QUICConfig、TLS profile、Alt-Svc 失败回退。
- HTTP/3 QUIC 性能 profile：token reuse、keepalive、窗口大小、初始包大小。
- 支持自定义 DNS resolver 和 DNS-over-TLS，HTTP/1.1、HTTP/2、HTTP/3 共用同一套解析策略。
- 支持从响应中提取 TLS 版本、证书信息和 SHA-256 指纹。
- 支持自定义 CookieJar factory，兼容 `func() http.CookieJar` 和旧的 `func() *cookiejar.Jar`。
- 请求构造补强：Any 类型参数、多值 Header、Raw Path 参数、带 Content-Type 的 multipart field、显式 Content-Length。
- 保留 req 原有的 debug、dump、retry、download、upload、middleware、自动 JSON/XML 等能力。

## 方法速查

| 场景 | 常用方法 |
| --- | --- |
| 创建请求 | `C()`、`R()`、`Get()`、`Post()`、`Put()`、`Patch()`、`Delete()` |
| URL 参数 | `SetBaseURL`、`SetScheme`、`SetQueryParamAny`、`SetQueryParamsFromStruct`、`SetQueryString`、`SetPathRawParam` |
| Header/Cookie | `SetHeaderAny`、`SetHeaderValues`、`SetHeaders`、`SetCookies`、`SetCookieJarFactory` |
| Body | `SetBody`、`SetContentLength`、`SetBodyJsonMarshal`、`SetBodyXmlMarshal`、`SetFormData`、`SetOrderedFormData` |
| 结果解析 | `SetSuccessResult`、`SetErrorResult`、`Into`、`UnmarshalJson`、`UnmarshalXml`、`ToString`、`ToBytes` |
| 错误处理 | `SetCommonErrorResult`、`SetResultStateCheckFunc`、`OnError`、`OnAfterResponse` |
| 重试 | `SetCommonRetryCount`、`SetCommonRetryBackoffInterval`、`SetRetryCount`、`SetRetryCondition` |
| 调试 | `DevMode`、`EnableDump`、`EnableDumpEachRequest`、`EnableTraceAll`、`EnableTrace`、`TraceInfo` |
| 浏览器伪装 | `ImpersonateChromeWithOS`、`ImpersonateChromeRandomOS`、`ImpersonateFirefoxWithOS`、`ImpersonateSafari` |
| TLS 指纹 | `SetTLSFingerprintJA3`、`SetTLSFingerprintSpec`、`SetTLSFingerprintChrome` |
| DNS | `SetDNSResolver`、`SetDNSOverTLSCloudflare`、`SetDNSOverTLS` |
| HTTP/2 | `EnableForceHTTP2`、`SetHTTP2SettingsFrame`、`SetHTTP2InitialStreamID` |
| HTTP/3 | `EnableHTTP3`、`EnableForceHTTP3`、`EnableHTTP3FallbackOnError`、`SetHTTP3QUICPerformanceProfile` |
| 上传下载 | `SetFile`、`SetFiles`、`SetFileBytes`、`SetMultipartField`、`SetOutputFile`、`SetOutputDirectory`、`NewParallelDownload` |
| 扩展集成 | `GetClient`、`GetTransport`、`Do`、`WrapRoundTripFunc`、`SetResponseBodyTransformer` |

## 安装

```sh
go get github.com/jwwsjlm/req/v3
```

要求 Go `1.24+`。

如果你还不熟 Go module、`main.go`、`go run` 这些基础步骤，先按 [示例.md](示例.md) 跑一遍最小项目。

## 推荐使用方式

普通 API 调用建议长期复用一个 client：

```go
var apiClient = req.C().
	SetTimeout(30 * time.Second).
	SetCommonHeader("Accept", "application/json").
	SetCommonHeader("Accept-Language", "zh-CN,zh;q=0.9").
	SetCommonRetryCount(2).
	SetCommonRetryBackoffInterval(300*time.Millisecond, 3*time.Second)
```

偏浏览器访问、反爬压测、站点抓取时用浏览器 profile：

```go
var browserClient = req.C().
	ImpersonateChromeWithOS(req.BrowserOSWindows).
	SetDNSOverTLSCloudflare().
	EnableHTTP3().
	EnableHTTP3FallbackOnError().
	SetHTTP3AltSvcFailureCooldown(30 * time.Second).
	SetCommonRetryCount(2)
```

只想稳定优先，不想强制 HTTP/3：

```go
var stableClient = req.C().
	SetTimeout(20 * time.Second).
	EnableHTTP3().
	EnableHTTP3FallbackOnError().
	SetCommonRetryCount(2)
```

调试时再开 dump，不建议生产默认全量 dump：

```go
client := req.C().
	DevMode().
	EnableDumpEachRequestWithoutBody()
```

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

也可以直接从 client 创建不同方法的请求：

```go
resp := client.Get("https://httpbin.org/get").
	SetQueryParam("q", "req").
	Do()
if resp.Err != nil {
	log.Fatal(resp.Err)
}
```

常用方法：

```go
client.Get(url)
client.Post(url)
client.Put(url)
client.Patch(url)
client.Delete(url)
client.Head(url)
client.Options(url)
```

## BaseURL、Scheme 和公共参数

写 API client 时推荐把公共配置放到 client 上，单个请求只写相对路径和差异参数。

```go
type SearchParams struct {
	Q    string   `url:"q"`
	Page int      `url:"page"`
	Tags []string `url:"tag"`
}

client := req.C().
	SetBaseURL("https://api.example.com").
	SetCommonHeader("Accept", "application/json").
	SetCommonQueryParam("locale", "zh-CN").
	SetCommonPathParam("version", "v1")

resp, err := client.R().
	SetPathParam("id", "42").
	SetQueryParamsFromStruct(SearchParams{
		Q:    "req",
		Page: 1,
		Tags: []string{"go", "http"},
	}).
	Get("/{version}/users/{id}")
```

参数值不是字符串时，可以直接用 `Any` 版本：

```go
resp, err := client.R().
	SetQueryParamAny("page", 2).
	SetPathParamAny("id", 42).
	Get("/v1/users/{id}")
```

默认 `SetPathParam` 会做 `url.PathEscape`。如果路径参数本身就包含 `/`，并且你希望保留它：

```go
resp, err := client.R().
	SetPathRawParam("path", "groups/developers").
	Get("/v1/files/{path}")
```

如果传入的是没有 scheme 的完整域名，可以给 client 设置默认 scheme：

```go
client := req.C().
	SetScheme("https")

resp, err := client.R().Get("example.com/api")
```

已经有 `url.Values` 时可以直接复用：

```go
values := url.Values{}
values.Add("tag", "go")
values.Add("tag", "http")

resp, err := client.R().
	SetQueryParamsFromValues(values).
	Get("https://api.example.com/search")
```

## 请求构造

Query 参数：

```go
resp, err := client.R().
	SetQueryParam("page", "1").
	SetQueryParams(map[string]string{
		"sort": "created",
		"q":    "req",
	}).
	Get("https://api.example.com/repos")
```

追加同名参数：

```go
resp, err := client.R().
	AddQueryParams("tag", "go", "http").
	Get("https://api.example.com/search")
```

原始 query string：

```go
resp, err := client.R().
	SetQueryString("page=1&tag=go&tag=http").
	Get("https://api.example.com/search")
```

Path 参数：

```go
resp, err := client.R().
	SetPathParam("owner", "jwwsjlm").
	SetPathParam("repo", "req").
	Get("https://api.example.com/repos/{owner}/{repo}")
```

Header 和 Cookie：

```go
resp, err := client.R().
	SetHeader("X-Request-ID", "demo").
	SetHeaders(map[string]string{
		"Accept": "application/json",
	}).
	SetCookies(&http.Cookie{Name: "sid", Value: "xxx"}).
	Get("https://api.example.com/me")
```

Header 值不是字符串，或一个 Header 有多个值：

```go
resp, err := client.R().
	SetHeaderAny("X-Retry", 2).
	SetHeaderValues("Accept", "application/json", "application/problem+json").
	Get("https://api.example.com/me")
```

Form 表单：

```go
resp, err := client.R().
	SetFormData(map[string]string{
		"username": "demo",
		"password": "secret",
	}).
	Post("https://example.com/login")
```

需要固定字段顺序的表单：

```go
resp, err := client.R().
	SetOrderedFormData(
		"username", "demo",
		"password", "secret",
		"otp", "123456",
	).
	Post("https://example.com/login")
```

原始 body：

```go
resp, err := client.R().
	SetContentType("text/plain").
	SetBodyString("hello").
	Post("https://httpbin.org/post")
```

需要手动指定 `Content-Length` 时：

```go
resp, err := client.R().
	SetBody(strings.NewReader("hello")).
	SetContentLength(5).
	Post("https://api.example.com/raw")
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
	SetBody(&Repo{Name: "req", URL: "https://github.com/jwwsjlm/req"}).
	SetSuccessResult(&result).
	Post("https://httpbin.org/post")
if err != nil {
	log.Fatal(err)
}
if !resp.IsSuccessState() {
	log.Fatalf("bad status: %s", resp.Status)
}
```

只想发 JSON 字符串：

```go
resp, err := client.R().
	SetBodyJsonString(`{"name":"req"}`).
	Post("https://httpbin.org/post")
```

自定义 JSON 编解码器：

```go
client := req.C().
	SetJsonMarshal(json.Marshal).
	SetJsonUnmarshal(json.Unmarshal)
```

手动读取响应：

```go
text, err := resp.ToString()
body, err := resp.ToBytes()
```

自动反序列化：

```go
var out struct {
	Origin string `json:"origin"`
}

resp, err := client.R().
	SetSuccessResult(&out).
	Get("https://httpbin.org/ip")
```

## XML 请求和响应

```go
type UserXML struct {
	XMLName xml.Name `xml:"user"`
	Name    string   `xml:"name"`
}

resp, err := client.R().
	SetBodyXmlMarshal(UserXML{Name: "req"}).
	Post("https://api.example.com/users")
if err != nil {
	log.Fatal(err)
}

var out UserXML
if err := resp.UnmarshalXml(&out); err != nil {
	log.Fatal(err)
}
```

也可以直接发送 XML 字符串：

```go
resp, err := client.R().
	SetBodyXmlString(`<user><name>req</name></user>`).
	Post("https://api.example.com/users")
```

自定义 XML 编解码器：

```go
client := req.C().
	SetXmlMarshal(xml.Marshal).
	SetXmlUnmarshal(xml.Unmarshal)
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

请求级错误结果：

```go
var errBody ErrorMessage

resp, err := client.R().
	SetErrorResult(&errBody).
	Get("https://api.example.com/data")
if err != nil {
	log.Fatal(err)
}
if resp.IsErrorState() {
	log.Printf("api error: %+v", errBody)
}
```

自定义哪些状态码算成功或错误：

```go
client := req.C().
	SetResultStateCheckFunc(func(resp *req.Response) req.ResultState {
		if resp.StatusCode == http.StatusNotModified {
			return req.SuccessState
		}
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return req.SuccessState
		}
		if resp.StatusCode >= 400 {
			return req.ErrorState
		}
		return req.UnknownState
	})
```

## 认证

Bearer：

```go
resp, err := client.R().
	SetBearerAuthToken("token").
	Get("https://api.example.com/me")
```

Bearer token 也可以用更短的写法：

```go
resp, err := client.R().
	SetAuthToken("token").
	Get("https://api.example.com/me")
```

自定义认证 scheme：

```go
resp, err := client.R().
	SetAuthSchemeToken("OAuth", "token").
	Get("https://api.example.com/me")
```

Basic：

```go
resp, err := client.R().
	SetBasicAuth("user", "pass").
	Get("https://api.example.com/me")
```

Digest：

```go
client := req.C().
	SetCommonDigestAuth("user", "pass")
```

## 超时、Context 和重试

全局超时：

```go
client := req.C().
	SetTimeout(15 * time.Second)
```

请求级 context：

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

resp, err := client.R().
	SetContext(ctx).
	Get("https://api.example.com/slow")
```

推荐重试配置：

```go
client := req.C().
	SetCommonRetryCount(2).
	SetCommonRetryBackoffInterval(300*time.Millisecond, 3*time.Second).
	SetCommonRetryCondition(func(resp *req.Response, err error) bool {
		if err != nil {
			return true
		}
		return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
	})
```

单个请求覆盖重试：

```go
resp, err := client.R().
	SetRetryCount(3).
	SetRetryFixedInterval(time.Second).
	Get("https://api.example.com/flaky")
```

## 代理和重定向

HTTP/HTTPS/SOCKS5 代理：

```go
client := req.C().
	SetProxyURL("http://127.0.0.1:7890")

client = req.C().
	SetProxyURL("socks5://127.0.0.1:1080")
```

自定义代理逻辑：

```go
client := req.C().
	SetProxy(func(r *http.Request) (*url.URL, error) {
		if strings.HasSuffix(r.URL.Hostname(), ".internal") {
			return nil, nil
		}
		return url.Parse("http://127.0.0.1:7890")
	})
```

重定向策略：

```go
client := req.C().
	SetRedirectPolicy(
		req.MaxRedirectPolicy(5),
		req.SameDomainRedirectPolicy(),
	)
```

不跟随重定向：

```go
client := req.C().
	SetRedirectPolicy(req.NoRedirectPolicy())
```

## Middleware

请求前统一加签名、日志、动态 header：

```go
client := req.C().
	OnBeforeRequest(func(c *req.Client, r *req.Request) error {
		r.SetHeader("X-Token", "token")
		return nil
	})
```

响应后统一处理错误：

```go
client := req.C().
	OnAfterResponse(func(c *req.Client, resp *req.Response) error {
		if resp.Err != nil {
			return nil
		}
		if resp.StatusCode >= 500 {
			resp.Err = fmt.Errorf("server error: %s", resp.Status)
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

## TraceInfo 性能排查

需要看 DNS、TCP、TLS、首包和响应耗时时打开 trace。`DevMode()` 会自动启用 trace，生产中建议按需开启。

```go
client := req.C().
	EnableTraceAll()

resp, err := client.R().Get("https://example.com")
if err != nil {
	log.Fatal(err)
}

trace := resp.TraceInfo()
fmt.Println(trace)
fmt.Println(trace.Blame())
fmt.Println(trace.TotalTime, trace.DNSLookupTime, trace.TLSHandshakeTime)
```

也可以只给单次请求开启：

```go
resp, err := client.R().
	EnableTrace().
	Get("https://example.com")
```

注意：当前 HTTP/3 不支持 `TraceInfo`，排查 H3 建议同时打开 dump、debug log 或回退到 H2 对照。

## 标准库兼容和扩展点

拿到底层 `*http.Client`：

```go
client := req.C().
	SetTimeout(10 * time.Second)

httpClient := client.GetClient()
```

直接执行标准库 `*http.Request`：

```go
client := req.C()

rawReq, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
if err != nil {
	log.Fatal(err)
}
rawReq.Header.Set("Accept", "application/json")

rawResp, err := client.Do(rawReq)
if err != nil {
	log.Fatal(err)
}
defer rawResp.Body.Close()
```

`Do(*http.Request)` 是标准库直通，会复用底层 client/transport，但不会自动套用 `R()` 的 query、body、结果解析和 req 级 middleware。

包装 req 级 round trip，适合统一埋点、日志、限流：

```go
client := req.C().
	WrapRoundTripFunc(func(rt req.RoundTripper) req.RoundTripFunc {
		return func(r *req.Request) (*req.Response, error) {
			start := time.Now()
			resp, err := rt.RoundTrip(r)
			log.Printf("%s %s cost=%s err=%v", r.Method, r.RawURL, time.Since(start), err)
			return resp, err
		}
	})
```

包装底层 `http.RoundTripper`，适合跟只认识标准库的组件对接：

```go
client.GetTransport().
	WrapRoundTripFunc(func(rt http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			r.Header.Set("X-From", "req")
			return rt.RoundTrip(r)
		}
	})
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
req.BrowserOSRandom
```

随机系统 profile：

```go
client := req.C().
	ImpersonateChromeRandomOS()
```

Firefox：

```go
client := req.C().
	ImpersonateFirefoxWithOS(req.BrowserOSLinux)
```

Firefox 也可以随机系统：

```go
client := req.C().
	ImpersonateFirefoxRandomOS()
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

## TLS、证书和安全开关

跳过证书校验，仅建议本地测试或明确知道风险时用：

```go
client := req.C().
	EnableInsecureSkipVerify()
```

自定义根证书：

```go
client := req.C().
	SetRootCertsFromFile("./ca.pem")
```

客户端证书：

```go
client := req.C().
	SetCertFromFile("./client.pem", "./client-key.pem")
```

完全自定义 TLS config：

```go
client := req.C().
	SetTLSClientConfig(&tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: "example.com",
	})
```

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

## 协议选择和特殊传输

强制 HTTP/1.1：

```go
client := req.C().
	EnableForceHTTP1()
```

强制 HTTP/2：

```go
client := req.C().
	EnableForceHTTP2()
```

H2C，也就是明文 HTTP/2：

```go
client := req.C().
	EnableH2C()
```

Unix Socket：

```go
client := req.C().
	SetUnixSocket("/var/run/demo.sock")
```

自定义 dial：

```go
client := req.C().
	SetDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	})
```

## 压缩、解码和响应读取

自动解压 gzip/deflate/br/zstd：

```go
client := req.C().
	EnableAutoDecompress()
```

禁用自动解压：

```go
client := req.C().
	DisableAutoDecompress()
```

自动把非 UTF-8 文本转成 UTF-8 默认开启；如果想自己处理：

```go
client := req.C().
	DisableAutoDecode()
```

只对指定 Content-Type 自动转码：

```go
client := req.C().
	SetAutoDecodeContentType("text", "html")
```

自己决定哪些响应需要转码：

```go
client := req.C().
	SetAutoDecodeContentTypeFunc(func(contentType string) bool {
		return strings.Contains(contentType, "text/") ||
			strings.Contains(contentType, "json")
	})
```

统一改写响应 body，适合解包、解密、去 BOM、兼容非标准 API：

```go
client := req.C().
	SetResponseBodyTransformer(func(rawBody []byte, r *req.Request, resp *req.Response) ([]byte, error) {
		return bytes.TrimSpace(rawBody), nil
	})
```

大响应不想自动读入内存：

```go
resp, err := req.C().R().
	DisableAutoReadResponse().
	Get("https://example.com/large")
if err != nil {
	log.Fatal(err)
}
defer resp.Body.Close()
```

## Cookie 使用

默认 `req.C()` 会启用内存 CookieJar。只要复用同一个 client，服务端 `Set-Cookie` 会自动保存，后续同域名请求会自动带上 Cookie。

```go
package main

import (
	"fmt"
	"log"

	"github.com/jwwsjlm/req/v3"
)

func main() {
	client := req.C()

	_, err := client.R().
		SetBodyJsonString(`{"username":"demo","password":"secret"}`).
		Post("https://example.com/login")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.R().Get("https://example.com/me")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.String())
}
```

手动给单次请求设置 Cookie：

```go
resp, err := client.R().
	SetCookies(
		&http.Cookie{Name: "sid", Value: "xxx"},
		&http.Cookie{Name: "theme", Value: "dark"},
	).
	Get("https://example.com/me")
```

给 client 设置公共 Cookie：

```go
client := req.C().
	SetCommonCookies(
		&http.Cookie{Name: "locale", Value: "zh-CN"},
	)
```

读取当前 Cookie：

```go
cookies, err := client.GetCookies("https://example.com")
if err != nil {
	log.Fatal(err)
}
for _, cookie := range cookies {
	fmt.Println(cookie.Name, cookie.Value)
}
```

清空 Cookie：

```go
client.ClearCookies()
```

自定义 CookieJar：

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

如果你想禁用自动 Cookie：

```go
client := req.C().
	SetCookieJar(nil)
```

`Clone()` 时要注意：

- `SetCookieJarFactory`：clone 后会重新创建 CookieJar，适合每个账号/任务隔离 Cookie。
- `SetCookieJar`：clone 后会共享同一个 CookieJar，适合多个 client 共用同一登录态。

多账号推荐这样写：

```go
func NewAccountClient() *req.Client {
	return req.C().
		SetCookieJarFactory(func() http.CookieJar {
			jar, _ := cookiejar.New(nil)
			return jar
		}).
		ImpersonateChromeWithOS(req.BrowserOSWindows)
}

accountA := NewAccountClient()
accountB := NewAccountClient()
```

## 文件上传

```go
resp, err := req.C().R().
	SetFile("file", "./demo.txt").
	Post("https://httpbin.org/post")
```

多文件：

```go
resp, err := req.C().R().
	SetFiles(map[string]string{
		"avatar": "./avatar.png",
		"doc":    "./demo.pdf",
	}).
	Post("https://httpbin.org/post")
```

内存内容上传：

```go
resp, err := req.C().R().
	SetFileBytes("file", "demo.txt", []byte("hello")).
	Post("https://httpbin.org/post")
```

Reader 上传：

```go
file, err := os.Open("./demo.txt")
if err != nil {
	log.Fatal(err)
}
defer file.Close()

resp, err := req.C().R().
	SetFileReader("file", "demo.txt", file).
	Post("https://httpbin.org/post")
```

Reader 上传时指定单个 part 的 Content-Type：

```go
resp, err := req.C().R().
	SetMultipartField(
		"manifest",
		"manifest.json",
		"application/json",
		strings.NewReader(`{"name":"demo"}`),
	).
	Post("https://httpbin.org/post")
```

上传进度：

```go
resp, err := req.C().R().
	SetUploadCallback(func(info req.UploadInfo) {
		fmt.Println(info.UploadedSize, info.FileSize)
	}).
	SetFile("file", "./big.bin").
	Post("https://httpbin.org/post")
```

## 文件下载

```go
resp, err := req.C().R().
	SetOutputFile("./out.bin").
	Get("https://example.com/file.bin")
```

下载到 writer：

```go
var buf bytes.Buffer

resp, err := req.C().R().
	SetOutput(&buf).
	Get("https://example.com/file.bin")
```

下载进度：

```go
resp, err := req.C().R().
	SetDownloadCallback(func(info req.DownloadInfo) {
		fmt.Println(info.DownloadedSize, info.Response.ContentLength)
	}).
	SetOutputFile("./out.bin").
	Get("https://example.com/file.bin")
```

统一下载目录：

```go
client := req.C().
	SetOutputDirectory("./downloads")

resp, err := client.R().
	SetOutputFile("file.bin").
	Get("https://example.com/file.bin")
```

并行分片下载，适合服务端支持 `Range`，并且 `HEAD` 能返回 `Content-Length` 的大文件：

```go
err := req.C().
	SetOutputDirectory("./downloads").
	NewParallelDownload("https://example.com/big.zip").
	SetOutputFile("big.zip").
	SetConcurrency(8).
	SetSegmentSize(16 * 1024 * 1024).
	Do()
if err != nil {
	log.Fatal(err)
}
```

## 推荐自用模板

```go
func NewHTTPClient() *req.Client {
	return req.C().
		SetTimeout(30 * time.Second).
		ImpersonateChromeWithOS(req.BrowserOSWindows).
		SetDNSOverTLSCloudflare().
		EnableHTTP3().
		EnableHTTP3FallbackOnError().
		SetHTTP3AltSvcFailureCooldown(30 * time.Second).
		SetCommonRetryCount(2).
		EnableDumpEachRequest()
}
```

## DNS-over-TLS 和自定义 Resolver

直接使用内置 DoT provider：

```go
client := req.C().
	SetDNSOverTLSCloudflare()
```

也可以指定自己的 DoT 上游：

```go
client := req.C().
	SetDNSOverTLS(req.DNSOverTLSProvider{
		ServerName: "dns.example.com",
		Addresses:  []string{"203.0.113.10:853"},
	})
```

如果你已经有自己的 resolver，也可以直接塞进去：

```go
resolver := &net.Resolver{PreferGo: true}

client := req.C().
	SetDNSResolver(resolver)
```

## TLS 信息

```go
resp, err := req.C().R().Get("https://example.com")
if err != nil {
	log.Fatal(err)
}

tlsInfo := resp.TLSInfo()
if tlsInfo != nil {
	fmt.Println(tlsInfo.Version)
	fmt.Println(tlsInfo.FingerprintSHA256)
	fmt.Println(tlsInfo.FingerprintSHA256OpenSSL)
}
```

## HTTP 指纹正确使用

HTTP 指纹不是只改 `User-Agent`。真实浏览器访问时，服务端通常会同时看到：

- TLS 指纹：JA3、JA4、扩展顺序、cipher suites、ALPN。
- HTTP/2 指纹：SETTINGS、WINDOW_UPDATE、pseudo header order、header order、priority。
- Header 组合：`sec-ch-ua`、`sec-fetch-*`、`accept-language`、`accept-encoding`、`priority` 等。
- Cookie 行为：登录态、同域自动携带、跳转后的 Cookie 更新。
- 协议选择：HTTP/2、HTTP/3、Alt-Svc 回退。

推荐优先使用内置浏览器 profile：

```go
client := req.C().
	ImpersonateChromeWithOS(req.BrowserOSWindows).
	SetDNSOverTLSCloudflare().
	EnableHTTP3().
	EnableHTTP3FallbackOnError().
	SetHTTP3AltSvcFailureCooldown(30 * time.Second)
```

Firefox：

```go
client := req.C().
	ImpersonateFirefoxWithOS(req.BrowserOSWindows).
	EnableHTTP3().
	EnableHTTP3FallbackOnError()
```

随机系统 profile：

```go
client := req.C().
	ImpersonateChromeRandomOS()
```

登录型站点建议配合 CookieJar，保持一个 client 对应一个账号：

```go
func NewBrowserSession() *req.Client {
	return req.C().
		SetCookieJarFactory(func() http.CookieJar {
			jar, _ := cookiejar.New(nil)
			return jar
		}).
		ImpersonateChromeWithOS(req.BrowserOSWindows).
		EnableHTTP3().
		EnableHTTP3FallbackOnError()
}
```

只设置 UA 不够：

```go
client := req.C().
	SetUserAgent("Mozilla/5.0 ... Chrome/133.0.0.0 ...")
```

上面这种只会改 header，TLS/HTTP2 指纹仍然不像浏览器。需要自己细调时，至少要组合这些方法：

```go
client := req.C().
	SetTLSFingerprintChrome().
	SetCommonHeaderOrder(
		"sec-ch-ua",
		"sec-ch-ua-mobile",
		"sec-ch-ua-platform",
		"user-agent",
		"accept",
		"sec-fetch-site",
		"sec-fetch-mode",
		"sec-fetch-dest",
		"accept-encoding",
		"accept-language",
	).
	SetCommonPseudoHeaderOder(":method", ":authority", ":scheme", ":path").
	SetHTTP2ConnectionFlow(15663105)
```

更推荐直接用 `ImpersonateChromeWithOS`，因为它会把 TLS、HTTP/2、HTTP/3、header、multipart boundary 一起配置好。

## 指纹测试

可以用 [tls.peet.ws/api/all](https://tls.peet.ws/api/all) 检查当前请求的 TLS、JA3/JA4、HTTP/2 Akamai 指纹和请求头。

最小测试代码：

```go
const endpoint = "https://tls.peet.ws/api/all"

clients := map[string]*req.Client{
	"default": req.C(),
	"chrome": req.C().
		ImpersonateChromeWithOS(req.BrowserOSWindows),
	"firefox": req.C().
		ImpersonateFirefoxWithOS(req.BrowserOSWindows),
}

for name, client := range clients {
	resp, err := client.R().
		SetHeader("Accept", "application/json").
		Get(endpoint)
	if err != nil {
		log.Println(name, err)
		continue
	}
	fmt.Println(name, resp.String())
}
```

我在 `2026-06-03` 本机跑到的结果摘要：

| 模式 | HTTP | User-Agent | JA4 | Peetprint Hash | HTTP/2 Akamai Hash |
| --- | --- | --- | --- | --- | --- |
| default | h2 | `req/v3 (https://github.com/jwwsjlm/req)` | `t13d1312h1_f57a46bbacb6_e5728521abd4` | `45373699620b7002e99c83b48eb8d1bf` | `d7b77e8c74a096366dd6190cbb2fa50a` |
| Chrome Windows | h2 | `Mozilla/5.0 ... Chrome/133.0.0.0 ...` | `t13d1516h2_8daaf6152771_d8a2da3f94cd` | `1d4ffe9b0e34acac0bd883fa7f79d7b5` | `52d84b11737d980aef856699f885ca86` |
| Firefox Windows | h2 | `Mozilla/5.0 ... Firefox/120.0` | `t13d1715h2_5b57614c22b0_5c2c66f702b0` | `b9c611f928c8c1f20c414a48c66abf27` | `6ea73faa8fc5aac76bded7bd238f6433` |

结论：

- `ImpersonateChromeWithOS` 和 `ImpersonateFirefoxWithOS` 会同时改变 `User-Agent`、TLS/JA4、Peetprint、HTTP/2 SETTINGS/顺序，也就是 HTTP 指纹伪装是生效的。
- JA3 hash 可能因为 GREASE、session、uTLS 随机项在不同请求间变化，不要只看 JA3；建议一起看 JA4、Peetprint、HTTP/2 Akamai 和 headers。
- `EnableForceHTTP3()` 访问这个 endpoint 时，本机测试不回退会 `timeout: no recent network activity`；开启 `EnableHTTP3FallbackOnError()` 后会稳定回退到 h2，并保留 Chrome-like H2/TLS 指纹。
- 这不是“所有风控必过”的保证，只说明 req 发出的 TLS/HTTP/2/header 指纹已经能从默认 Go/req 指纹切换成浏览器 profile。

## 测试说明

CI 会在 Linux 和 Windows 上跑 Go 1.24/1.25。自用时本地直接跑：

```sh
go test ./...
```

## 致谢

- 感谢 [imroc/req](https://github.com/imroc/req)，这个库的基础能力来自原项目。
- 感谢 [enetx/surf](https://github.com/enetx/surf)，HTTP/3 tuning、现代浏览器 profile、TLS impersonation 等思路给了很多参考。
- 感谢 [go-resty/resty](https://github.com/go-resty/resty)，请求构造、multipart field、raw path 参数和易用 API 设计给了很多启发。

## License

MIT，见 [LICENSE](LICENSE)。
