# API 索引

本页按场景索引公开 API。精确签名、注释和当前可用项以 `go doc github.com/jwwsjlm/req/v3` 为准。

## 创建和执行

| 场景 | API |
| --- | --- |
| Client | `C`、`NewClient`、`DefaultClient`、`SetDefaultClient`、`Client.Clone` |
| Request | `Client.R`、`Client.NewRequest`、`R`、`NewRequest` |
| 链式方法 builder | `Client.Get`、`Post`、`Put`、`Patch`、`Delete`、`Head`、`Options` |
| 发送 | `Request.Get`、`Post`、`Put`、`Patch`、`Delete`、`Head`、`Options`、`Query`、`Send`、`Do` |
| Must | `MustGet`、`MustPost`、`MustPut`、`MustPatch`、`MustDelete`、`MustHead`、`MustOptions`、`MustQuery` |
| 标准库直通 | `Client.GetClient`、`Client.Do(*http.Request)` |

## URL、Query 与 Header

| 场景 | API |
| --- | --- |
| BaseURL/Scheme | `SetBaseURL`、`SetScheme`、`SetURL` |
| Path | `SetPathParam`、`SetPathParamAny`、`SetPathParams`、`SetPathRawParam`、`SetPathRawParamAny`、`SetPathRawParams` |
| Query | `SetQueryParam`、`SetQueryParamAny`、`AddQueryParam`、`AddQueryParams`、`SetQueryParams`、`SetQueryParamsAnyType`、`SetQueryParamsFromValues`、`SetQueryParamsFromStruct`、`SetQueryString` |
| Header | `SetHeader`、`SetHeaderAny`、`SetHeaderValues`、`SetHeaderMultiValues`、`SetHeaders`、`SetHeaderNonCanonical`、`SetHeadersNonCanonical` |
| 顺序 | `SetHeaderOrder`、`SetPseudoHeaderOrder`、`SetCommonHeaderOrder`、`SetCommonPseudoHeaderOder` |
| Client 公共项 | 相应的 `SetCommon*` / `AddCommon*` 方法 |

## Body、表单和结果

| 场景 | API |
| --- | --- |
| Body | `SetBody`、`SetBodyBytes`、`SetBodyString`、`SetBodyJsonString`、`SetBodyJsonBytes`、`SetBodyJsonMarshal`、`SetBodyXmlString`、`SetBodyXmlBytes`、`SetBodyXmlMarshal` |
| 元数据 | `SetContentType`、`SetContentLength`、`SetCommonContentType` |
| Form | `SetFormData`、`SetFormDataAny`、`SetFormDataAnyType`、`SetFormDataFromValues`、`SetOrderedFormData` |
| Multipart | `SetMultipartField`、`SetFile`、`SetFiles`、`SetFileBytes`、`SetFileReader`、`SetFileUpload`、`EnableForceMultipart`、`EnableForceChunkedEncoding` |
| 结果 | `SetSuccessResult`、`SetErrorResult`、`SetCommonErrorResult`、`SuccessResult`、`ErrorResult`、`Into`、`Unmarshal`、`UnmarshalJson`、`UnmarshalXml` |
| 状态 | `ResultState`、`IsSuccessState`、`IsErrorState`、`SetResultStateCheckFunc` |

## Response

| 场景 | API |
| --- | --- |
| Body | `String`、`Bytes`、`ToString`、`ToBytes`、`SetBody`、`SetBodyString` |
| 状态/Header | `GetStatus`、`GetStatusCode`、`GetContentType`、`GetHeader`、`GetHeaderValues`、`HeaderToString` |
| 时间 | `TotalTime`、`ReceivedAt`、`TraceInfo` |
| TLS | `TLSInfo`、`TLSGrabber` |
| Dump | `Dump` |

## 认证、Cookie 与重定向

| 场景 | API |
| --- | --- |
| Token | `SetBearerAuthToken`、`SetAuthToken`、`SetAuthSchemeToken` 及 `SetCommon*` 版本 |
| Basic/Digest | `SetBasicAuth`、`SetDigestAuth` 及 `SetCommon*` 版本 |
| Cookie | `SetCookies`、`SetCommonCookies`、`SetCookieJar`、`SetCookieJarFactory`、`GetCookies`、`ClearCookies` |
| Redirect | `SetRedirectPolicy`、`DefaultRedirectPolicy`、`MaxRedirectPolicy`、`NoRedirectPolicy`、`SameHostRedirectPolicy`、`SameDomainRedirectPolicy`、`AllowedHostRedirectPolicy`、`AllowedDomainRedirectPolicy` |
| Redirect Header | `AlwaysCopyHeaderRedirectPolicy`、`SensitiveHeadersRedirectPolicy` |

## 超时、重试和响应限制

| 场景 | API |
| --- | --- |
| 总超时/Context | `SetTimeout`、`SetContext`、`Context`、`Do(ctx)`、`SetContextData`、`GetContextData` |
| Retry count | `SetRetryCount`、`SetCommonRetryCount`、`GetRetryOption` |
| Retry interval | `SetRetryInterval`、`SetRetryFixedInterval`、`SetRetryBackoffInterval` 及 `SetCommon*` 版本 |
| Retry condition/hook | `SetRetryCondition`、`AddRetryCondition`、`SetRetryHook`、`AddRetryHook` 及 `SetCommon*` 版本 |
| 响应限制 | `SetMaxResponseSize`、`ErrResponseBodyTooLarge`、`ResponseBodyTooLargeError` |
| 读取模式 | `DisableAutoReadResponse`、`EnableAutoReadResponse`、`EnableCloseConnection` |

