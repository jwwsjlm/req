# 构建请求

## URL、BaseURL 与路径参数

```go
client := req.C().SetBaseURL("https://api.example.com")

resp, err := client.R().
	SetPathParam("user", "a/b").
	Get("/users/{user}")
```

`SetPathParam` 会进行路径转义。只有确定值已经是目标 URL 片段时才使用 `SetPathRawParam`；raw 值不会自动转义。

公共版本为 `SetCommonPathParam`、`SetCommonPathRawParam` 等。绝对请求 URL 不会拼接 BaseURL。

## Query 参数

```go
client := req.C().
	SetCommonQueryParam("lang", "zh-CN").
	AddCommonQueryParams("tag", "go", "http")

resp, err := client.R().
	SetQueryParam("page", "2").
	SetQueryParam("lang", "en"). // 覆盖 client 的 lang
	AddQueryParams("field", "id", "name").
	Get("https://api.example.com/users")
```

可选入口包括 `SetQueryParams`、`SetQueryParamsAnyType`、`SetQueryParamsFromValues`、`SetQueryParamsFromStruct` 和 `SetQueryString`。最终编码使用标准库 `url.Values.Encode`，key 顺序稳定；同一 key 的多值顺序保留。

## Header 与顺序

```go
resp, err := client.R().
	SetHeader("Accept", "application/json").
	SetHeaderValues("X-Tag", "a", "b").
	SetHeaderOrder("host", "user-agent", "accept", "x-tag").
	Get("https://example.com")
```

常用方法：`SetHeaders`、`SetHeaderAny`、`SetHeaderValues`、`SetHeaderMultiValues`。需要保留非规范大小写时使用 `SetHeaderNonCanonical`；这是指纹或特殊协议兼容能力，不建议普通业务依赖 Header 大小写。

HTTP/2/HTTP/3 还可用 `SetPseudoHeaderOrder` 控制伪 Header 顺序。公共方法名 `SetCommonPseudoHeaderOder` 为兼容保留了历史拼写。

## JSON、XML 与任意 body

```go
payload := struct {
	Name string `json:"name"`
}{Name: "demo"}

resp, err := client.R().
	SetBody(payload).
	Post("https://api.example.com/items")
```

`SetBody` 支持 string、`[]byte`、`io.Reader`、map 和 struct。需要明确格式时可用：

- `SetBodyJsonMarshal`、`SetBodyJsonString`、`SetBodyJsonBytes`
- `SetBodyXmlMarshal`、`SetBodyXmlString`、`SetBodyXmlBytes`
- `SetBodyString`、`SetBodyBytes`
- `SetContentType`、`SetContentLength`

## 表单和 multipart

URL encoded 表单：

```go
resp, err := client.R().
	SetFormData(map[string]string{"username": "alice"}).
	Post("https://example.com/login")
```

需要字段顺序时使用 `SetOrderedFormData("key", "value", ...)`，参数数量必须为偶数。multipart 上传见 [上传与下载](08-upload-download.md)。

## HTTP 方法

终结方法包括 `Get`、`Post`、`Put`、`Patch`、`Delete`、`Head`、`Options`、`Query` 和通用 `Send`。`QUERY` 对应 RFC 10008。

client 默认允许 GET payload，可通过 `DisableAllowGetMethodPayload` 关闭。依赖 GET body 的接口可能被代理、缓存或服务端忽略，普通查询优先放在 URL Query 中。

## 成功和错误结果类型

```go
var okResult APIResult
var errorResult APIError

resp, err := client.R().
	SetSuccessResult(&okResult).
	SetErrorResult(&errorResult).
	Get("https://api.example.com/items/42")
```

默认 2xx 为成功、4xx/5xx 为错误、其余为未知状态。可使用 `SetResultStateCheckFunc` 自定义。详见 [错误处理](04-error-handling.md)。
