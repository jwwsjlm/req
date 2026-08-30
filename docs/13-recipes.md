# 生产配方

以下组合是起点，不是所有服务通用的固定参数。

## 稳定 API Client

```go
func NewAPIClient(baseURL string) *req.Client {
	return req.C().
		SetBaseURL(baseURL).
		SetTimeout(30 * time.Second).
		SetCommonHeader("Accept", "application/json").
		SetMaxResponseSize(16 << 20).
		SetCommonRetryCount(2).
		SetCommonRetryBackoffInterval(200*time.Millisecond, 2*time.Second).
		SetCommonRetryCondition(func(resp *req.Response, err error) bool {
			if resp == nil || resp.Request == nil {
				return false
			}
			switch resp.Request.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, "QUERY":
			default:
				return false
			}
			if err != nil {
				return !errors.Is(err, context.Canceled) &&
					!errors.Is(err, context.DeadlineExceeded)
			}
			return resp != nil && (resp.GetStatusCode() == 429 || resp.GetStatusCode() >= 500)
		})
}
```

公共重试在这里仅覆盖只读方法。POST/PATCH 等写请求默认不重试；只有服务端支持幂等键、去重或其他业务幂等保证时，才在单个 request 上配置重试。

完整可编译版本：[production_client_test.go](examples/production_client_test.go)。

## JSON 成功/错误结果

```go
var result User
var apiErr APIError

resp, err := client.R().
	SetSuccessResult(&result).
	SetErrorResult(&apiErr).
	SetPathParam("id", userID).
	Get("/users/{id}")
if err != nil {
	return User{}, err
}
if resp.IsErrorState() {
	return User{}, fmt.Errorf("api error: %s", apiErr.Message)
}
return result, nil
```

## 多账号 Cookie 隔离

```go
template := req.C().
	SetCookieJarFactory(func() http.CookieJar {
		jar, _ := cookiejar.New(nil)
		return jar
	}).
	ImpersonateChromeWithOS(req.BrowserOSWindows)

accountA := template.Clone().SetCommonBearerAuthToken(tokenA)
accountB := template.Clone().SetCommonBearerAuthToken(tokenB)
```

## 自定义认证 Header 与安全重定向

```go
client := req.C().
	SetCommonHeader("X-API-Key", apiKey).
	SetRedirectPolicy(
		req.MaxRedirectPolicy(5),
		req.AllowedHostRedirectPolicy("api.example.com", "login.example.com"),
		req.SensitiveHeadersRedirectPolicy("X-API-Key"),
	)
```

这里的 host allowlist 是主要安全边界。`SensitiveHeadersRedirectPolicy` 的同域判断不使用 Public Suffix List；若 scheme、端口也属于凭据边界，请改用自定义 exact-origin policy。

## 浏览器 profile + HTTP/3 回退

```go
client := req.C().ImpersonateChromeWithOS(req.BrowserOSWindows)
client.Transport.EnableHTTP3()
client.Transport.EnableHTTP3FallbackOnError().SetHTTP3AltSvcFailureCooldown(30 * time.Second)
```

## 大文件下载

```go
const maxArchiveSize int64 = 4 << 30
tempPath := "archive.zip.part"

resp, err := client.R().
	SetMaxResponseSize(maxArchiveSize).
	SetOutputFile(tempPath).
	Get(downloadURL)
if err != nil {
	_ = os.Remove(tempPath)
	return err
}
if !resp.IsSuccessState() {
	_ = os.Remove(tempPath)
	return fmt.Errorf("download failed: %s", resp.GetStatus())
}
if err := os.Rename(tempPath, "archive.zip"); err != nil {
	_ = os.Remove(tempPath)
	return err
}
```

正的 `SetMaxResponseSize` 会同时检查可信的声明长度并限制实际流式读取，避免无界磁盘写入。HEAD 只能用于提前判断，不能替代 GET 期间的真实字节上限；下载到临时文件，成功后再 rename/替换，并在失败时清理。rename 是否原子、能否覆盖既有目标取决于操作系统和文件系统。支持 Range 时可用 `NewParallelDownload`，同样需要总大小和临时目录预算。

## 流式响应

```go
resp, err := client.R().
	SetMaxResponseSize(64 << 20).
	DisableAutoReadResponse().
	Get(url)
if err != nil {
	return err
}
defer resp.Body.Close()
_, err = io.Copy(dst, resp.Body)
```

## 自定义网络

```go
client := req.C().SetDNSOverTLSCloudflare()
```

或注入 dialer：

```go
dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
client.Transport.SetDial(func(ctx context.Context, network, address string) (net.Conn, error) {
	return dialer.DialContext(ctx, network, address)
})
```

自定义 `SetDial` 后，解析策略由这个 dial path 负责。完整示例：[custom_network_test.go](examples/custom_network_test.go)。

## 仅失败时查看 Dump

```go
resp, err := client.R().EnableDumpWithoutBody().Get(url)
if err != nil || !resp.IsSuccessState() {
	log.Printf("request failed: err=%v dump=%s", err, resp.Dump())
}
```

即使只在失败时输出，也应在写日志前处理 token、Cookie 和个人信息。
