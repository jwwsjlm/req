# Go 与 req 零基础入门：第一次使用这个增强版 req

这篇文档写给两类读者：

- 第一次使用 `github.com/jwwsjlm/req/v3` 的人。
- Go 还不熟，希望一边发 HTTP 请求，一边理解 Go 基础语法的人。

你不需要先学完 Go。跟着本文从空目录开始，最终会得到一个可复用、可取消、可测试的 API Client。

> 这个仓库是基于 `imroc/req` 扩展的增强版。安装和导入时必须使用
> `github.com/jwwsjlm/req/v3`，不要混用 `github.com/imroc/req/v3`。

## 学完以后你能做什么

完成本文后，你应该能够：

1. 创建 Go module，安装并导入这个版本的 req。
2. 发送 GET 请求，添加 Query 和 Header，读取响应。
3. 理解 `Client -> Request -> Response` 这条调用链。
4. 区分 Go 错误、HTTP 错误状态和 JSON 业务错误。
5. 用 struct 接收 JSON，并把请求封装成自己的 API Client。
6. 用 `context` 设置一次调用的超时或取消。
7. 用 `httptest` 写一个完全不访问公网的测试。

## 1. 准备 Go 环境

先在终端执行：

```sh
go version
```

本仓库当前 `go.mod` 使用 Go `1.26.7`。以后版本要求可能变化，最可靠的依据始终是仓库根目录的 `go.mod`。如果本机 Go 版本过低，请先安装或升级 Go。

你还会用到这些命令：

```sh
go run .          # 编译并运行当前 module 的 main 包
go test ./...     # 测试当前 module 下的所有包
go fmt ./...      # 格式化 Go 代码
go mod tidy       # 整理实际需要的依赖
```

这里的 `./...` 可以理解为“当前目录以及下面的所有 Go package”。

## 2. 从空目录创建项目

新建项目目录并初始化 module：

```sh
mkdir req-beginner
cd req-beginner
go mod init example.com/req-beginner
go get github.com/jwwsjlm/req/v3@latest
```

`@latest` 适合第一次练习。正式项目为了让构建结果可复现，通常应在验证后固定 `go.mod` 中的具体版本，并提交 `go.mod` 与 `go.sum`。

执行后会看到两个重要文件：

- `go.mod`：记录 module 名、Go 版本和直接依赖。
- `go.sum`：记录依赖内容的校验信息；它不是依赖清单的重复副本。

`example.com/req-beginner` 是你这个练习项目的 module path。真实项目通常使用公司的域名或代码托管地址，例如 `github.com/your-name/your-project`。

## 3. 第一个 GET 请求

在项目目录中新建 `main.go`：

```go
package main

import (
	"fmt"
	"log"
	"time"

	req "github.com/jwwsjlm/req/v3"
)

func main() {
	client := req.C().
		SetTimeout(10 * time.Second)

	resp, err := client.R().
		SetQueryParam("from", "req-beginner").
		SetHeader("Accept", "application/json").
		Get("https://httpbin.org/get")
	if err != nil {
		log.Fatal(err)
	}
	if !resp.IsSuccessState() {
		log.Fatalf("HTTP 请求失败: %s\n%s", resp.GetStatus(), resp.String())
	}

	fmt.Println(resp.String())
}
```

运行：

```sh
go run .
```

