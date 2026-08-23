# 中间件与可观测性

## 请求和响应 middleware

```go
client := req.C().
	OnBeforeRequest(func(_ *req.Client, r *req.Request) error {
		r.SetHeader("X-Request-ID", newRequestID())
		return nil
	}).
	OnAfterResponse(func(_ *req.Client, resp *req.Response) error {
		metrics.Observe(resp.Request.Method, resp.GetStatusCode(), resp.TotalTime())
		return nil
	})
```

`OnBeforeRequest` 在内部 URL/body 解析前运行，因此此时使用 `r.RawURL`，不要假定 `r.URL` 已经生成。middleware 在重试的每次 attempt 都可能执行，需要避免重复追加 Header、重复扣费或其他非幂等副作用。

Request 还可以通过 `OnAfterResponse` 注册单次响应 middleware。

## WrapRoundTrip

需要包裹完整 req round trip 时：

```go
client.WrapRoundTripFunc(func(next req.RoundTripper) req.RoundTripFunc {
	return func(r *req.Request) (*req.Response, error) {
		start := time.Now()
		resp, err := next.RoundTrip(r)
		log.Printf("%s %s cost=%s err=%v", r.Method, r.URL, time.Since(start), err)
		return resp, err
	}
})
```

这个层级拿到的是已经解析过的 request。若集成只认识标准库的组件，可通过 `client.GetTransport().WrapRoundTripFunc` 包裹 `http.RoundTripper`。

## Trace

```go
resp, err := client.R().EnableTrace().Get(url)
if err != nil {
	return err
}

trace := resp.TraceInfo()
log.Printf("total=%s dns=%s connect=%s tls=%s first-byte=%s blame=%s",
	trace.TotalTime,
	trace.DNSLookupTime,
	trace.ConnectTime,
	trace.TLSHandshakeTime,
	trace.FirstResponseTime,
	trace.Blame(),
)
```

所有请求开启用 `EnableTraceAll`。Trace 有额外开销，生产中建议采样开启。HTTP/3 会转发连接、TLS、连接复用和首字节等核心 trace 事件，因此可读取 `TraceInfo`；但字段受协议和解析路径影响，`DNSLookupTime` 可能为零，历史命名的 `TCPConnectTime` 也不能在 H3 下解释为真实 TCP 握手。排查时结合 dump/debug log，并与 H2 对照。

## Dump

单次：

```go
resp, err := client.R().EnableDumpWithoutBody().Get(url)
if resp != nil {
	log.Print(resp.Dump())
}
```

client 级可使用 `EnableDumpAll`、`EnableDumpAllTo`、`EnableDumpAllToFile`，或使用 `EnableDumpEachRequest` 仅把单次 dump 暂存在 response 中。上传下载建议关闭 body dump。

Dump 可能包含 Authorization、Cookie、token 和个人信息。写日志前应脱敏；不要因为 async dump 降低了时延影响，就忽略信息泄露和磁盘容量风险。

## Logger 和错误 hook

`SetLogger` 注入实现了 `req.Logger` 的日志器，传 nil 可关闭日志。`OnError` 统一观察请求错误；不要只依赖 middleware 推断最终失败，因为 retry condition 可能提前停止或预算耗尽。

## 响应体转换

```go
client.SetResponseBodyTransformer(func(raw []byte, r *req.Request, resp *req.Response) ([]byte, error) {
	return bytes.TrimSpace(raw), nil
})
```

transformer 在自动读取后、反序列化前运行，适合去 BOM、解包或解密。禁用自动读取时不会自动应用；大 payload 的 transformer 会额外持有字节切片，应纳入内存预算。

完整本地示例：[middleware_test.go](examples/middleware_test.go)。
