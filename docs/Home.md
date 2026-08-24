# req 中文 Wiki 首页

`req` 是一个链式 Go HTTP 客户端。本仓库在原有能力上补充了浏览器 profile、TLS/HTTP 指纹、HTTP/3、DNS、响应大小限制、流式 multipart 和兼容性加固。

## 推荐阅读顺序

| 阶段 | 文档 | 解决的问题 |
| --- | --- | --- |
| Go 零基础 / 第一次使用这个 fork | [Go 与 req 零基础入门](00-go-req-beginner.md) | 从空目录开始，学习 Go 基础、首个请求、业务 Client 和本地测试 |
| 已有 Go module / 快速查用法 | [快速入门](01-getting-started.md) | 安装、首个请求、结果解析和 client 复用 |
| 理解模型 | [Client、Request 与 Response](02-client-request-response.md) | 三个核心对象的生命周期和覆盖关系 |
| 写业务请求 | [构建请求](03-building-requests.md) | URL、参数、Header、Body、表单和结果类型 |
| 生产可靠性 | [错误处理](04-error-handling.md)、[超时、重试与 Context](06-timeout-retry-context.md) | 区分网络错误与 HTTP 错误，控制取消和重试 |
| 会话与网络 | [认证与 Cookie](05-auth-cookie.md)、[代理、DNS 与重定向](07-proxy-dns-redirect.md) | 登录态、解析策略、代理和凭据保护 |
| 数据传输 | [上传与下载](08-upload-download.md) | 流式 multipart、输出 writer、并行下载 |
| 排障 | [中间件与可观测性](09-middleware-observability.md) | middleware、trace、dump 和日志 |
| 指纹和协议 | [浏览器与 TLS 指纹](10-browser-tls-fingerprint.md)、[HTTP/2 与 HTTP/3](11-http2-http3.md) | 固定版本 profile、协议边界、选择和回退 |
| 长期维护 | [性能与稳定性](12-performance-stability.md)、[迁移与兼容](14-migration-compatibility.md)、[上游项目、致谢与许可](16-upstream-credits.md) | 资源边界、兼容原则、升级清单和第三方归属 |
| 快速查找 | [生产配方](13-recipes.md)、[API 索引](15-api-index.md) | 可复制组合和按场景查方法 |

## 共同约定

- 长期复用 `*req.Client`，不要为每次请求重新创建 client。
- client 级设置作为公共默认值；request 级设置用于单次覆盖。
- `err == nil` 只表示请求流程没有返回错误，不表示 HTTP 状态一定成功。
- 超时、取消和重试应一起设计；不要无条件重试有副作用且不可重放的请求。
- 浏览器伪装优先使用一致的 `Impersonate*` profile，而不是只修改 `User-Agent`；OS 选项主要改变 UA/Header，Safari 当前没有专用 H3 profile。
- HTTP/3 面向普通业务时建议启用回退；只有明确需要失败即失败时才强制 HTTP/3。
- 对不可信服务设置 `SetMaxResponseSize`，大响应使用 `DisableAutoReadResponse` 或直接输出到文件/writer。

## 可编译示例

[examples](examples/README.md) 覆盖：

- beginner
- basic
- production-client
- auth-cookie
- upload-download
- middleware
- browser-http3
- tls-fingerprint
- custom-network

示例使用 `httptest.Server` 或只构造配置，不会在测试过程中访问公网。
