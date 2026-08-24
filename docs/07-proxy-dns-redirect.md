# 代理、DNS 与重定向

## 代理

```go
client := req.C().SetProxyURL("http://127.0.0.1:7890")
```

支持的常见 URL scheme 包括 HTTP、HTTPS、SOCKS5、SOCKS4 和 SOCKS4a：

```go
req.C().SetProxyURL("socks5://127.0.0.1:1080")
req.C().SetProxyURL("socks4://127.0.0.1:1080")
req.C().SetProxyURL("socks4a://127.0.0.1:1080")
```

SOCKS4 只支持 IPv4 目标并在本地解析域名；SOCKS4a 把域名交给代理解析。动态代理规则使用 `SetProxy(func(*http.Request) (*url.URL, error))`。CONNECT Header 可通过 transport 的 `SetProxyConnectHeader` 或 `SetGetProxyConnectHeader` 配置。

## Resolver 的两个入口

需要 HTTP/1、HTTP/2 和 HTTP/3 共用 resolver 时使用：

```go
client.SetDNSResolver(&net.Resolver{PreferGo: true})
```

`SetDNSResolver` 写入 transport 的 resolver，也同步到 HTTP/3。兼容入口 `SetResolver` 通过自定义 dialer 工作，只覆盖 HTTP/1 和 HTTP/2；随后调用 `SetDial`、`SetHosts` 或 `SetUnixSocket` 会替换它。

## DNS-over-TLS

```go
client := req.C().SetDNSOverTLSCloudflare()
```

内置入口还有 `SetDNSOverTLSGoogle`、`SetDNSOverTLSQuad9`、`SetDNSOverTLSAdGuard` 和 `SetDNSOverTLSAli`。自定义 provider：

```go
client.SetDNSOverTLS(req.DNSOverTLSProvider{
	ServerName: "dns.example.com",
	Addresses:  []string{"203.0.113.53:853"},
})
```

provider 的 `ServerName` 用于 TLS SNI 和证书校验，`Addresses` 必须是可直接拨号的 IP:port，避免解析 DoT 服务自身时产生递归依赖。

## 静态 Hosts

```go
client.SetHosts(map[string]string{
	"api.internal": "10.0.0.5",
	"v6.internal":  "::1",
})
```

`SetHosts` 是 fail-closed 映射：未列出的非 IP 域名立即返回 no such host，不回退系统 DNS。key 不含端口，匹配不区分大小写并做 IDNA 规范化；value 必须是 IPv4/IPv6 字面量。传入 map 会被复制。

限制：

- 只作用于 HTTP/1 和 HTTP/2 dial。
- 不能与代理组合；代理可能远端解析目标并绕过映射。
- `SetDial`、`SetResolver`、`SetUnixSocket` 会替换相同 dial 路径。

## 自定义 Dial 与 Unix Socket

```go
dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
client.SetDial(func(ctx context.Context, network, address string) (net.Conn, error) {
	return dialer.DialContext(ctx, network, address)
})
```

Unix Socket 使用 `SetUnixSocket(path)`。高级 TLS 接入点包括 `SetDialTLS` 和 `SetTLSHandshake`。这些入口会改变网络职责边界，应配套连接超时、取消和测试。

自定义 TLS hook 的连接所有权是强契约：成功时应返回包装/继续使用传入 plain connection 的 TLS 连接；如果另建一条独立连接，hook 必须在成功返回前关闭原 plain connection。失败、超时或取消时 req 会关闭原连接以及 hook 返回的非 nil/迟到连接。hook 还必须响应传入 context 或底层连接关闭；任何永不响应 context、连接关闭或外部信号的用户回调都无法由 Go 强制终止，应在高并发集成测试中覆盖该契约。

## 重定向策略

```go
client.SetRedirectPolicy(
	req.MaxRedirectPolicy(5),
	req.AllowedHostRedirectPolicy("api.example.com", "login.example.com"),
)
```

可用策略：

- `DefaultRedirectPolicy`：最多 10 次。
- `NoRedirectPolicy`：返回最后一个响应，不继续跳转。
- `SameHostRedirectPolicy`、`SameDomainRedirectPolicy`。
- `AllowedHostRedirectPolicy`、`AllowedDomainRedirectPolicy`。
- `AlwaysCopyHeaderRedirectPolicy`。
- `SensitiveHeadersRedirectPolicy`。

`SameDomainRedirectPolicy`、`AllowedDomainRedirectPolicy` 和 `SensitiveHeadersRedirectPolicy` 使用的是简单域名标签裁剪，不是 Public Suffix List/eTLD+1 判断；例如 `foo.co.uk` 与 `bar.co.uk` 可能被视为同域。它们适合兼容性规则，不应单独作为高价值凭据的安全边界。

自定义凭据 Header 可使用 `SensitiveHeadersRedirectPolicy` 删除明显跨域跳转中的值，但应同时用 `AllowedHostRedirectPolicy` 限定明确 hostname。`AllowedHostRedirectPolicy` 不约束 scheme 或端口；需要 exact-origin allowlist 时实现自定义 `RedirectPolicy`。多个策略按注册顺序执行，任一返回错误即停止跳转。

可编译 dial 示例：[custom_network_test.go](examples/custom_network_test.go)。