如果网络可以访问 `httpbin.org`，终端会打印一段 JSON。若所在网络无法访问它，可直接跳到[第 8 节](#8-写一个不访问公网的测试)，用本地测试服务器完成练习。

### 顺便认识 Go 语法

这段程序已经包含了不少 Go 基础：

- `package main`：声明这是一个可执行程序；它还需要一个 `main` 函数作为入口。
- `import`：导入标准库和第三方 package。
- `req "github.com/jwwsjlm/req/v3"`：左边的 `req` 是当前文件使用的包名，右边是 module 中的导入路径。
- `func main()`：定义程序入口函数。
- `:=`：声明变量并让编译器推断类型。
- `resp, err := ...`：Go 函数可以返回多个值，这里同时得到响应和错误。
- `if err != nil`：Go 通常显式检查错误，而不是依赖异常处理。
- `client.R().SetHeader(...).Get(...)`：每个方法返回下一步需要的对象，因此可以链式调用。

代码中的点号不是特殊的 req 语法，而是普通的 Go 方法调用。

## 4. 先建立正确的三个对象模型

req 最重要的调用关系只有三层：

| 对象 | 创建方式 | 生命周期 | 应该放什么 |
| --- | --- | --- | --- |
| `*req.Client` | `req.C()` 或 `req.C()` | 长期复用 | BaseURL、总超时、公共 Header、Cookie、代理、连接池配置 |
| `*req.Request` | `client.R()` | 一次请求 | 本次 Query、Path、Header、Body、Context、结果类型 |
| `*req.Response` | `Get`、`Post` 等返回 | 一次响应 | 状态码、Header、响应体、解析结果、耗时和 TLS 信息 |

可以把它记成：

```text
长期 Client --R()--> 单次 Request --Get/Post()--> Response
```

推荐：

```go
client := req.C().
	SetBaseURL("https://api.example.com").
	SetTimeout(10 * time.Second).
	SetCommonHeader("Accept", "application/json")

resp, err := client.R().
	SetQueryParam("page", "1").
	Get("/users")
```

不推荐在每个业务函数里都重新 `req.C()`。长期复用 client 才能复用连接、Cookie 和公共配置。client 初始化完成后，通常把公共配置当作只读；每次调用创建新的 `client.R()`。

## 5. `err == nil` 不代表 HTTP 成功

这是 HTTP 新手最容易踩的坑。

```go
resp, err := client.R().Get("https://example.com/not-found")
if err != nil {
	// DNS、连接、TLS、超时、取消、请求构造等错误通常走这里。
	return err
}
if !resp.IsSuccessState() {
	// 服务端已经返回响应，但状态是 4xx 或 5xx，通常走这里。
	return fmt.Errorf("unexpected HTTP status: %s", resp.GetStatus())
}
```

三种失败需要分别理解：

| 情况 | 例子 | 怎么判断 |
| --- | --- | --- |
| 请求过程失败 | DNS 错误、连接被拒绝、超时、TLS 校验失败 | `err != nil` |
| HTTP 状态失败 | `404 Not Found`、`500 Internal Server Error` | `resp.IsErrorState()` 或 `!resp.IsSuccessState()` |
| 业务失败 | HTTP 200，但 JSON 中 `code != 0` | 按接口协议检查解析后的字段 |

req 的发送方法会返回一个非 `nil` 的 `*req.Response` 包装对象，但请求过程失败时，其中嵌入的标准库 `*http.Response` 可能还不存在。仍然要先处理 `err`，再读取状态码、Header 或响应体。

## 6. 用 struct 接收 JSON

假设接口成功时返回：

```json
{"id": 42, "name": "Ada"}
```

先定义对应的 Go struct：

```go
type DemoUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
```

下面把解析过程放进一个返回 `error` 的辅助函数，因此其中可以直接 `return err`。把它和 `DemoUser` 放在 `main.go` 的 `main` 函数外面：

```go
func fetchDemoUser(client *req.Client, url string) (DemoUser, error) {
	var user DemoUser
	resp, err := client.R().
		SetSuccessResult(&user).
		Get(url)
	if err != nil {
		return DemoUser{}, err
	}
	if !resp.IsSuccessState() {
		return DemoUser{}, fmt.Errorf("get user failed: %s", resp.GetStatus())
	}
	return user, nil
}
```

这里可以学到三个 Go 概念：

- `type User struct` 把相关字段组成一个新类型。
- ``json:"name"`` 是 struct tag，告诉 JSON 解码器字段在 JSON 中叫什么。
- `&user` 取得变量地址。req 需要通过这个指针把解码结果写回 `user`。

发送 JSON 时可以反过来把 struct 编码成请求体：

```go
type DemoCreateUserRequest struct {
	Name string `json:"name"`
}

resp, err := client.R().
	SetBodyJsonMarshal(DemoCreateUserRequest{Name: "Ada"}).
	Post("/users")
```

`SetBodyJsonMarshal` 会编码 JSON 并设置合适的 `Content-Type`。不要手工拼接包含用户输入的 JSON 字符串。

## 7. 从脚本进化成自己的 API Client

真实项目里，建议把 req 封装在自己的业务类型中。这样 BaseURL、超时、错误处理不会散落在每个调用点。

接下来开始正式的业务封装。新建 `api.go`；这里使用新的 `User` 业务类型，不会与上一节只用于演示 JSON 的 `DemoUser` 重名：

```go
package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	req "github.com/jwwsjlm/req/v3"
)

type APIClient struct {
	http *req.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		http: req.C().
			SetBaseURL(baseURL).
			SetTimeout(10 * time.Second).
			SetCommonHeader("Accept", "application/json"),
	}
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (api *APIClient) GetUser(ctx context.Context, id int) (User, error) {
	var user User
	var apiErr APIError

	resp, err := api.http.R().
		SetContext(ctx).
		SetPathParam("id", strconv.Itoa(id)).
		SetSuccessResult(&user).
		SetErrorResult(&apiErr).
		Get("/users/{id}")
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	if !resp.IsSuccessState() {
		if apiErr.Message != "" {
			return User{}, fmt.Errorf("get user: %s: %s", resp.GetStatus(), apiErr.Message)
		}
		return User{}, fmt.Errorf("get user: %s", resp.GetStatus())
	}

	return user, nil
}
```

这段代码对应几个常见 Go 设计：

- `APIClient` 是你自己的类型，内部保存第三方库的 `*req.Client`。
- `NewAPIClient` 是构造函数惯例；Go 不强制构造函数，但常用 `NewXxx` 命名。
- `(api *APIClient)` 是方法接收者，表示 `GetUser` 属于 `APIClient`。
- `context.Context` 把取消和 deadline 从调用方传到网络请求。
- `fmt.Errorf("...: %w", err)` 保留原错误，调用方之后仍可用 `errors.Is` 判断。
- 出错时返回 `User{}`，它是 `User` 的零值。

调用它：

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

api := NewAPIClient("https://api.example.com")
user, err := api.GetUser(ctx, 42)
if err != nil {
	log.Fatal(err)
}
fmt.Println(user.Name)
```

`defer cancel()` 会在当前函数结束前释放与这个 context 关联的资源。即使 client 已有总超时，业务调用仍值得传入 context，因为上层可以主动取消整个调用链。

## 8. 写一个不访问公网的测试

Go 标准库的 `httptest` 可以在本机启动临时 HTTP 服务。它速度快、可重复，也不会依赖第三方网站是否在线。

新建 `api_test.go`：

```go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClientGetUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/42" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"name":"Ada"}`))
	}))
	defer server.Close()

	api := NewAPIClient(server.URL)
	user, err := api.GetUser(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 42 || user.Name != "Ada" {
		t.Fatalf("unexpected user: %#v", user)
	}
}
```

运行：

```sh
go test ./...
```

这又引入了几个 Go 基础：

- 测试文件以 `_test.go` 结尾。
- 测试函数以 `Test` 开头，并接收 `*testing.T`。
- `httptest.NewServer` 返回一个只在测试期间存在的本地服务。
- `defer server.Close()` 保证测试结束时关闭监听端口。
- `t.Fatal` 报告失败并停止当前测试。

本仓库也保存了可直接编译的同类示例：[beginner_test.go](examples/beginner_test.go)。在仓库根目录运行：

```sh
go test ./docs/examples -run Beginner -count=1
```

## 9. 最常用的请求写法

先掌握下面几种组合就够了，完整选项见[构建请求](03-building-requests.md)。

### Query 参数

```go
resp, err := client.R().
	SetQueryParam("q", "golang").
	SetQueryParamAny("page", 1).
	Get("/search")
