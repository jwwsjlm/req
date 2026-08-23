# 超时、重试与 Context

## Client 总超时

```go
client := req.C().SetTimeout(30 * time.Second)
```

`SetTimeout` 设置底层 `http.Client.Timeout`，覆盖一次请求（包括连接、重定向和读取响应体）的总时间。零值表示不设置 client 总超时；生产代码通常应设置明确上限。

更细的 transport 超时可通过 `GetTransport()` 配置：

```go
client.GetTransport().
	SetTLSHandshakeTimeout(5 * time.Second).
	SetResponseHeaderTimeout(10 * time.Second).
	SetExpectContinueTimeout(time.Second).
	SetIdleConnTimeout(90 * time.Second)
```

## 单次 Context

```go
ctx, cancel := context.WithTimeout(parent, 3*time.Second)
defer cancel()

resp, err := client.R().
	SetContext(ctx).
	Get(url)
```

也可先构造 request，再用 `request.Do(ctx)`。context 负责调用链取消、deadline 和值传递；`SetContextData`/`GetContextData` 是 request 自带的数据入口，不等同于标准库 `context.WithValue`。

## 重试基本配置

```go

resp, err := client.R().
	SetRetryCount(2).
	SetRetryBackoffInterval(200*time.Millisecond, 2*time.Second).
	SetRetryCondition(func(resp *req.Response, err error) bool {
		if err != nil {
			return !errors.Is(err, context.Canceled) &&
				!errors.Is(err, context.DeadlineExceeded)
		}
		return resp != nil && (resp.GetStatusCode() == 429 || resp.GetStatusCode() >= 500)
	}).
	Get(url)
```

这里把重试限制在明确的 GET 请求上。`count` 是初次请求之外的最大重试次数；负数表示无限重试，必须同时受 context/deadline 约束。没有自定义 condition 时，默认只对非 nil error 重试；配置 condition 后由 condition 决定。只有专用于只读/幂等调用的 client 才建议使用对应的 `SetCommonRetry*`。

## Request 覆盖

```go
resp, err := client.R().
	SetRetryCount(4).
	SetRetryFixedInterval(500 * time.Millisecond).
	SetRetryCondition(condition).
	Get(url)
```

Request 创建时会克隆 client 的 `RetryOption`。`SetRetryCondition`、`SetRetryHook` 覆盖已有列表；`AddRetryCondition`、`AddRetryHook` 追加。

middleware 可通过 `request.GetRetryOption()` 读取当前配置。`RetryAttempt` 从 0 开始，每次准备重试时递增。

## 可重放性和幂等性

- GET、HEAD、OPTIONS、QUERY 通常适合重试，但仍要考虑服务端实现。
- PUT、DELETE 在协议语义上幂等，业务系统未必真正幂等。
- POST/PATCH 应使用幂等键或明确的服务端去重机制。
- 不可重放的 `io.Reader` body 与启用的重试不能组合；使用 `SetBodyBytes`、`SetBodyString`、可重新打开的 `FileUpload.GetFileContent`，或关闭重试。

上传过程中开始产生副作用后，网络失败不代表服务端没有处理。重试前必须以业务幂等性为依据，而不是只看客户端是否收到响应。

## 重试等待也响应取消

退避等待会监听 request context。context 取消后不会继续等待或重试。不要用 `time.Sleep` 在 condition/hook 中自行阻塞；使用 `SetRetryInterval` 或内置 fixed/backoff 方法。

## 建议组合

```go
client := req.C().
	SetTimeout(30 * time.Second)

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.R().
	SetContext(ctx).
	SetRetryCount(2).
	SetRetryBackoffInterval(200*time.Millisecond, 2*time.Second).
	Get(url)
```

总超时是最后防线，context 是调用级预算，retry backoff 控制恢复节奏，三者不应相互替代。
