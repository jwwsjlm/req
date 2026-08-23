# 性能与稳定性

性能优化的前提是保持协议语义、输入不变性、错误传播和资源释放。不要为了减少一次分配改变 Query 覆盖规则、Header 匹配规则或响应读取边界。

## 当前热路径优化

### Query 合并

client 与 request Query 同时存在时使用浅层 map 合并，request 同名 key 仍覆盖 client。输入 map 和 value slice 不被修改，最终继续交给标准库 `url.Values.Encode`，保持确定的 key 排序和编码规则。

### Header 顺序

排序前一次性计算每个 Header 的 rank，避免在 sort 的每次比较中重复规范化。小列表使用与 `textproto.CanonicalMIMEHeaderKey` 等价的无分配匹配，大列表直接调用该标准库函数，从而保持合法/非法字段名、大小写和重复排序键的旧行为。

### 响应体预分配

不超过 8 KiB 的正 `Content-Length` 只作为有上限的容量提示。未知、异常或较大长度继续走 `io.ReadAll` 的自适应扩容；HEAD 和 `http.NoBody` 不使用声明长度。无论提示是否准确，都读取到 EOF。这样能优化常见小响应，同时避免远端用伪造长度诱导大额预分配。

## Client 与连接池

- 长期复用 client 和 transport。
- 不要在每个请求后调用 `CloseIdleConnections`。
- 根据目标和并发设置 `SetMaxIdleConns`、`SetMaxConnsPerHost`、`SetIdleConnTimeout`。
- 更大的 read/write buffer 不一定更快，会按连接增加内存。
- 只有明确不希望复用连接时才 `DisableKeepAlives` 或 request `EnableCloseConnection`。

## 响应内存边界

```go
client.SetMaxResponseSize(16 << 20)
```

普通 JSON 可以自动读取；大文件优先 `SetOutputFile`/`SetOutput`，或 `DisableAutoReadResponse` 后流式复制。`SetResponseBodyTransformer` 会在内存中处理完整 body，不适合无限大响应。

## 上传和重试稳定性

- 文件上传使用每次可重新打开的 `GetFileContent`。
- 不可重放 reader 不与 retry 混用。
- 上传进度回调会使用 chunked，先确认服务端支持。
- retry condition 必须限制状态和错误类型，backoff 要有上限与 jitter。
- context cancellation 必须覆盖重试等待和网络阶段。

## 可观测性的成本

全量 dump、trace 和 debug log 都有成本。body dump 会复制或保留大块数据，还可能泄露凭据；生产环境按请求采样并默认不记录 body。middleware 和进度回调应快速返回。

## 并发使用原则

初始化完成后的 client 可供多个 goroutine 发请求；不要在并发请求期间持续改写同一个 client 的公共配置。`*Request` 是单次可变 builder，不并发共享。账号之间按 Cookie/token 边界拆分或 clone client。

## 基准与回归命令

仓库内相关 benchmark：

```sh
go test . -run '^$' -bench 'Benchmark(ParseRequestURL|ResponseToBytes)' -benchmem -count=5
go test ./internal/header -run '^$' -bench BenchmarkSortKeyValues -benchmem -count=5
```

比较优化前后应使用同一机器、同一 Go 版本、同一电源策略，并用 `benchstat` 分析多次样本。不要只发布一次运行的最小值。

稳定性验证：

```sh
go vet ./...
go test ./... -count=1
go test -race ./... -count=1
go test ./... -count=5
```

`-race` 不能证明没有竞态，但能覆盖测试实际走到的并发路径；高风险网络和资源释放改动还应有取消、超时、短读、错误注入和平台测试。