## 上传与下载

| 场景 | API |
| --- | --- |
| 上传回调 | `SetUploadCallback`、`SetUploadCallbackWithInterval`、`UploadInfo` |
| 下载目标 | `SetOutputFile`、`SetOutput`、`SetOutputDirectory` |
| 下载回调 | `SetDownloadCallback`、`SetDownloadCallbackWithInterval`、`DownloadInfo` |
| 并行下载 | `NewParallelDownload`、`ParallelDownload.SetConcurrency`、`SetSegmentSize`、`SetTempRootDir`、`SetFileMode`、`SetOutput`、`SetOutputFile`、`Do` |

## Middleware、Dump 与 Trace

| 场景 | API |
| --- | --- |
| Middleware | `OnBeforeRequest`、`OnAfterResponse`、`Request.OnAfterResponse`、`OnError` |
| req round trip | `WrapRoundTrip`、`WrapRoundTripFunc`、`RoundTripper`、`RoundTripFunc` |
| http round trip | `Transport.WrapRoundTrip`、`Transport.WrapRoundTripFunc`、`HttpRoundTripFunc` |
| Trace | `EnableTrace`、`DisableTrace`、`EnableTraceAll`、`DisableTraceAll`、`TraceInfo`、`TraceInfo.Blame` |
| Dump | `EnableDump`、`EnableDumpTo`、`EnableDumpToFile`、`EnableDumpAll*`、`EnableDumpEachRequest*`、`SetDumpOptions`、`SetCommonDumpOptions` |
| Logger | `SetLogger`、`GetLogger`、`NewLogger`、`NewLoggerFromStandardLogger` |

## 代理、DNS 与 Transport

| 场景 | API |
| --- | --- |
| Proxy | `SetProxyURL`、`SetProxy`、`Transport.SetProxyConnectHeader`、`Transport.SetGetProxyConnectHeader` |
| DNS | `SetDNSResolver`、`SetResolver`、`SetHosts`、`SetDNSOverTLS`、`SetDNSOverTLSCloudflare`、`SetDNSOverTLSGoogle`、`SetDNSOverTLSQuad9`、`SetDNSOverTLSAdGuard`、`SetDNSOverTLSAli`、`NewDNSOverTLSResolver` |
| Dial | `SetDial`、`SetDialTLS`、`SetTLSHandshake`、`SetUnixSocket` |
| 连接池 | `SetMaxIdleConns`、`GetMaxIdleConns`、`SetMaxConnsPerHost`、`SetIdleConnTimeout`、`EnableKeepAlives`、`DisableKeepAlives`、`CloseIdleConnections` |
| 限制/缓冲 | `SetResponseHeaderTimeout`、`SetTLSHandshakeTimeout`、`SetExpectContinueTimeout`、`SetMaxResponseHeaderBytes`、`SetReadBufferSize`、`SetWriteBufferSize` |

## 浏览器、TLS 与协议

| 场景 | API |
| --- | --- |
| 浏览器 | `ImpersonateChrome`、`ImpersonateChromeWithOS`、`ImpersonateChromeRandomOS`、Firefox 对应方法、`ImpersonateSafari`、`RandomBrowserOS` |
| TLS 指纹 | `SetTLSFingerprint`、`SetTLSFingerprintJA3`、`SetTLSFingerprintSpec`、`SetTLSFingerprintSpecFactory`、各内置 `SetTLSFingerprint*` |
| 证书/TLS | `SetTLSClientConfig`、`GetTLSClientConfig`、`SetRootCertFromString`、`SetRootCertsFromFile`、`SetCertFromFile`、`SetCerts`、`EnableInsecureSkipVerify` |
| HTTP 版本 | `EnableForceHTTP1`、`EnableForceHTTP2`、`EnableForceHTTP3`、`DisableForceHttpVersion`、`EnableH2C`、`EnableHTTP3`、`DisableHTTP3` |
| HTTP/2 | `SetHTTP2SettingsFrame`、`SetHTTP2ConnectionFlow`、`SetHTTP2InitialStreamID`、`SetHTTP2HeaderPriority`、`SetHTTP2PriorityFrames`、HTTP/2 timeout/limit 方法 |
| HTTP/3 | `SetHTTP3AdditionalSettings`、`SetHTTP3AdditionalSetting`、`SetHTTP3Grease`、Datagram/Extended CONNECT、`SetHTTP3MaxResponseHeaderBytes`、QUIC/TLS profile、fallback/cooldown |

## 压缩、解码与转换

| 场景 | API |
| --- | --- |
| 压缩 | `EnableCompression`、`DisableCompression` |
| 字符集 | `EnableAutoDecode`、`DisableAutoDecode`、`SetAutoDecodeContentType`、`SetAutoDecodeAllContentType`、`SetAutoDecodeContentTypeFunc` |
| Body 转换 | `SetResponseBodyTransformer` |
| 序列化器 | `SetJsonMarshal`、`SetJsonUnmarshal`、`SetXmlMarshal`、`SetXmlUnmarshal` |
