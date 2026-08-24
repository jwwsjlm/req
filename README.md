# req

这个仓库是我自用的 `req` 增强版，基于 [imroc/req](https://github.com/imroc/req) 做扩展，重点加强浏览器伪装、HTTP/3、TLS/JA3 指纹、HTTP/2/HTTP/3 细节控制和一些日常使用体验。

原版文档仍然可以参考：[https://req.cool](https://req.cool)

第一次使用这个增强版 req，或者想一边动手一边理解 Go，建议先看：[Go 与 req 零基础入门](docs/00-go-req-beginner.md)。它从空目录、`go mod init` 和第一个请求开始，带你写到可复用、可取消、可测试的业务 API Client。需要按功能查大量代码片段时，再看 [示例.md](示例.md)。

按主题查阅和生产实践请看 [中文 Wiki](docs/Home.md)；仓库内可离线编译验证的示例见 [docs/examples](docs/examples/README.md)。

## 主要能力

- 简洁链式 API，保留 `req.C().R().Get(...)` 这种写法。
- 支持 HTTP/1.1、HTTP/2、HTTP/3，可以自动协商，也可以强制指定。
- 浏览器伪装增强：Chrome、Firefox、Safari，并支持不同系统 profile。
- 支持 uTLS TLS 指纹、JA3、自定义/捕获的 `ClientHelloSpec`、显式 ALPN/NoALPN 随机模式和固定 seed。
- HTTP/2 可控：SETTINGS、header order、pseudo header order、priority、initial stream id。
- HTTP/3 可控：SETTINGS、GREASE、Datagram、Extended CONNECT、QUICConfig、TLS profile、Alt-Svc 失败回退。
- HTTP/3 QUIC 性能 profile：token reuse、keepalive、窗口大小、初始包大小。
- 支持自定义 DNS resolver 和 DNS-over-TLS，HTTP/1.1、HTTP/2、HTTP/3 共用同一套解析策略。
- 支持从响应中提取 TLS 版本、证书信息和 SHA-256 指纹。
- 支持自定义 CookieJar factory，兼容 `func() http.CookieJar` 和旧的 `func() *cookiejar.Jar`。
- 请求构造补强：Any 类型参数、多值 Header、Raw Path 参数、HTTP `QUERY`、流式 multipart、带 Content-Type 的 multipart field、显式 Content-Length。
- 支持响应体大小上限、公开 retry 配置、自定义 hosts、SOCKS4/SOCKS4a 和跨域重定向敏感 Header 清理。
- 保留 req 原有的 debug、dump、retry、download、upload、middleware、自动 JSON/XML 等能力。

## 相比原版的增强点

原版 [imroc/req](https://github.com/imroc/req) 已经提供了成熟的链式 HTTP client、HTTP/2/HTTP/3、调试、重试、上传下载、Cookie、middleware 等基础能力。这个 fork 不是重写一套新库，而是在原版基础上继续补强我自己常用的场景：

- **浏览器伪装更完整**：内置固定版本的 Chrome、Firefox、Safari profile，并区分 Windows、macOS、Linux、Android、iOS 的 UA/Header；`ImpersonateChromeWithOS` 这类方法会同时配置常见请求头、TLS 指纹、HTTP/2 顺序和该 profile 明确提供的 HTTP/3 设置，而不是只改一个 Header。
- **TLS/HTTP 指纹可调得更细**：支持 JA3、fresh uTLS `ClientHelloSpec`、严格导入捕获 ClientHello、可复现随机指纹、Chrome TLS profile、HTTP/2 SETTINGS/header order/pseudo header order/priority/initial stream id，方便在授权环境验证实际指纹。
- **HTTP/3 控制更偏实战**：补了 HTTP/3 SETTINGS、GREASE、Datagram、Extended CONNECT、QUICConfig、QUIC 性能 profile、Alt-Svc 失败冷却和失败回退；普通抓取建议 `EnableHTTP3().EnableHTTP3FallbackOnError()`，不要一上来强制 H3。
- **DNS 和 TLS 信息更方便**：支持自定义 resolver、DNS-over-TLS provider，并能从响应读取 TLS 版本、证书信息和 SHA-256 指纹，排查网络和证书问题更省事。
- **请求构造更适合日常业务**：增加 Any 类型参数、多值 Header、raw path 参数、`SetQueryParamsFromStruct`、带 Content-Type 的 multipart field、显式 `Content-Length`、自定义 CookieJar factory 等补充方法。
- **资源释放和高并发更稳**：对 dump、trace、retry、multipart 上传、parallel download 做了并发和资源释放加固，重点处理 response body、文件句柄、临时目录、goroutine/channel 退出这些长期运行时容易踩的坑。
- **中文新手文档更完整**：README 和 [示例.md](示例.md) 都使用 `github.com/jwwsjlm/req/v3`，并覆盖从 `go mod init` 到完整业务 client 封装的用法。

## 本轮更新：uTLS 兼容与模式增强

- **标准 TLS 配置兼容桥**：uTLS 路径保留显式 SNI、mTLS、验证回调的执行/错误语义、session cache、renegotiation 和 ECH 客户端配置；ClientHello 形状字段由所选指纹 spec 主导，完整边界见 TLS 专题文档。
- **Clone 隔离**：指纹握手会重新绑定到 clone 自己的 `tls.Config`，原 client 与 clone 可使用不同 CA、SNI 和验证策略。
- **IPv6 与错误路径**：正确处理 host、`host:port`、括号/非括号 IPv6；nil spec factory 返回错误而不是 panic。
- **随机模式**：新增显式 ALPN/NoALPN 与 `WithSeed` API，调用方 seed/weights 会被防御性复制。
- **捕获 ClientHello**：新增 `ParseTLSClientHello`，严格校验完整 TLS record，拒绝未知扩展并为每次握手返回 fresh spec。
- **稳定 profile**：Chrome、Firefox、Safari TLS preset 固定明确版本，profile 切换会清理未来连接使用的旧 Header/H2/H3 状态。

完整边界、兼容说明和示例见 [浏览器与 TLS 指纹](docs/10-browser-tls-fingerprint.md)。

## 上一轮更新：兼容性优先的热路径优化

本次更新不改变公开 API，重点减少请求与响应热路径中的重复分配，同时把异常输入和不可信远端数据的资源上限放在首位：

- **Query 合并**：client 与 request 参数改为浅层 map 合并；request 同名 key 仍覆盖 client，key 仍区分大小写，输入 map 和 value slice 不被修改，最终继续由标准库 `url.Values.Encode` 按 key 排序编码。
- **Header 排序**：排序前一次性计算 rank。常见小列表使用与 `textproto.CanonicalMIMEHeaderKey` 等价的无分配匹配，大列表直接调用该标准库函数；合法字段、非法字段、Unicode、伪 Header、重复排序键以及 HTTP/1.1、HTTP/2、HTTP/3 的原有语义均保持不变。
- **响应读取**：仅对已知长度且不超过 8 KiB 的小响应做有限预分配；`Content-Length` 只是容量提示，从不作为读取边界。短报、长报、未知长度和大响应仍读取到 EOF，HEAD/`http.NoBody` 不使用声明长度，读取错误继续连同已读数据返回。
- **资源边界**：Header map 和 Query map 的容量提示都按实际可用上界限制，避免大量重复 key 造成不必要的大块提前分配。
- **工具链**：`go.mod` 与 Linux/Windows CI 统一使用 Go `1.26.7`。

兼容性测试会把优化实现与 `v3.61.1` 的旧行为逐项对照，并覆盖随机大小写、非法与 Unicode Header、Query 输入不变性、8 KiB 响应边界、错误注入和 HEAD 响应。性能数据见下方最终同机基准；不同 CPU、系统和 Go 版本下应关注相对变化，而不是绝对耗时。

### 同机基准（2026-08-23）

环境：Windows/amd64、Go `1.26.7`、Intel Core i5-14600KF；基线为 `af48c83`（`v3.61.1`），每项运行 10 次，下表使用中位数。`ns/op`、`B/op` 和 `allocs/op` 三项指标均为**越低越好**；负百分比表示耗时、内存或分配次数下降。百分比是本机描述性结果，不代表所有环境都能得到相同幅度。

| 场景 | ns/op（越低越好）：基线 → 当前 | B/op（越低越好）：基线 → 当前 | allocs/op（越低越好）：基线 → 当前 |
| --- | ---: | ---: | ---: |
| 仅 client Query | 1,376 → 927（-32.6%） | 1,344 → 1,216（-9.5%） | 21 → 13（-38.1%） |
| client + request Query | 2,177 → 1,718（-21.1%） | 2,264 → 1,992（-12.0%） | 29 → 16（-44.8%） |
| 常规小写 Header 顺序 | 10,541 → 1,467（-86.1%） | 3,520 → 224（-93.6%） | 133 → 2（-98.5%） |
| 常规规范化 Header 顺序 | 5,648 → 1,654（-70.7%） | 1,808 → 224（-87.6%） | 16 → 2（-87.5%） |
| 128 项 Header 顺序 | 33,308 → 14,833（-55.5%） | 20,040 → 11,736（-41.4%） | 420 → 261（-37.9%） |
| 1 KiB 响应读取 | 1,045 → 673（-35.6%） | 2,496 → 1,856（-25.6%） | 8 → 5（-37.5%） |
| 8 KiB 响应读取 | 5,627 → 2,260（-59.9%） | 17,856 → 9,792（-45.2%） | 14 → 5（-64.3%） |
| 64 KiB 响应读取 | 不作为本次优化结论 | 138,432 → 138,432（不变） | 20 → 20（不变） |

64 KiB 场景有意不使用长度预分配，继续走 `io.ReadAll`；其内存与分配次数没有变化。两批非随机交错样本虽观测到耗时差异，但无法归因于本次实现，因此不作为优化收益。复现命令与稳定性验证见 [性能与稳定性](docs/12-performance-stability.md)。

## 文档导航

| 主题 | 文档 |
| --- | --- |
| 第一次使用这个 fork / Go 零基础 | [Go 与 req 零基础入门](docs/00-go-req-beginner.md) |
| 首页与推荐阅读顺序 | [中文 Wiki 首页](docs/Home.md) |
| 快速入门与核心对象 | [快速入门](docs/01-getting-started.md)、[Client / Request / Response](docs/02-client-request-response.md) |
| 请求、响应、认证和可靠性 | [构建请求](docs/03-building-requests.md)、[错误处理](docs/04-error-handling.md)、[认证与 Cookie](docs/05-auth-cookie.md)、[超时、重试与 Context](docs/06-timeout-retry-context.md) |
| 网络、上传下载与可观测性 | [代理、DNS 与重定向](docs/07-proxy-dns-redirect.md)、[上传与下载](docs/08-upload-download.md)、[中间件与可观测性](docs/09-middleware-observability.md) |
| 浏览器、TLS、HTTP/2 与 HTTP/3 | [浏览器与 TLS 指纹](docs/10-browser-tls-fingerprint.md)、[HTTP/2 与 HTTP/3](docs/11-http2-http3.md) |
| 性能、配方、迁移与 API | [性能与稳定性](docs/12-performance-stability.md)、[生产配方](docs/13-recipes.md)、[迁移与兼容](docs/14-migration-compatibility.md)、[API 索引](docs/15-api-index.md)、[上游项目与许可](docs/16-upstream-credits.md) |
| 可编译示例 | [docs/examples](docs/examples/README.md) |

## 方法速查

| 场景 | 常用方法 |
| --- | --- |
| 创建 client/request | `C()`、`NewClient()`、`DefaultClient()`、`SetDefaultClient()`、`NewTransport()`、`T()`、`R()`、`NewRequest()`、`Clone()` |
| HTTP 方法 | `Get()`、`Post()`、`Put()`、`Patch()`、`Delete()`、`Head()`、`Options()`、`Query()`、`Send()`、`Do()`、`MustGet()`、`MustPost()`、`MustPut()`、`MustPatch()`、`MustDelete()`、`MustOptions()`、`MustHead()`、`MustQuery()`、`EnableAllowGetMethodPayload`、`DisableAllowGetMethodPayload` |
| BaseURL/路径 | `SetBaseURL`、`SetScheme`、`SetURL`、`SetPathParam`、`SetPathParamAny`、`SetPathParams`、`SetPathRawParam`、`SetPathRawParamAny`、`SetPathRawParams` |
| Query 参数 | `SetQueryParam`、`SetQueryParamAny`、`AddQueryParam`、`AddQueryParams`、`SetQueryParams`、`SetQueryParamsAnyType`、`SetQueryParamsFromValues`、`SetQueryParamsFromStruct`、`SetQueryString` |
| Header | `SetHeader`、`SetHeaderAny`、`SetHeaderValues`、`SetHeaderMultiValues`、`SetHeaders`、`SetHeaderNonCanonical`、`SetHeadersNonCanonical`、`SetHeaderOrder`、`SetPseudoHeaderOrder` |
| 公共 Header | `SetUserAgent`、`SetCommonHeader`、`SetCommonHeaderAny`、`SetCommonHeaderValues`、`SetCommonHeaderMultiValues`、`SetCommonHeaders`、`SetCommonHeaderNonCanonical`、`SetCommonHeadersNonCanonical`、`SetCommonHeaderOrder`、`SetCommonPseudoHeaderOder`、`SetCommonContentType` |
| 公共 Query/路径 | `SetCommonQueryParam`、`SetCommonQueryParamAny`、`SetCommonQueryParams`、`AddCommonQueryParam`、`AddCommonQueryParams`、`SetCommonQueryParamsFromValues`、`SetCommonQueryParamsFromStruct`、`SetCommonQueryString`、`SetCommonPathParam`、`SetCommonPathParamAny`、`SetCommonPathParams`、`SetCommonPathRawParam`、`SetCommonPathRawParamAny`、`SetCommonPathRawParams` |
| Cookie | `SetCookies`、`SetCommonCookies`、`SetCookieJarFactory`、`SetCookieJar`、`GetCookies`、`ClearCookies` |
| 认证 | `SetAuthToken`、`SetBearerAuthToken`、`SetAuthSchemeToken`、`SetBasicAuth`、`SetDigestAuth`、`SetCommonAuthToken`、`SetCommonBearerAuthToken`、`SetCommonAuthSchemeToken`、`SetCommonBasicAuth`、`SetCommonDigestAuth` |
| Body/Content-Type | `SetBody`、`SetBodyBytes`、`SetBodyString`、`SetBodyJsonString`、`SetBodyJsonBytes`、`SetBodyJsonMarshal`、`SetBodyXmlString`、`SetBodyXmlBytes`、`SetBodyXmlMarshal`、`SetContentType`、`SetContentLength` |
| 表单/multipart | `SetFormData`、`SetFormDataAny`、`SetFormDataAnyType`、`SetFormDataFromValues`、`SetOrderedFormData`、`SetCommonFormData`、`SetCommonFormDataAny`、`SetCommonFormDataAnyType`、`SetCommonFormDataFromValues`、`SetMultipartBoundaryFunc`、`SetMultipartField`、`SetFileUpload`、`EnableForceMultipart`、`DisableForceMultipart` |
| 上传 | `SetFile`、`SetFiles`、`SetFileBytes`、`SetFileReader`、`SetUploadCallback`、`SetUploadCallbackWithInterval`、`EnableForceChunkedEncoding`、`DisableForceChunkedEncoding` |
| 下载 | `SetOutputFile`、`SetOutput`、`SetOutputDirectory`、`SetDownloadCallback`、`SetDownloadCallbackWithInterval`、`NewParallelDownload`、`SetConcurrency`、`SetSegmentSize`、`SetTempRootDir`、`SetFileMode` |
| 结果解析 | `SetSuccessResult`、`SetErrorResult`、`SetResult`、`SetError`、`SuccessResult`、`ErrorResult`、`Result`、`Error`、`Into`、`Unmarshal`、`UnmarshalJson`、`UnmarshalXml`、`ToString`、`ToBytes` |
| Response 读取 | `String`、`Bytes`、`Dump`、`GetStatus`、`GetStatusCode`、`GetContentType`、`GetHeader`、`GetHeaderValues`、`HeaderToString`、`IsSuccess`、`IsError`、`TLSInfo`、`TLSGrabber`、`TotalTime`、`ReceivedAt` |
| 错误处理 | `SetCommonErrorResult`、`SetCommonError`、`SetResultStateCheckFunc`、`OnError`、`OnBeforeRequest`、`OnAfterResponse`、`IsSuccessState`、`IsErrorState`、`ResultState` |
| 超时/context/响应限制 | `SetTimeout`、`SetContext`、`Context`、`SetContextData`、`GetContextData`、`SetClient`、`SetMaxResponseSize`、`DisableAutoReadResponse`、`EnableAutoReadResponse`、`EnableCloseConnection` |
| 重试 | `SetCommonRetryCount`、`SetCommonRetryInterval`、`SetCommonRetryFixedInterval`、`SetCommonRetryBackoffInterval`、`SetCommonRetryCondition`、`AddCommonRetryCondition`、`SetCommonRetryHook`、`AddCommonRetryHook`、`SetRetryCount`、`SetRetryInterval`、`SetRetryFixedInterval`、`SetRetryBackoffInterval`、`SetRetryCondition`、`AddRetryCondition`、`SetRetryHook`、`AddRetryHook`、`GetRetryOption` |
| 调试 dump | `DevMode`、`SetDebug`、`EnableDebugLog`、`DisableDebugLog`、`EnableDumpAll`、`EnableDumpAllTo`、`EnableDumpAllToFile`、`EnableDumpAllAsync`、`EnableDumpEachRequest`、`EnableDumpEachRequestWithoutBody`、`EnableDump`、`EnableDumpTo`、`EnableDumpToFile`、`DisableDump`、`DisableDumpAll` |
| Dump 细节 | `SetCommonDumpOptions`、`SetDumpOptions`、`EnableDumpAllWithoutBody`、`EnableDumpAllWithoutHeader`、`EnableDumpAllWithoutRequest`、`EnableDumpAllWithoutRequestBody`、`EnableDumpAllWithoutResponse`、`EnableDumpAllWithoutResponseBody`、`EnableDumpEachRequestWithoutHeader`、`EnableDumpEachRequestWithoutRequest`、`EnableDumpEachRequestWithoutRequestBody`、`EnableDumpEachRequestWithoutResponse`、`EnableDumpEachRequestWithoutResponseBody`、`EnableDumpWithoutBody`、`EnableDumpWithoutHeader`、`EnableDumpWithoutRequest`、`EnableDumpWithoutRequestBody`、`EnableDumpWithoutResponse`、`EnableDumpWithoutResponseBody` |
| Trace | `EnableTraceAll`、`DisableTraceAll`、`EnableTrace`、`DisableTrace`、`TraceInfo`、`Blame` |
| 浏览器伪装 | `ImpersonateChrome`、`ImpersonateChromeWithOS`、`ImpersonateChromeRandomOS`、`ImpersonateFirefox`、`ImpersonateFirefoxWithOS`、`ImpersonateFirefoxRandomOS`、`ImpersonateSafari`、`RandomBrowserOS` |
| TLS 指纹 | `SetTLSFingerprint`、`SetTLSFingerprintJA3`、`SetTLSFingerprintSpec`、`SetTLSFingerprintSpecFactory`、`SetTLSFingerprintRandomizedALPN`、`SetTLSFingerprintRandomizedNoALPN`、两种 `WithSeed`、内置浏览器 preset、`ParseTLSClientHello` |
| TLS/证书 | `SetTLSClientConfig`、`GetTLSClientConfig`、`SetRootCertFromString`、`SetRootCertsFromFile`、`SetCertFromFile`、`SetCerts`、`EnableInsecureSkipVerify`、`DisableInsecureSkipVerify` |
| DNS | `NewDNSOverTLSResolver`、`SetDNSResolver`、`SetResolver`、`SetHosts`、`SetDNSOverTLS`、`SetDNSOverTLSCloudflare`、`SetDNSOverTLSGoogle`、`SetDNSOverTLSQuad9`、`SetDNSOverTLSAdGuard`、`SetDNSOverTLSAli` |
| 代理/dial | `SetProxyURL`、`SetProxy`、`SetProxyConnectHeader`、`SetGetProxyConnectHeader`、`SetUnixSocket`、`SetDial`、`SetDialTLS`、`SetTLSHandshake`、`SetTLSHandshakeTimeout` |
| 重定向 | `SetRedirectPolicy`、`MaxRedirectPolicy`、`DefaultRedirectPolicy`、`NoRedirectPolicy`、`SameDomainRedirectPolicy`、`SameHostRedirectPolicy`、`AllowedHostRedirectPolicy`、`AllowedDomainRedirectPolicy`、`AlwaysCopyHeaderRedirectPolicy`、`SensitiveHeadersRedirectPolicy` |
| 压缩/解码 | `EnableAutoDecompress`、`DisableAutoDecompress`、`EnableCompression`、`DisableCompression`、`EnableAutoDecode`、`DisableAutoDecode`、`SetAutoDecodeContentType`、`SetAutoDecodeAllContentType`、`SetAutoDecodeContentTypeFunc`、`SetResponseBodyTransformer` |
| HTTP 版本 | `EnableForceHTTP1`、`EnableForceHTTP2`、`EnableForceHTTP3`、`DisableForceHttpVersion`、`EnableH2C`、`DisableH2C`、`EnableHTTP3`、`DisableHTTP3`、`EnableHTTP3FallbackOnError` |
| HTTP/2 细节 | `SetHTTP2SettingsFrame`、`SetHTTP2ConnectionFlow`、`SetHTTP2InitialStreamID`、`SetHTTP2HeaderPriority`、`SetHTTP2PriorityFrames`、`SetHTTP2MaxHeaderListSize`、`SetHTTP2ReadIdleTimeout`、`SetHTTP2PingTimeout`、`SetHTTP2WriteByteTimeout`、`SetHTTP2StrictMaxConcurrentStreams` |
| HTTP/3 细节 | `SetHTTP3AdditionalSettings`、`SetHTTP3AdditionalSetting`、`SetHTTP3Grease`、`EnableHTTP3Datagrams`、`DisableHTTP3Datagrams`、`EnableHTTP3ExtendedConnect`、`DisableHTTP3ExtendedConnect`、`SetHTTP3MaxResponseHeaderBytes`、`SetHTTP3QUICConfig`、`SetHTTP3QUICPerformanceProfile`、`SetHTTP3QUICChromeProfile`、`SetHTTP3TLSClientConfig`、`SetHTTP3TLSChromeProfile`、`SetHTTP3TLSFirefoxProfile`、`EnableHTTP3FallbackOnError`、`DisableHTTP3FallbackOnError`、`SetHTTP3AltSvcFailureCooldown` |
| Transport/性能 | `GetTransport`、`SetMaxIdleConns`、`GetMaxIdleConns`、`SetMaxConnsPerHost`、`SetIdleConnTimeout`、`SetResponseHeaderTimeout`、`SetMaxResponseHeaderBytes`、`SetExpectContinueTimeout`、`SetReadBufferSize`、`SetWriteBufferSize`、`DisableKeepAlives`、`EnableKeepAlives`、`CloseIdleConnections`、`CancelRequest` |
| 扩展集成 | `GetClient`、`Do(*http.Request)`、`RoundTrip`、`WrapRoundTripFunc`、`WrapRoundTrip`、`NewLogger`、`NewLoggerFromStandardLogger`、`GetLogger`、`SetLogger`、`SetJsonMarshal`、`SetJsonUnmarshal`、`SetXmlMarshal`、`SetXmlUnmarshal` |

## 安装

```sh
go get github.com/jwwsjlm/req/v3
```

要求 Go `1.26.7` 或更高版本；准确要求以 [go.mod](go.mod) 为准。

如果你还不熟 Go module、`main.go`、`go run` 这些基础步骤，先按 [示例.md](示例.md) 跑一遍最小项目。

## 推荐使用方式

普通 API 调用建议长期复用一个 client：

```go
var apiClient = req.C().
	SetTimeout(30 * time.Second).
	SetCommonHeader("Accept", "application/json").
	SetCommonHeader("Accept-Language", "zh-CN,zh;q=0.9")
```

偏浏览器访问、反爬压测、站点抓取时用浏览器 profile：

```go
var browserClient = req.C().
	ImpersonateChromeWithOS(req.BrowserOSWindows).
	SetDNSOverTLSCloudflare().
	EnableHTTP3().
	EnableHTTP3FallbackOnError().
	SetHTTP3AltSvcFailureCooldown(30 * time.Second)
```

只想稳定优先，不想强制 HTTP/3：

```go
var stableClient = req.C().
	SetTimeout(20 * time.Second).
	EnableHTTP3().
	EnableHTTP3FallbackOnError()
```

重试优先在明确的只读/幂等 request 上配置。公共 `SetCommonRetry*` 会作用于这个 client 的所有方法；除非 condition 显式检查请求方法，或业务使用幂等键/去重，否则不要让它自动重试 POST/PATCH 等写请求。

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
	SetBodyJsonMarshal(&Repo{Name: "req", URL: "https://github.com/jwwsjlm/req"}).
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

推荐把重试限制在明确的只读请求：

```go
resp, err := client.R().
	SetRetryCount(2).
	SetRetryBackoffInterval(300*time.Millisecond, 3*time.Second).
	SetRetryCondition(func(resp *req.Response, err error) bool {
		if err != nil {
			return !errors.Is(err, context.Canceled) &&
				!errors.Is(err, context.DeadlineExceeded)
		}
		return resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500)
	}).
	Get("https://api.example.com/flaky")
```

专用于 GET/HEAD/OPTIONS/QUERY 的 client 可以使用 `SetCommonRetry*`；通用 client 的公共 condition 应先检查 `resp.Request.Method`。POST/PATCH 等写请求只有在业务提供幂等键、去重或等价保证时才配置重试。

单个请求覆盖重试：

```go
resp, err := client.R().
	SetRetryCount(3).
	SetRetryFixedInterval(time.Second).
	Get("https://api.example.com/flaky")
```

middleware 需要读取当前请求最终生效的重试配置时，可以使用 `GetRetryOption()`；没有配置重试时返回 `nil`：

```go
client.OnAfterResponse(func(c *req.Client, resp *req.Response) error {
	if option := resp.Request.GetRetryOption(); option != nil {
		log.Printf("retry count=%d", option.MaxRetries)
	}
	return nil
})
```

## 响应体大小限制

面对不可信接口或可能异常返回超大 body 的服务，可以设置字节上限，避免自动读取时无限占用内存和带宽：

```go
client := req.C().
	SetMaxResponseSize(8 * 1024 * 1024) // 8 MiB

// 单个请求可以覆盖 client 配置；0 表示禁用限制。
resp, err := client.R().
	SetMaxResponseSize(1024 * 1024).
	Get("https://api.example.com/data")
```

已知 `Content-Length` 超限时会在读取前报错；chunked、压缩或未知长度响应会在流式读取达到上限时报错。限制针对 transport 解压后、应用实际读取到的字节。

## 代理和重定向

HTTP/HTTPS/SOCKS5/SOCKS4 代理：

```go
client := req.C().
	SetProxyURL("http://127.0.0.1:7890")

client = req.C().
	SetProxyURL("socks5://127.0.0.1:1080")

client = req.C().
	SetProxyURL("socks4://127.0.0.1:1080")

// 由代理端解析域名。
client = req.C().
	SetProxyURL("socks4a://127.0.0.1:1080")
```

`socks4` 只支持 IPv4 目标并在本地解析域名；`socks4a` 会把域名交给代理解析。

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
		req.AllowedHostRedirectPolicy("api.example.com", "login.example.com"),
	)
```

不跟随重定向：

```go
client := req.C().
	SetRedirectPolicy(req.NoRedirectPolicy())
```

跨域重定向时删除自定义认证 Header，避免凭据泄漏；同域跳转会保留：

```go
client := req.C().
	SetRedirectPolicy(
		req.AllowedHostRedirectPolicy("api.example.com", "login.example.com"),
		req.SensitiveHeadersRedirectPolicy("X-API-Key", "X-Token"),
	)
```

`SameDomainRedirectPolicy`、`AllowedDomainRedirectPolicy` 和 `SensitiveHeadersRedirectPolicy` 使用简单标签裁剪，不是 Public Suffix List/eTLD+1 判断；例如 `foo.co.uk` 与 `bar.co.uk` 可能被视为同域。高价值凭据应以明确的 `AllowedHostRedirectPolicy` 为主要边界；若还需约束 scheme、端口或完整 origin，请实现自定义策略。

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

HTTP/3 会提供连接、TLS、复用和首字节等核心 `TraceInfo`，但 DNS 与历史命名的 `TCPConnectTime` 字段可能为零或不代表真实 TCP。排查 H3 时应结合 dump/debug log，并与 H2 对照。

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
- Chrome/Firefox 使用 method-aware headers，GET/POST 会采用不同请求头；Safari 当前使用一组静态 common headers。
- Chrome/Firefox 明确提供的 HTTP/3 SETTINGS、TLS profile、QUIC profile；Safari 当前没有专用 H3 profile。

Chrome 固定 uTLS Chrome 133，Firefox 固定 Firefox 120，Safari Header/UA 为 16.6 风格而 TLS preset 为 Safari 16.0。OS 选项主要改变 UA、Client Hints 和 Header，不代表 TLS ClientHello 会随 OS 完全变化。

应在第一个请求前完成配置。profile 切换会清理未来连接的旧 Header/H2/H3 状态，但不会改写已经建立的连接；切换身份优先使用新 client 或尚未使用的 clone。

## JA3 和自定义 TLS 指纹

JA3：

```go
ja3 := "771,4865-4866-4867-49195-49199,0-5-10-11-13-16-43-51,29-23-24,0"
ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0"

client := req.C().
	SetTLSFingerprintJA3(ja3, ua, false)
```

自定义 uTLS spec。client 可能建立多条 TLS 连接时，必须让 factory 每次返回一个全新的 spec：

```go
client := req.C().
	SetTLSFingerprintSpecFactory(func() *utls.ClientHelloSpec {
		spec, err := utls.UTLSIdToSpec(utls.HelloChrome_133)
		if err != nil {
			panic(err)
		}
		return &spec
	})
```

旧的 `SetTLSFingerprintSpec` 仍保留兼容，但 uTLS 的 `ApplyPreset` 会修改传入 spec；同一个 client 连续连接不同域名时应改用 factory，避免重复握手失败。Transport 可并发建连，自定义 factory 也必须自行保护共享状态。

需要同一个 client 的随机结构在新连接间保持稳定时，使用固定 seed：

```go
seed, err := utls.NewPRNGSeed()
if err != nil {
	log.Fatal(err)
}

client := req.C().
	SetTLSFingerprintRandomizedALPNWithSeed(seed)
```

只走 HTTP/1.1 的对端可改用 `SetTLSFingerprintRandomizedNoALPN` 或对应 `WithSeed`。req 会复制 seed，调用后修改原值不会影响 client。

从授权环境捕获的一条完整明文 ClientHello record 也可以严格导入：

```go
factory, err := req.ParseTLSClientHello(rawTLSRecord)
if err != nil {
	log.Fatal(err)
}

client := req.C().SetTLSFingerprintSpecFactory(factory)
```

解析器限制单条 TLS plaintext record 为 16 KiB，拒绝未知扩展和任何捕获到的 `pre_shared_key` 扩展，并为每次握手从独立字节副本重新生成 fresh spec。

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

启用 uTLS 指纹后，上述标准配置中的显式 `ServerName`、客户端证书/动态证书回调、两个验证回调、session cache、renegotiation、key log 和 ECH 客户端配置都会桥接到 uTLS。`MinVersion`、`MaxVersion`、`CipherSuites`、`CurvePreferences`、`NextProtos` 会先转换，但浏览器、随机或自定义指纹 spec 会按自身扩展重写这些 ClientHello 形状字段；不要把它们当成 preset 下的强约束。`Client.Clone` 会使用 clone 自己的 TLS config。uTLS v1.8.2 无法为标准 `ConnectionState` 补出 `CurveID`、`HelloRetryRequest` 或私有 keying-material exporter；安全策略依赖这些信息时请阅读 [TLS 兼容桥边界](docs/10-browser-tls-fingerprint.md)。TLS 1.3 session 恢复还要求所选 ClientHello 自带真实 `PreSharedKeyExtension`；普通 Chrome parrot preset 不会被强行添加该扩展。

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

multipart 文件现在默认以流式方式写入请求体，不会先把整个文件缓冲到内存。已知文件大小时会计算并发送 `Content-Length`；使用无法预知长度的 Reader 时会使用 chunked 传输。若服务端不接受 chunked，请提供可确定大小的文件/Reader，或按接口要求显式设置长度。

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
		SetHTTP3AltSvcFailureCooldown(30 * time.Second)
}
```

重试按具体请求的幂等性配置；dump/debug 也按需临时开启，避免生产默认记录凭据和大 body。

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

也可以使用与上游兼容的 `SetResolver`，或者用静态 hosts 映射让指定域名只连接到给定 IP：

```go
client := req.C().
	SetResolver(&net.Resolver{PreferGo: true}).
	SetHosts(map[string]string{
		"api.example.com": "203.0.113.10",
	})
```

`SetHosts` 是严格映射：未列出的域名会快速失败，并且不能与代理同时使用。传入的 map 会被复制，之后修改原 map 不会影响 client。

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

CI 会在 Linux 和 Windows 上使用 Go `1.26.7`。自用时本地直接跑：

```sh
go test ./...
```

## 致谢

- 感谢 [imroc/req](https://github.com/imroc/req)，这个库的基础能力来自原项目。
- 感谢 [refraction-networking/uTLS](https://github.com/refraction-networking/utls)，本项目通过它提供可定制 ClientHello、浏览器 TLS preset、随机指纹和捕获 ClientHello 解析能力，并在其上完成 req Transport 与标准 `tls.Config` 的兼容桥。uTLS 使用 BSD 3-Clause License；本项目与 uTLS、Refraction Networking、Google 或其贡献者不存在官方隶属或背书关系。
- 感谢 [enetx/surf](https://github.com/enetx/surf)，HTTP/3 tuning、现代浏览器 profile、TLS impersonation 等思路给了很多参考。
- 感谢 [go-resty/resty](https://github.com/go-resty/resty)，请求构造、multipart field、raw path 参数和易用 API 设计给了很多启发。

## License

本项目原创代码在未另行注明时使用 MIT，见 [LICENSE](LICENSE)。仓库内 Go Authors 派生代码与 uTLS 的版权、再分发条件和免责声明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)；其他依赖继续遵循各自许可证，上游归属说明见 [docs/16-upstream-credits.md](docs/16-upstream-credits.md)。
