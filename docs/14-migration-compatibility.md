# 迁移与兼容

## 模块路径

本 fork 的导入路径是：

```go
github.com/jwwsjlm/req/v3
```

从 `github.com/imroc/req/v3` 或其他 fork 迁移时，先替换 module path，再编译和测试。不要只依赖 README 中方法名相似就假设行为完全一致。

## Go 版本

工具链和语言版本以目标 tag 的 `go.mod` 为准。CI、本机和发布环境应使用受支持版本；升级 Go 后重新执行普通测试、race 测试和 benchmark，因为标准库 HTTP/TLS 行为及性能可能变化。

## 已移除的兼容 API

- 使用 `C`、`T` 和 `Client.R`，不再提供 `NewClient`、`NewTransport`、`NewRequest` 及包级请求包装函数。
- 使用 `SetSuccessResult`、`SetErrorResult`、`SetCommonErrorResult`、`SuccessResult`、`ErrorResult`、`IsSuccessState` 和 `IsErrorState`。
- Digest 认证统一通过 `Client.SetCommonDigestAuth` 配置。
- `SetCookieJarFactory` 只接受 `func() http.CookieJar`。
- Transport 配置通过 `client.Transport` 完成；相关方法不再额外返回 `*Client`。
- 公共伪 Header 方法仍保留历史拼写 `SetCommonPseudoHeaderOder`。

## 行为强化点

### Query 合并

request 同名 key 覆盖 client 参数，空 request value 可移除对应 client 编码结果；输入 `url.Values` 不会被修改。依赖重复合并而非覆盖的旧代码应改用 `AddQueryParam(s)` 明确表达。

### Header 顺序

排序继续使用标准库 `CanonicalMIMEHeaderKey` 的匹配语义，但实现已从比较期重复处理改为预计算 rank。若代码依赖非法 Header 名的特殊行为，应保留回归测试。

### 响应大小限制

`SetMaxResponseSize` 可在已知 Content-Length 超限时提前关闭 body，也可在流式读取时返回 `ResponseBodyTooLargeError`。新增限制后，原本成功读取的大响应会按设计失败；调用方应处理 `errors.Is(err, req.ErrResponseBodyTooLarge)`。

### multipart

multipart 文件默认流式发送。已知 size/content-type 时可以计算 Content-Length；未知长度 reader 使用 chunked。上传 callback 也会强制 chunked。迁移前确认服务端和中间代理接受该传输方式。

### Hosts 和 DNS

`SetHosts` 是严格 allowlist，未知域名快速失败，且禁止和代理组合。`SetResolver` 只覆盖 H1/H2 dial；需要 HTTP/3 一致解析时使用 `SetDNSResolver`。

### TLS 与 HTTP/3

uTLS 自定义只用于 HTTP/1.1/2。HTTP/3 使用 Go `crypto/tls` 和 QUIC 专用配置。自定义指纹使用 `SetTLSFingerprintSpecFactory`，每次返回非 nil 的新 spec；nil factory/spec 会返回握手错误。

uTLS 路径现在保留标准 `tls.Config` 的显式 SNI、mTLS、验证回调执行与错误语义、session cache、renegotiation、key log 和 ECH 客户端配置，并在 `Client.Clone` 后重新绑定 clone 自己的 TLS config。`MinVersion`、`MaxVersion`、`CipherSuites`、`CurvePreferences`、`NextProtos` 会先转换，但浏览器、随机或自定义 spec 会按自身 ClientHello 形状覆盖它们；需要强约束这些字段时应选用满足策略的 spec 或标准 TLS。uTLS v1.8.2 无法补出标准 `ConnectionState` 的 `CurveID`、`HelloRetryRequest` 与私有 keying-material exporter，依赖这些信息的策略应保留标准 TLS 路径。依赖旧版本中“开启指纹后安全/身份配置被忽略”的代码属于不安全偶然行为，应按新语义修正测试。

原有 `SetTLSFingerprintRandomized` 保持兼容。新代码应根据协议明确选择 `SetTLSFingerprintRandomizedALPN` 或 `SetTLSFingerprintRandomizedNoALPN`；需要跨新连接稳定时使用对应 `WithSeed` API。`ParseTLSClientHello` 只接受完整、未分片的明文 ClientHello record，并默认严格拒绝未知扩展。

Chrome、Firefox、Safari convenience preset 已固定为当前明确版本，避免以后升级 uTLS 时 `_Auto` 静默漂移。浏览器 OS 选项主要改变 UA、Client Hints 和 Header，不代表 TLS ClientHello 随 OS 完全变化。Safari 当前没有专用 HTTP/3 profile。

切换浏览器 profile 会清理未来连接使用的 profile-owned Header/H2/H3 状态，但不会改写已经建立的连接。已开始发送请求的 client 不应原地切换身份；优先新建或 clone 尚未使用的 client。

### Trace

HTTP/3 会转发连接、TLS、连接复用和首字节等核心 trace 事件，但字段完整度与 HTTP/1.1/2 不完全相同。旧逻辑若假定所有协议都有 DNS 分段或把 `TCPConnectTime` 解释为真实 TCP 握手，应为 H3 处理零值和协议差异，并通过回退路径对照验证。

## 标准库直通差异

```go
rawResp, err := client.Do(rawRequest)
```

这个入口复用底层 client/transport，但绕过 req `Request` 构造、自动 body/result 处理和 req middleware。迁移标准库调用时不要误以为 `SetCommonQueryParam` 或 `OnBeforeRequest` 会自动应用。

## 升级清单

1. 阅读目标 tag 的 `go.mod` 和变更说明。
2. 搜索 deprecated API，并按上面的替代方法迁移。
3. 检查 Cookie clone、redirect credential、proxy/DNS 和 H3 回退策略。
4. 对不可重放 body、上传 callback、大响应限制补测试。
5. 运行：

```sh
go mod tidy
go vet ./...
go test ./... -count=1
go test -race ./... -count=1
```

6. 在同环境验证真实 HTTP/TLS 指纹和服务端兼容性，不能用单元测试代替线上协议验证。
