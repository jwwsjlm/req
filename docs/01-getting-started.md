# 快速入门

## 安装

在你的 Go module 中安装本 fork：

```sh
go get github.com/jwwsjlm/req/v3
```

导入路径必须带 `/v3`：

```go
import req "github.com/jwwsjlm/req/v3"
```

Go 工具链要求以所使用版本中的 `go.mod` 为准。升级依赖后应至少执行 `go test ./...`。

## 第一个 GET

```go
client := req.C()

resp, err := client.R().
	SetQueryParam("q", "golang").
	SetHeader("Accept", "application/json").
	Get("https://api.example.com/search")
if err != nil {
	log.Fatal(err)
}
if !resp.IsSuccessState() {
	log.Fatalf("unexpected status: %s", resp.GetStatus())
}

fmt.Println(resp.String())
```

`C()`/`NewClient()` 创建 client，`R()`/`NewRequest()` 创建单次请求，`Get`、`Post` 等方法发送请求。

## 自动解析 JSON

```go
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var user User
resp, err := client.R().
	SetSuccessResult(&user).
	Get("https://api.example.com/users/42")
if err != nil {
	log.Fatal(err)
}
if !resp.IsSuccessState() {
	log.Fatalf("request failed: %s", resp.GetStatus())
}

fmt.Println(user.Name)
```

也可以请求完成后调用 `resp.Into(&user)`。`SetSuccessResult` 更适合同时配置成功和错误结果类型。

## 生产中复用 client

```go
var apiClient = req.C().
	SetBaseURL("https://api.example.com").
	SetTimeout(30 * time.Second).
	SetCommonHeader("Accept", "application/json").
	SetMaxResponseSize(16 << 20)
```

复用 client 可以复用连接池、TLS 会话和 CookieJar。完成初始化后，将公共配置视为只读；每次业务调用使用新的 `client.R()`。

## 两种发送风格

直接构建并发送：

```go
resp, err := client.R().Get("https://example.com")
```

先由 client 选择方法，再调用 `Do`：

```go
resp := client.Get("https://example.com").
	SetQueryParam("page", "1").
	Do()
if resp.Err != nil {
	log.Fatal(resp.Err)
}
```

业务代码通常优先使用第一种，因为显式返回 `error`。`MustGet` 等 `Must*` 方法遇错会 panic，只适合测试或初始化阶段。

## 下一步

- 核心对象：[Client、Request 与 Response](02-client-request-response.md)
- 完整参数构造：[构建请求](03-building-requests.md)
- 错误模型：[错误处理](04-error-handling.md)
- 可运行版本：[basic_test.go](examples/basic_test.go)
