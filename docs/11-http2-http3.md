# HTTP/2 与 HTTP/3

## 协议选择

默认 HTTPS 请求通过 TLS ALPN 在 HTTP/1.1 和 HTTP/2 之间协商。可显式选择：

```go
client.EnableForceHTTP1()
client.EnableForceHTTP2()
client.DisableForceHttpVersion()
```

H2C 是无 TLS 的 HTTP/2：

```go
client.EnableH2C()
```

只有明确知道目标支持 h2c 时使用；公网 HTTPS 不需要它。

## HTTP/2 调优与指纹

```go
client.SetHTTP2SettingsFrame(
	http2.Setting{ID: http2.SettingHeaderTableSize, Val: 65536},
	http2.Setting{ID: http2.SettingInitialWindowSize, Val: 6291456},
).
	SetHTTP2ConnectionFlow(15663105).
	SetHTTP2InitialStreamID(3)
```

其他入口：

- `SetHTTP2HeaderPriority`
- `SetHTTP2PriorityFrames`
- `SetHTTP2MaxHeaderListSize`
- `SetHTTP2StrictMaxConcurrentStreams`
- `SetHTTP2ReadIdleTimeout`
- `SetHTTP2PingTimeout`
- `SetHTTP2WriteByteTimeout`
- `SetCommonHeaderOrder`
- `SetCommonPseudoHeaderOder`

这些参数既会影响协议行为，也可能成为指纹的一部分。普通 API client 不应盲目复制某个浏览器的单个数值；优先使用完整浏览器 profile，或保持默认值。

## 启用 HTTP/3

```go
client := req.C().
	EnableHTTP3().
	EnableHTTP3FallbackOnError().
	SetHTTP3AltSvcFailureCooldown(30 * time.Second)
```

`EnableHTTP3` 建立 HTTP/3 能力和 Alt-Svc 状态；服务端可通过 Alt-Svc 引导后续请求使用 H3。`EnableForceHTTP3` 会对 HTTPS 强制尝试 H3：

```go
client.EnableHTTP3FallbackOnError().EnableForceHTTP3()
```

建议先开启 fallback 再强制，除非业务明确要求 H3 失败直接返回错误。

## 回退和可重放 body

`EnableHTTP3FallbackOnError` 允许强制或 Alt-Svc 选择的 H3 在失败后回退 H2/H1，但前提是请求尚未变成不可重放状态。流式 reader、已经部分发送的 body 或有副作用的业务语义，都可能使透明回退不安全或不可行。

`SetHTTP3AltSvcFailureCooldown` 控制失败端点暂时跳过多久：0 使用默认值，负值禁用 cooldown。

## HTTP/3 SETTINGS、Datagram 与 Extended CONNECT

```go
client.SetHTTP3AdditionalSetting(
	req.HTTP3SettingQpackMaxTableCapacity,
	65536,
).
	SetHTTP3MaxResponseHeaderBytes(256 << 10).
	SetHTTP3Grease()
```

公开 setting 常量包括 QPACK table、blocked streams、max field section、CONNECT protocol、H3 Datagram 和 WebTransport。未知 setting 可通过数值 ID 添加，但需确认对端和协议约束。

Datagram 与 Extended CONNECT：

```go
client.EnableHTTP3Datagrams().EnableHTTP3ExtendedConnect()
```

启用 Datagram 会同步 QUIC 层配置。只有上层协议真正需要时才开启。

## QUIC 与 HTTP/3 TLS

平衡性能 profile：

```go
client.SetHTTP3QUICPerformanceProfile()
```

自定义：

```go
client.SetHTTP3QUICConfig(&quic.Config{
	HandshakeIdleTimeout: 5 * time.Second,
	MaxIdleTimeout:       45 * time.Second,
	KeepAlivePeriod:      15 * time.Second,
})
```

传入 config 会被 clone。HTTP/3 专用 TLS 使用 `SetHTTP3TLSClientConfig`，不会把 uTLS ClientHello 直接带入 QUIC；具体边界见 [浏览器与 TLS 指纹](10-browser-tls-fingerprint.md)。

## 排障顺序

1. 先用默认 H1/H2 验证业务接口。
2. 开启 `EnableHTTP3` 与 fallback，观察 Alt-Svc 路径。
3. 再尝试 `EnableForceHTTP3` 区分服务端 H3 可达性问题。
4. 检查 UDP、防火墙、代理、DNS、证书和 QUIC idle timeout。
5. H3 可提供核心 `TraceInfo`，但 DNS 与历史 TCP 字段可能为零或不代表 TCP；结合 debug/dump 并与 H2 结果对照。

可编译配置示例：[browser_http3_test.go](examples/browser_http3_test.go)。
