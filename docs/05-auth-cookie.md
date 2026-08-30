# 认证与 Cookie

## Bearer、Basic 和自定义 scheme

单次请求：

```go
resp, err := client.R().
	SetBearerAuthToken(token).
	Get("https://api.example.com/me")
```

公共认证：

```go
client := req.C().SetCommonBearerAuthToken(token)
```

其他入口：

- `SetBasicAuth` / `SetCommonBasicAuth`
- `SetCommonDigestAuth`
- `SetAuthSchemeToken` / `SetCommonAuthSchemeToken`
- `SetAuthToken` / `SetCommonAuthToken`

不同账号不要共享同一个携带公共 token 的 client；可以从无凭据模板 `Clone()` 后分别设置。

## 默认 CookieJar

`req.C()` 默认创建内存 CookieJar。复用同一个 client 时，服务端的 `Set-Cookie` 会自动保存并在符合域、路径和安全规则时发送。

```go
_, err := client.R().
	SetBodyJsonString(`{"username":"alice","password":"secret"}`).
	Post("https://example.com/login")
if err != nil {
	return err
}

resp, err := client.R().Get("https://example.com/me")
```

单次手工 Cookie：

```go
resp, err := client.R().
	SetCookies(&http.Cookie{Name: "session", Value: "abc"}).
	Get("https://example.com/me")
```

公共 Cookie 使用 `SetCommonCookies`。读取和清理使用 `GetCookies(url)`、`ClearCookies()`。

## 自定义与禁用 CookieJar

```go
jar, err := cookiejar.New(nil)
if err != nil {
	return err
}
client.SetCookieJar(jar)
```

`SetCookieJar(nil)` 禁用自动 Cookie。若 clone 需要新 jar：

```go
client.SetCookieJarFactory(func() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
})
```

兼容旧签名 `func() *cookiejar.Jar`。传入其他类型会 panic，因此 factory 应在初始化阶段配置并由测试覆盖。

## 重定向中的凭据

Go 标准库会对部分敏感 Header 应用跨域规则，但自定义认证 Header 需要显式保护：

```go
client.SetRedirectPolicy(
	req.MaxRedirectPolicy(10),
	req.AllowedHostRedirectPolicy("api.example.com", "login.example.com"),
	req.SensitiveHeadersRedirectPolicy("X-API-Key", "X-Auth-Token"),
)
```

`SensitiveHeadersRedirectPolicy` 的“同域”判断是简单的标签裁剪，不使用 Public Suffix List；例如 `foo.co.uk` 与 `bar.co.uk` 可能被判为同域。凭据安全场景应先用 `AllowedHostRedirectPolicy` 限定明确 hostname；如果还需约束 scheme、端口或完整 origin，请实现自定义 `RedirectPolicy`。不要对不可信目标使用 `AlwaysCopyHeaderRedirectPolicy("Authorization")`，它会强制复制指定 Header。更多规则见 [代理、DNS 与重定向](07-proxy-dns-redirect.md)。

完整本地示例：[auth_cookie_test.go](examples/auth_cookie_test.go)。
