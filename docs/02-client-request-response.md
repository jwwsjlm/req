# Client、Request 与 Response

## Client：长期配置与连接复用

`*req.Client` 持有底层 `*http.Client`、`*req.Transport`、公共 Header/Query/Cookie、middleware、重试和协议配置。通常一个远端服务或一个凭据边界对应一个长期 client。

```go
client := req.C().
	SetBaseURL("https://api.example.com").
	SetCommonHeader("Accept", "application/json").
	SetTimeout(20 * time.Second)
```

可通过 `GetClient()` 获取底层 `*http.Client`，通过 `GetTransport()` 获取 `*req.Transport`。`Client.Do(*http.Request)` 是标准库直通：它复用底层 client/transport，但不会套用 `R()` 的 body 处理、自动结果解析和 req 级 middleware。

## Request：一次调用的覆盖层

`client.R()` 每次返回新的 `*Request`，并复制当时的重试选项。Request 设置优先于 client 公共设置，例如同名 Query 参数由 request 值覆盖。

```go
resp, err := client.R().
	SetPathParam("id", "42").
	SetQueryParam("expand", "profile").
	SetHeader("X-Request-ID", requestID).
	Get("/users/{id}")
```

不要并发复用同一个 `*Request`，也不要把已经发送过并带不可重放 reader 的请求再次发送。每次调用重新执行 `client.R()`。

## Response：HTTP 响应与 req 状态

`*Response` 嵌入底层 `*http.Response`，并增加：

- `Err`：请求流程错误。
- `Request`：关联的 req 请求。
- `IsSuccessState`、`IsErrorState`、`ResultState`：业务结果状态。
- `SuccessResult`、`ErrorResult`：自动反序列化结果。
- `String`、`Bytes`、`ToString`、`ToBytes`：响应体访问。
- `TraceInfo`、`TotalTime`、`ReceivedAt`：耗时信息。
- `TLSInfo`：TLS 和证书信息。

`String()`/`Bytes()` 只返回已经读取的 body；禁用自动读取时，优先使用 `ToBytes()`/`ToString()`，或直接读取并关闭 `resp.Body`。

## Clone 与 Cookie 隔离

`client.Clone()` 会克隆 transport、底层 `http.Client`、公共参数、middleware 和重试配置。

- 使用默认 CookieJar 或 `SetCookieJarFactory` 时，clone 会通过 factory 创建新的 jar。
- 使用 `SetCookieJar` 注入具体 jar 时，clone 后共享该 jar。

```go
base := req.C().SetCommonHeader("Accept", "application/json")
tenantA := base.Clone().SetCommonBearerAuthToken("token-a")
tenantB := base.Clone().SetCommonBearerAuthToken("token-b")
```

如果需要账号 Cookie 强隔离，显式使用 `SetCookieJarFactory(func() http.CookieJar { ... })`。

## DefaultClient

包级 `req.Get`、`req.Post` 等使用全局默认 client。小脚本可以使用，服务端程序建议显式持有 client，避免全局配置和测试相互影响。

```go
req.SetDefaultClient(req.C().SetTimeout(10 * time.Second))
resp, err := req.Get("https://example.com")
```
