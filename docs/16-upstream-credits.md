# 上游项目、致谢与许可

本仓库是在成熟开源项目之上持续演进的增强版。功能归属、协议边界和许可证应保持清晰，不能把上游能力描述成完全由本仓库独立实现。

## imroc/req

链式 Client/Request/Response、middleware、重试、上传下载、Cookie、HTTP/2/HTTP/3 等核心基础来自 [imroc/req](https://github.com/imroc/req)。本仓库继续沿用根目录 `LICENSE` 中的 MIT License。

## refraction-networking/uTLS

HTTP/1.1 与 HTTP/2 的可定制 ClientHello、浏览器 TLS preset、随机指纹、`UConn`、`ClientHelloSpec` 和捕获 ClientHello 解析能力由 [refraction-networking/uTLS](https://github.com/refraction-networking/utls) 提供。

本仓库在其上完成 req Transport 对接、标准 `crypto/tls.Config` 兼容桥、Header/HTTP/2/HTTP/3 profile 组合和链式 API。HTTP/3 仍由 quic-go 与 Go `crypto/tls` 实现，不是 uTLS QUIC ClientHello。

uTLS 使用 BSD 3-Clause License。本仓库与 uTLS、Refraction Networking、Google 或其贡献者不存在官方隶属或背书关系。

## Go Authors 派生代码

仓库中的部分 transport、HTTP/2、text protocol、SOCKS 和辅助代码保留 Go Authors 源文件头，并继续受对应 BSD-style license 约束。

## 其他设计参考

- [enetx/surf](https://github.com/enetx/surf)：HTTP/3 tuning、浏览器 profile 与 TLS impersonation 设计参考。
- [go-resty/resty](https://github.com/go-resty/resty)：请求构造、multipart field、raw path 参数和易用 API 设计参考。

本页涉及的 Go Authors 派生代码与 uTLS 的版权、再分发条件和免责声明见仓库根目录 [THIRD_PARTY_NOTICES.md](https://github.com/jwwsjlm/req/blob/master/THIRD_PARTY_NOTICES.md)。该文件不是依赖 SBOM；其他依赖继续遵循各自许可证。发布源码、二进制或容器时应保留适用的 notices、根 `LICENSE` 和依赖许可证材料。
