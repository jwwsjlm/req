# 静态分析与 v3.62.0 发布检查

检查日期：2026-10-03。工具从 [analysis-tools-dev/static-analysis](https://github.com/analysis-tools-dev/static-analysis#go) 的 Go 分类中选取。项目是 Go HTTP 客户端，因此使用直接分析 Go 代码的工具；没有额外引入聚合 runner 或运行时依赖。

## 工具与范围

| 工具 | 版本 | 用途 |
| --- | --- | --- |
| [Staticcheck](https://staticcheck.dev/) | 2026.1 / v0.7.0 | 潜在错误、未使用代码、无效条件、简化写法、弃用 API |
| [govulncheck](https://go.dev/security/vuln/) | v1.6.0 | 检查已知漏洞及本项目是否导入、调用相关代码 |
| go vet | Go 1.26.8 | Go 工具链内置静态检查，补充格式串、锁复制等诊断 |

主模块与以下 6 个独立模块分别执行 `staticcheck ./...`、`govulncheck ./...` 和 `go test ./... -count=1`：

- `examples/find-popular-repo`
- `examples/opentelemetry-jaeger-tracing`
- `examples/upload/uploadclient`
- `examples/upload/uploadserver`
- `examples/uploadcallback/uploadclient`
- `examples/uploadcallback/uploadserver`

主模块另执行 `go vet ./...` 与 `go test -race ./... -count=1`。本机为 Windows/amd64、Go 1.26.8；还用最低工具链 Go 1.26.7 复查了主模块漏洞扫描。CI 保持 Linux/Windows 测试矩阵，并在 Linux 上运行固定版本的两项静态分析工具。

## 首轮发现与处理

Staticcheck 首轮主模块报告 53 项，独立 tracing 示例另有 1 项。处理结果：

- 删除确认无引用的 HTTP/2 goroutine 调试链、旧拨号包装文件、闲置 buffer pool、HTTP/2 错误包装和空函数、Request 私有字段、HTTP/3 空壳 body 类型、没有接入运行时的 GODEBUG 更新链。删除前同时检查 GitNexus 与全仓文本引用。
- 简化恒假的 int32 上界比较，保留负数验证；直接返回 Header 排除表结果；使用 `time.Since`；移除冗余布尔比较和空 `return`。
- 保留 8 处旧 TLS 字段和 `http.Request.Cancel` 的兼容行为，逐行使用有原因的 `//lint:ignore SA1019`。没有禁用整个检查类别，也没有为消除弃用提示而删掉旧调用方仍可使用的取消行为或更改自定义 TLS 连接的 ALPN 判断。
- 两处测试专门验证非规范化 Header 的大小写，因此用逐行 `SA1008` 例外保留原断言；把这些 key 改为规范化写法会破坏测试目的。
- 用受支持的 OTLP/HTTP exporter 替换弃用的 Jaeger exporter；改用标准 `OTEL_EXPORTER_OTLP_*` 环境变量，并修正旧语义 schema 与 SDK 默认 Resource 合并的问题。新增本地 collector 测试验证实际 protobuf 导出及 `service.name`。

所有模块复检均无剩余 Staticcheck 诊断。这里的“通过”包括上述逐行记录的 10 处兼容性/测试例外，并不意味着项目完全不含旧 API。

## 漏洞扫描结论

7 个模块均未发现可达漏洞，导入的包也未报告漏洞。数据库更新时间为 2026-10-01 20:24:15 UTC。

依赖模块 `golang.org/x/crypto v0.57.0` 存在 [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932)，针对已停维护的 `openpgp` 包，数据库没有修复版本。本项目和示例均未导入该包，govulncheck 判定不受此项影响；仍保留此记录，避免将“没有可达漏洞”写成“依赖图完全没有漏洞记录”。

这些结论受当前源码、平台、工具版本与漏洞数据库范围限制，不代表证明所有潜在缺陷不存在。

## 复现

```sh
go install honnef.co/go/tools/cmd/staticcheck@v0.7.0
go install golang.org/x/vuln/cmd/govulncheck@v1.6.0
staticcheck ./...
govulncheck -show verbose ./...
go vet ./...
go test -race ./... -count=1
```

独立示例须在各自目录执行；根目录的 `./...` 不会跨越嵌套模块边界。CI 的枚举方式见 [.github/workflows/ci.yml](../.github/workflows/ci.yml)。

## 版本与迁移

发布版本为 **v3.62.0**，保留模块路径 `github.com/jwwsjlm/req/v3`。相对上个 tag `v3.61.2`，版本还包括先前已提交的 API 删除：旧包级请求包装、构造别名及部分 Client 配置转发层已经移除。升级前阅读[迁移与兼容](14-migration-compatibility.md)，使用 `C()`、`T()`、`Client.R()` 和 `client.Transport`。

浏览器预设更新为 Chrome 152、Firefox 148、Safari 26.3；uTLS 固定到未打稳定 tag 的 `88ba76ae4ee3` 提交。详细的 Brotli、依赖、上游同步及 Ponytail 审计结果见[近期维护记录](17-maintenance.md)。