```

### 单次 Header

```go
resp, err := client.R().
	SetHeader("X-Request-ID", "demo-001").
	Get("/users/42")
```

### Bearer Token

```go
resp, err := client.R().
	SetAuthToken("your-token").
	Get("/me")
```

如果同一个 client 的后续请求都使用同一 token，可在 client 上调用 `SetCommonAuthToken`。认证和 Cookie 的完整说明见[认证与 Cookie](05-auth-cookie.md)。

### POST JSON

```go
resp, err := client.R().
	SetBodyJsonMarshal(map[string]any{
		"title": "learn Go",
		"done":  false,
	}).
	Post("/todos")
```

## 10. 这个增强版有哪些进阶能力

普通 JSON API 不需要先打开高级功能。等基础请求跑通后，再按目标选择：

- 浏览器 Header、TLS 和 HTTP/2 profile：见[浏览器与 TLS 指纹](10-browser-tls-fingerprint.md)。
- HTTP/2、HTTP/3、Alt-Svc 与失败回退：见[HTTP/2 与 HTTP/3](11-http2-http3.md)。
- DNS-over-TLS、代理、重定向策略：见[代理、DNS 与重定向](07-proxy-dns-redirect.md)。
- 上传、下载和大响应：见[上传与下载](08-upload-download.md)。

例如，在确有浏览器兼容需求的授权场景中，可以使用一致的 Chrome profile：

```go
client := req.C().ImpersonateChromeWithOS(req.BrowserOSWindows)
client.Transport.EnableHTTP3()
client.Transport.EnableHTTP3FallbackOnError()
```

这不只是修改 `User-Agent`，还会配置对应的 Header、TLS 和协议行为。它也不等于能够绕过所有站点策略；账号、Cookie、IP、访问频率和页面脚本仍是不同层面的问题。

## 11. 新手最常见的错误

### 导入路径漏了 `/v3`

正确：

```go
import req "github.com/jwwsjlm/req/v3"
```

安装命令也必须是：

```sh
go get github.com/jwwsjlm/req/v3@latest
```

### 混用了上游和这个 fork

不要在同一份代码中一会儿导入 `github.com/imroc/req/v3`，一会儿导入 `github.com/jwwsjlm/req/v3`。即使类型名字相同，它们也是不同的 Go package，类型不能直接互换。

### 每次请求都创建新 client

这样会丢失连接、Cookie 和公共配置复用。通常一个目标服务或一个登录会话使用一个长期 client。

### 只检查 `err`

服务端返回 404/500 时，网络过程可能完全成功。请求后同时检查 `err` 和 `resp.IsSuccessState()`。

### 在 `err != nil` 时直接读取 HTTP 状态

req 返回的外层 `*req.Response` 不会是 `nil`，但连接或 TLS 等阶段失败时，里面可能还没有标准库 HTTP 响应。先处理 `err`，再读取状态和 body。

### 流式响应忘记关闭 Body

默认自动读取响应时通常直接使用 `resp.String()` 即可。只有调用 `DisableAutoReadResponse()` 后，才需要自己读取并关闭：

```go
resp, err := client.R().DisableAutoReadResponse().Get("/stream")
if err != nil {
	return err
}
defer resp.Body.Close()
```

### 生产环境一直开着 `DevMode`

调试输出可能包含 Header、Cookie、Token 和请求体。只在受控的本地排障阶段启用，更多说明见[中间件与可观测性](09-middleware-observability.md)。

### 无条件重试 POST

写操作可能已经在服务端成功，只是响应丢失。除非接口有幂等键或明确的去重保证，否则不要自动重试有副作用的请求。见[超时、重试与 Context](06-timeout-retry-context.md)。

## 12. 用这个项目继续学习 Go

当你已经能完成本文示例，可以按下面的顺序阅读源码和文档：

| 想学的 Go 概念 | 从哪里看 | 能看到什么 |
| --- | --- | --- |
| struct、指针和方法 | [Client、Request 与 Response](02-client-request-response.md) | 类型职责、方法接收者、对象生命周期 |
| map、slice 和 URL 编码 | [构建请求](03-building-requests.md) | Query、Header、多值参数和覆盖关系 |
| error 与错误包装 | [错误处理](04-error-handling.md) | 网络错误、HTTP 状态和结果类型 |
| context 与取消传播 | [超时、重试与 Context](06-timeout-retry-context.md) | deadline、取消、重试边界 |
| interface 和中间件 | [中间件与可观测性](09-middleware-observability.md) | 函数类型、包装器、请求生命周期 hook |
| 并发与资源释放 | [性能与稳定性](12-performance-stability.md) | goroutine、连接池、响应体和内存上限 |
| 标准库测试工具 | [可编译示例](examples/README.md) | `testing`、`httptest`、本地 TLS 服务 |

读源码时不必从最大的文件开始。可以先找到教程中调用的方法，再沿着定义跳转：

```sh
go doc github.com/jwwsjlm/req/v3.Client.SetTimeout
go doc github.com/jwwsjlm/req/v3.Request.SetQueryParam
go doc github.com/jwwsjlm/req/v3.Response.IsSuccessState
```

然后用编辑器的“转到定义”查看实现。遇到 `interface`、`context`、`io.Reader` 等标准库概念时，再打开对应的 Go 官方 package 文档。

## 13. 建议的下一步

1. 把第一段 GET 程序在本机跑通。
2. 把 URL、Query 和 JSON struct 换成你实际要调用的接口。
3. 按第 7 节封装自己的 API Client。
4. 为成功、404 和超时至少各写一个本地测试。
5. 再阅读[快速入门](01-getting-started.md)和[生产配方](13-recipes.md)。
6. 查方法名时使用 [API 索引](15-api-index.md) 或 `go doc`。

如果只记住一句话：**长期复用 Client，每次创建 Request，先检查 Go error，再检查 HTTP 状态。**
