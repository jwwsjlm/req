# 错误处理

## 三层错误模型

一次调用需要分别检查：

1. 返回的 `error`：URL、DNS、连接、TLS、超时、middleware、读取、解析等错误。
2. HTTP 状态：例如 404、429、500 通常仍可能得到 `err == nil`。
3. 业务 payload：有些服务以 200 返回业务错误码，需要自行判定或定制 `ResultState`。

```go
resp, err := client.R().Get(url)
if err != nil {
	return fmt.Errorf("request failed: %w", err)
}
if !resp.IsSuccessState() {
	return fmt.Errorf("unexpected HTTP status: %s", resp.GetStatus())
}
```

`Send` 返回的 `*Response` 按契约始终非 nil，但错误路径上的底层 `resp.Response` 可能为 nil，访问状态时优先使用 `GetStatusCode` 等安全 helper。

## 结构化错误结果

```go
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var apiErr APIError
resp, err := client.R().
	SetErrorResult(&apiErr).
	Get(url)
if err != nil {
	return err
}
if resp.IsErrorState() {
	return fmt.Errorf("remote error %s: %s", apiErr.Code, apiErr.Message)
}
```

多个接口共用错误结构时用 `Client.SetCommonErrorResult`。旧的 `SetError`、`SetCommonError`、`Response.Error` 仍保留，但新代码优先使用带 `Result` 后缀的方法。

## 自定义结果状态

```go
client.SetResultStateCheckFunc(func(resp *req.Response) req.ResultState {
	if resp.GetStatusCode() >= 200 && resp.GetStatusCode() < 400 {
		return req.SuccessState
	}
	return req.ErrorState
})
```

定制后，`SetSuccessResult`/`SetErrorResult` 和 `IsSuccessState`/`IsErrorState` 都遵循同一判定。

## 响应体过大

```go
resp, err := client.R().
	SetMaxResponseSize(2 << 20).
	Get(url)
if errors.Is(err, req.ErrResponseBodyTooLarge) {
	var sizeErr *req.ResponseBodyTooLargeError
	if errors.As(err, &sizeErr) {
		log.Printf("limit=%d content-length=%d", sizeErr.Limit, sizeErr.ContentLength)
	}
	return err
}
_ = resp
```

限制作用于 transport 处理 Content-Encoding 后交给应用的字节。已知 `Content-Length` 超限时会提前关闭 body；未知长度或自动解压场景在读取过程中执行流式限制。

## OnError 与 middleware 错误

`OnError` 会在请求返回任意错误时调用，包括发送前的非法 URL。它适合统一统计和日志，不应吞掉调用方仍需处理的错误。

```go
client.OnError(func(_ *req.Client, r *req.Request, resp *req.Response, err error) {
	log.Printf("method=%s url=%s attempt=%d err=%v", r.Method, r.RawURL, r.RetryAttempt, err)
})
```

请求 middleware 返回错误会阻止发送；响应 middleware 返回错误会成为调用错误。保持错误可包装、可用 `errors.Is`/`errors.As` 判断。

## 不要在业务路径使用 Must*

`MustGet`、`MustPost` 等方法在错误时 panic。它们适合测试和必须成功的启动初始化，不适合作为普通服务请求的错误处理方式。
