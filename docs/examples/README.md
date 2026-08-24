# 可编译示例

本目录使用 `package examples` 的测试/Example 文件保存文档代码。测试不访问公网，可随全仓测试稳定编译和运行。

| 主题 | 文件 | 覆盖 |
| --- | --- | --- |
| basic | [basic_test.go](basic_test.go) | Client、Request、Path/Query/Header、成功结果解析 |
| production-client | [production_client_test.go](production_client_test.go) | 超时、响应限制、backoff、retry condition |
| auth-cookie | [auth_cookie_test.go](auth_cookie_test.go) | Bearer token、默认 CookieJar、会话复用 |
| upload-download | [upload_download_test.go](upload_download_test.go) | 内存 multipart、writer 下载 |
| middleware | [middleware_test.go](middleware_test.go) | before/after middleware |
| browser-http3 | [browser_http3_test.go](browser_http3_test.go) | Chrome profile、H3、fallback、cooldown |
| tls-fingerprint | [tls_fingerprint_test.go](tls_fingerprint_test.go) | 固定 seed 的随机 ALPN 指纹、严格解析捕获的 ClientHello |
| custom-network | [custom_network_test.go](custom_network_test.go) | 自定义 `SetDial` 与 `net.Dialer` |

运行：

```sh
go test ./docs/examples -count=1
go test ./... -count=1
```

其中 `Example_*` 仅验证配置可构造；需要真实网络、代理、DNS-over-TLS、浏览器指纹或 HTTP/3 的行为必须在受控集成环境另行验证。
