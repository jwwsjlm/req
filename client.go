package req

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	urlpkg "net/url"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/publicsuffix"

	"github.com/jwwsjlm/req/v3/internal/header"
	"github.com/jwwsjlm/req/v3/internal/util"

	"github.com/google/go-querystring/query"
)

// DefaultClient returns the global default Client.
//
// DefaultClient 返回包级默认 Client，供包级请求辅助函数复用。
func DefaultClient() *Client {
	return defaultClient
}

// SetDefaultClient overrides the global default Client.
//
// SetDefaultClient 将 c 设为包级默认 Client；后续包级请求辅助函数都使用该实例。
func SetDefaultClient(c *Client) {
	if c != nil {
		defaultClient = c
	}
}

var defaultClient = C()

// Client is the req's http client.
//
// Client 表示 req 的 HTTP 客户端，并保存可被 Request 继承的默认配置。
type Client struct {
	BaseURL               string
	PathParams            map[string]string
	RawPathParams         map[string]string
	QueryParams           urlpkg.Values
	FormData              urlpkg.Values
	DebugLog              bool
	AllowGetMethodPayload bool
	*Transport
	digestAuth              *digestAuth
	cookiejarFactory        func() http.CookieJar
	trace                   bool
	disableAutoReadResponse bool
	maxResponseSize         int64 // 0 means no limit
	commonErrorType         reflect.Type
	retryOption             *RetryOption
	jsonMarshal             func(v any) ([]byte, error)
	jsonUnmarshal           func(data []byte, v any) error
	xmlMarshal              func(v any) ([]byte, error)
	xmlUnmarshal            func(data []byte, v any) error
	multipartBoundaryFunc   func() string
	outputDirectory         string
	scheme                  string
	log                     Logger
	dumpOptions             *DumpOptions
	dumpOutputCloser        io.Closer
	httpClient              *http.Client
	beforeRequest           []RequestMiddleware
	udBeforeRequest         []RequestMiddleware
	afterResponse           []ResponseMiddleware
	wrappedRoundTrip        RoundTripper
	roundTripWrappers       []RoundTripWrapper
	responseBodyTransformer func(rawBody []byte, req *Request, resp *Response) (transformedBody []byte, err error)
	resultStateCheckFunc    func(resp *Response) ResultState
	onError                 ErrorHook
	browserProfile          *browserHeaderProfile
}

// ErrorHook is called when a Client request finishes with an error.
//
// ErrorHook 定义在客户端请求处理失败时调用的回调函数。
type ErrorHook func(client *Client, req *Request, resp *Response, err error)

// R creates a new request.
//
// R 创建关联到此 Client 的新 Request，并复制当前默认重试选项。
func (c *Client) R() *Request {
	return &Request{
		client:      c,
		retryOption: c.retryOption.Clone(),
	}
}

// Get create a new GET request, accepts 0 or 1 url.
//
// Get 创建 GET Request；url 可省略或只提供一个，省略时由后续配置补充目标地址。
func (c *Client) Get(url ...string) *Request {
	r := c.R()
	if len(url) > 0 {
		r.RawURL = url[0]
	}
	r.Method = http.MethodGet
	return r
}

// Post creates a new POST request.
//
// Post 创建 POST Request；url 可省略或只提供一个，提供时将其设为原始请求 URL。
func (c *Client) Post(url ...string) *Request {
	r := c.R()
	if len(url) > 0 {
		r.RawURL = url[0]
	}
	r.Method = http.MethodPost
	return r
}

// Patch creates a new PATCH request.
//
// Patch 创建 PATCH Request；url 可省略或只提供一个，提供时将其设为原始请求 URL。
func (c *Client) Patch(url ...string) *Request {
	r := c.R()
	if len(url) > 0 {
		r.RawURL = url[0]
	}
	r.Method = http.MethodPatch
	return r
}

// Delete creates a new DELETE request.
//
// Delete 创建 DELETE Request；url 可省略或只提供一个，提供时将其设为原始请求 URL。
func (c *Client) Delete(url ...string) *Request {
	r := c.R()
	if len(url) > 0 {
		r.RawURL = url[0]
	}
	r.Method = http.MethodDelete
	return r
}

// Put creates a new PUT request.
//
// Put 创建 PUT Request；url 可省略或只提供一个，提供时将其设为原始请求 URL。
func (c *Client) Put(url ...string) *Request {
	r := c.R()
	if len(url) > 0 {
		r.RawURL = url[0]
	}
	r.Method = http.MethodPut
	return r
}

// Head creates a new HEAD request.
//
// Head 创建 HEAD Request；url 可省略或只提供一个，提供时将其设为原始请求 URL。
func (c *Client) Head(url ...string) *Request {
	r := c.R()
	if len(url) > 0 {
		r.RawURL = url[0]
	}
	r.Method = http.MethodHead
	return r
}

// Options creates a new OPTIONS request.
//
// Options 创建 OPTIONS Request；url 可省略或只提供一个，提供时将其设为原始请求 URL。
func (c *Client) Options(url ...string) *Request {
	r := c.R()
	if len(url) > 0 {
		r.RawURL = url[0]
	}
	r.Method = http.MethodOptions
	return r
}

// GetTransport returns the underlying transport.
//
// GetTransport 返回此 Client 使用的底层 Transport。
func (c *Client) GetTransport() *Transport {
	return c.Transport
}

// SetResponseBodyTransformer sets a response body transformer. When automatic response
// body reading is enabled, it runs before the body is unmarshaled.
//
// SetResponseBodyTransformer 设置响应正文转换函数；未禁用自动读取时，转换在响应体自动反序列化之前执行。
func (c *Client) SetResponseBodyTransformer(fn func(rawBody []byte, req *Request, resp *Response) (transformedBody []byte, err error)) *Client {
	c.responseBodyTransformer = fn
	return c
}

// SetCommonErrorResult sets the default result into which a response body is unmarshaled
// when no request error occurs but Response.ResultState is ErrorState. By default,
// status codes greater than or equal to 400 are errors; use SetResultStateCheckFunc
// to customize the result-state logic.
//
// SetCommonErrorResult 设置默认错误结果接收对象；请求无 Go error 但结果状态为 ErrorState 时将响应体反序列化到该对象。
func (c *Client) SetCommonErrorResult(err any) *Client {
	if err != nil {
		c.commonErrorType = util.GetType(err)
	}
	return c
}

// ResultState represents the state of the result.
//
// ResultState 表示响应处理后的结果状态。
type ResultState int

const (
	// SuccessState indicates the response is in success state,
	// and result will be unmarshalled if Request.SetSuccessResult
	// is called.
	SuccessState ResultState = iota
	// ErrorState indicates the response is in error state,
	// and result will be unmarshalled if Request.SetErrorResult
	// or Client.SetCommonErrorResult is called.
	ErrorState
	// UnknownState indicates the response is in unknown state,
	// and handler will be invoked if Request.SetUnknownResultHandlerFunc
	// or Client.SetCommonUnknownResultHandlerFunc is called.
	UnknownState
)

// SetResultStateCheckFunc overrides the default result state checker with customized one,
// which returns SuccessState when HTTP status `code >= 200 and <= 299`, and returns
// ErrorState when HTTP status `code >= 400`, otherwise returns UnknownState.
//
// SetResultStateCheckFunc 设置结果状态判定函数，替换默认的 2xx 为成功、4xx 及以上为错误、其余为未知的规则。
func (c *Client) SetResultStateCheckFunc(fn func(resp *Response) ResultState) *Client {
	c.resultStateCheckFunc = fn
	return c
}

// SetCommonFormDataFromValues sets the form data from url.Values for requests
// fired from the client which request method allows payload.
//
// SetCommonFormDataFromValues 为此 Client 创建的允许携带请求体的 Request 设置默认表单字段，字段来自 url.Values。
func (c *Client) SetCommonFormDataFromValues(data urlpkg.Values) *Client {
	if c.FormData == nil {
		c.FormData = urlpkg.Values{}
	}
	for k, v := range data {
		for _, kv := range v {
			c.FormData.Add(k, kv)
		}
	}
	return c
}

// SetCommonFormData sets the form data from map for requests fired from the client
// which request method allows payload.
//
// SetCommonFormData 为此 Client 创建的允许携带请求体的 Request 设置默认字符串表单字段。
func (c *Client) SetCommonFormData(data map[string]string) *Client {
	if c.FormData == nil {
		c.FormData = urlpkg.Values{}
	}
	for k, v := range data {
		c.FormData.Set(k, v)
	}
	return c
}

// SetCommonFormDataAnyType sets form data from a map whose values can be any type.
//
// SetCommonFormDataAnyType 为此 Client 创建的允许携带请求体的 Request 设置默认表单字段，并用 fmt.Sprint 转换任意类型的值。
func (c *Client) SetCommonFormDataAnyType(data map[string]any) *Client {
	if c.FormData == nil {
		c.FormData = urlpkg.Values{}
	}
	for k, v := range data {
		c.FormData.Set(k, fmt.Sprint(v))
	}
	return c
}

// SetMultipartBoundaryFunc overrides the default function used to generate
// boundary delimiters for "multipart/form-data" requests with a customized one,
// which returns a boundary delimiter (without the two leading hyphens).
//
// Boundary delimiter may only contain certain ASCII characters, and must be
// non-empty and at most 70 bytes long (see RFC 2046, Section 5.1.1).
//
// SetMultipartBoundaryFunc 设置 multipart/form-data 边界生成函数；返回值不得为空、最长 70 字节，且只能含 RFC 2046 允许的 ASCII 字符。
func (c *Client) SetMultipartBoundaryFunc(fn func() string) *Client {
	c.multipartBoundaryFunc = fn
	return c
}

// SetBaseURL sets the default base URL, will be used if request URL is
// a relative URL.
//
// SetBaseURL 设置默认基础 URL；Request 使用相对 URL 时以它为基准解析。
func (c *Client) SetBaseURL(u string) *Client {
	c.BaseURL = strings.TrimRight(u, "/")
	return c
}

// SetOutputDirectory sets output directory that response will
// be downloaded to.
//
// SetOutputDirectory 设置下载响应正文时使用的默认输出目录。
func (c *Client) SetOutputDirectory(dir string) *Client {
	c.outputDirectory = dir
	return c
}

// SetCertFromFile helps to set client certificates from cert and key file.
//
// SetCertFromFile 从证书文件和私钥文件加载客户端 TLS 证书。
func (c *Client) SetCertFromFile(certFile, keyFile string) *Client {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		c.log.Errorf("failed to load client cert: %v", err)
		return c
	}
	config := c.GetTLSClientConfig()
	config.Certificates = append(config.Certificates, cert)
	return c
}

// SetCerts sets client certificates.
//
// SetCerts 设置 Client 在 TLS 握手中使用的客户端证书。
func (c *Client) SetCerts(certs ...tls.Certificate) *Client {
	config := c.GetTLSClientConfig()
	config.Certificates = append(config.Certificates, certs...)
	return c
}

func (c *Client) appendRootCertData(data []byte) {
	config := c.GetTLSClientConfig()
	if config.RootCAs == nil {
		config.RootCAs = x509.NewCertPool()
	}
	config.RootCAs.AppendCertsFromPEM(data)
}

// SetRootCertFromString sets root certificates from string.
//
// SetRootCertFromString 从 PEM 字符串加载并添加受信任的根证书。
func (c *Client) SetRootCertFromString(pemContent string) *Client {
	c.appendRootCertData([]byte(pemContent))
	return c
}

// SetRootCertsFromFile sets root certificates from files.
//
// SetRootCertsFromFile 从一个或多个 PEM 文件加载并添加受信任的根证书。
func (c *Client) SetRootCertsFromFile(pemFiles ...string) *Client {
	for _, pemFile := range pemFiles {
		rootPemData, err := os.ReadFile(pemFile)
		if err != nil {
			c.log.Errorf("failed to read root cert file: %v", err)
			return c
		}
		c.appendRootCertData(rootPemData)
	}
	return c
}

// GetTLSClientConfig returns the underlying tls.Config.
//
// GetTLSClientConfig 返回此 Client 当前使用的底层 tls.Config。
func (c *Client) GetTLSClientConfig() *tls.Config {
	if c.TLSClientConfig == nil {
		c.Transport.SetTLSClientConfig(&tls.Config{
			NextProtos: []string{"h2", "http/1.1"},
		})
	}
	return c.TLSClientConfig
}

// SetRedirectPolicy sets the RedirectPolicy which controls the behavior of receiving redirect
// responses (usually responses with 301 and 302 status code), see the predefined
// AllowedDomainRedirectPolicy, AllowedHostRedirectPolicy, DefaultRedirectPolicy, MaxRedirectPolicy,
// NoRedirectPolicy, SameDomainRedirectPolicy and SameHostRedirectPolicy.
//
// SetRedirectPolicy 设置处理重定向响应的策略；可组合域名、主机、最大次数或禁止重定向等 RedirectPolicy。
func (c *Client) SetRedirectPolicy(policies ...RedirectPolicy) *Client {
	if len(policies) == 0 {
		return c
	}
	c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		for _, f := range policies {
			if f == nil {
				continue
			}
			err := f(req, via)
			if err != nil {
				return err
			}
		}
		if c.DebugLog {
			c.log.Debugf("<redirect> %s %s", req.Method, req.URL.String())
		}
		return nil
	}
	return c
}

// EnableInsecureSkipVerify enables send https without verifying
// the server's certificates (disabled by default).
//
// EnableInsecureSkipVerify 启用跳过 HTTPS 服务端证书校验；默认关闭，仅应在受控测试环境使用。
func (c *Client) EnableInsecureSkipVerify() *Client {
	c.GetTLSClientConfig().InsecureSkipVerify = true
	return c
}

// DisableInsecureSkipVerify disables send https without verifying
// the server's certificates (disabled by default).
//
// DisableInsecureSkipVerify 关闭跳过 HTTPS 服务端证书校验，恢复正常证书验证。
func (c *Client) DisableInsecureSkipVerify() *Client {
	c.GetTLSClientConfig().InsecureSkipVerify = false
	return c
}

// SetCommonQueryParams sets URL query parameters with a map
// for requests fired from the client.
//
// SetCommonQueryParams 为此 Client 创建的 Request 设置默认 URL Query 参数；每个键只保留 map 中给出的值。
func (c *Client) SetCommonQueryParams(params map[string]string) *Client {
	for k, v := range params {
		c.SetCommonQueryParam(k, v)
	}
	return c
}

// AddCommonQueryParam adds a URL query parameter with a key-value
// pair for requests fired from the client.
//
// AddCommonQueryParam 为此 Client 创建的 Request 追加一个默认 URL Query 键值对，不清除同名已有值。
func (c *Client) AddCommonQueryParam(key, value string) *Client {
	if c.QueryParams == nil {
		c.QueryParams = make(urlpkg.Values)
	}
	c.QueryParams.Add(key, value)
	return c
}

// AddCommonQueryParams adds one or more values of specified URL query parameter
// for requests fired from the client.
//
// AddCommonQueryParams 为此 Client 创建的 Request 为指定 Query 键追加一个或多个默认值。
func (c *Client) AddCommonQueryParams(key string, values ...string) *Client {
	if c.QueryParams == nil {
		c.QueryParams = make(urlpkg.Values)
	}
	vs := c.QueryParams[key]
	vs = append(vs, values...)
	c.QueryParams[key] = vs
	return c
}

// SetCommonQueryParamAny sets a query parameter value converted from any type with fmt.Sprint.
//
// SetCommonQueryParamAny 为此 Client 创建的 Request 设置一个默认 Query 参数，并用 fmt.Sprint 转换 value。
func (c *Client) SetCommonQueryParamAny(key string, value any) *Client {
	return c.SetCommonQueryParam(key, fmt.Sprint(value))
}

func (c *Client) pathParams() map[string]string {
	if c.PathParams == nil {
		c.PathParams = make(map[string]string)
	}
	return c.PathParams
}

// SetCommonPathParam sets a path parameter for requests fired from the client.
//
// SetCommonPathParam 为此 Client 创建的 Request 设置默认路径参数；普通路径参数会进行 URL 路径转义。
func (c *Client) SetCommonPathParam(key, value string) *Client {
	c.pathParams()[key] = value
	return c
}

// SetCommonPathParamAny sets a path parameter value converted from any type with fmt.Sprint.
//
// SetCommonPathParamAny 为此 Client 创建的 Request 设置默认路径参数，并用 fmt.Sprint 转换 value 后进行路径转义。
func (c *Client) SetCommonPathParamAny(key string, value any) *Client {
	return c.SetCommonPathParam(key, fmt.Sprint(value))
}

// SetCommonPathParams sets path parameters for requests fired from the client.
//
// SetCommonPathParams 为此 Client 创建的 Request 批量设置默认路径参数；值会进行 URL 路径转义。
func (c *Client) SetCommonPathParams(pathParams map[string]string) *Client {
	m := c.pathParams()
	for k, v := range pathParams {
		m[k] = v
	}
	return c
}

func (c *Client) rawPathParams() map[string]string {
	if c.RawPathParams == nil {
		c.RawPathParams = make(map[string]string)
	}
	return c.RawPathParams
}

// SetCommonPathRawParam sets a path parameter without url.PathEscape.
//
// SetCommonPathRawParam 为此 Client 创建的 Request 设置未经 url.PathEscape 处理的默认路径参数。
func (c *Client) SetCommonPathRawParam(key, value string) *Client {
	c.rawPathParams()[key] = value
	return c
}

// SetCommonPathRawParamAny sets a raw path parameter value converted from any type with fmt.Sprint.
//
// SetCommonPathRawParamAny 为此 Client 创建的 Request 设置未经路径转义的默认路径参数，并用 fmt.Sprint 转换 value。
func (c *Client) SetCommonPathRawParamAny(key string, value any) *Client {
	return c.SetCommonPathRawParam(key, fmt.Sprint(value))
}

// SetCommonPathRawParams sets multiple path parameters without url.PathEscape.
//
// SetCommonPathRawParams 为此 Client 创建的 Request 批量设置未经 url.PathEscape 处理的默认路径参数。
func (c *Client) SetCommonPathRawParams(pathParams map[string]string) *Client {
	m := c.rawPathParams()
	for k, v := range pathParams {
		m[k] = v
	}
	return c
}

// SetCommonQueryParam sets a URL query parameter with a key-value
// pair for requests fired from the client.
//
// SetCommonQueryParam 为此 Client 创建的 Request 设置一个默认 URL Query 键值对；同名默认值会被覆盖。
func (c *Client) SetCommonQueryParam(key, value string) *Client {
	if c.QueryParams == nil {
		c.QueryParams = make(urlpkg.Values)
	}
	c.QueryParams.Set(key, value)
	return c
}

// SetCommonQueryString sets URL query parameters with a raw query string
// for requests fired from the client.
//
// SetCommonQueryString 为此 Client 创建的 Request 设置由原始 query string 解析出的默认 URL Query 参数。
func (c *Client) SetCommonQueryString(query string) *Client {
	params, err := urlpkg.ParseQuery(strings.TrimSpace(query))
	if err != nil {
		c.log.Warnf("failed to parse query string (%s): %v", query, err)
		return c
	}
	if c.QueryParams == nil {
		c.QueryParams = make(urlpkg.Values)
	}
	for p, v := range params {
		for _, pv := range v {
			c.QueryParams.Add(p, pv)
		}
	}
	return c
}

// SetCommonQueryParamsFromValues sets URL query parameters from a url.Values map
// for requests fired from the client.
//
// SetCommonQueryParamsFromValues 为此 Client 创建的 Request 从 url.Values 批量设置默认 URL Query 参数。
func (c *Client) SetCommonQueryParamsFromValues(params urlpkg.Values) *Client {
	if c.QueryParams == nil {
		c.QueryParams = make(urlpkg.Values)
	}
	for p, v := range params {
		for _, pv := range v {
			c.QueryParams.Add(p, pv)
		}
	}
	return c
}

// SetCommonQueryParamsFromStruct sets URL query parameters from a struct using go-querystring
// for requests fired from the client.
//
// SetCommonQueryParamsFromStruct 为此 Client 创建的 Request 从 struct 生成默认 URL Query 参数，编码规则由 go-querystring 决定。
func (c *Client) SetCommonQueryParamsFromStruct(v any) *Client {
	values, err := query.Values(v)
	if err != nil {
		c.log.Warnf("failed to convert struct to query parameters: %v", err)
		return c
	}
	return c.SetCommonQueryParamsFromValues(values)
}

// SetCommonCookies sets HTTP cookies for requests fired from the client.
//
// SetCommonCookies 为此 Client 创建的 Request 设置默认 HTTP Cookie。
func (c *Client) SetCommonCookies(cookies ...*http.Cookie) *Client {
	c.Cookies = append(c.Cookies, cookies...)
	return c
}

// DisableDebugLog disables debug level log (disabled by default).
//
// DisableDebugLog 关闭调试级别日志；默认即为关闭。
func (c *Client) DisableDebugLog() *Client {
	c.DebugLog = false
	return c
}

// EnableDebugLog enables debug level log (disabled by default).
//
// EnableDebugLog 启用调试级别日志；默认关闭。
func (c *Client) EnableDebugLog() *Client {
	c.DebugLog = true
	return c
}

// DevMode enables:
// 1. Dump content of all requests and responses to see details.
// 2. Output debug level log for deeper insights.
// 3. Trace all requests, so you can get trace info to analyze performance.
//
// DevMode 同时启用完整请求/响应 dump、调试级别日志和所有 Request 的 trace。
func (c *Client) DevMode() *Client {
	return c.EnableDumpAll().
		EnableDebugLog().
		EnableTraceAll()
}

// SetScheme sets the default scheme for client, will be used when
// there is no scheme in the request URL (e.g. "github.com/imroc/req").
//
// SetScheme 设置缺少 scheme 的 Request URL 使用的默认 scheme，例如为 github.com/imroc/req 补充 https。
func (c *Client) SetScheme(scheme string) *Client {
	if !util.IsStringEmpty(scheme) {
		c.scheme = strings.TrimSpace(scheme)
	}
	return c
}

// GetLogger returns the internal logger, usually used in middleware.
//
// GetLogger 返回此 Client 的内部 Logger，通常供中间件记录日志。
func (c *Client) GetLogger() Logger {
	if c.log != nil {
		return c.log
	}
	c.log = createDefaultLogger()
	return c.log
}

// SetLogger sets the customized logger for client, will disable log if set to nil.
//
// SetLogger 设置此 Client 的 Logger；传入 nil 会关闭日志输出。
func (c *Client) SetLogger(log Logger) *Client {
	if log == nil {
		c.log = &disableLogger{}
		return c
	}
	c.log = log
	return c
}

// SetTimeout sets timeout for requests fired from the client.
//
// SetTimeout 设置此 Client 发起请求的默认超时时间。
func (c *Client) SetTimeout(d time.Duration) *Client {
	c.httpClient.Timeout = d
	return c
}

func (c *Client) getDumpOptions() *DumpOptions {
	if c.dumpOptions == nil {
		c.dumpOptions = newDefaultDumpOptions()
	}
	return c.dumpOptions
}

// EnableDumpAll enables dump for requests fired from the client, including
// all content for the request and response by default.
//
// EnableDumpAll 为此 Client 的请求启用完整 dump，默认包含请求和响应的全部内容。
func (c *Client) EnableDumpAll() *Client {
	if c.Dump != nil { // dump already started
		return c
	}
	c.EnableDump(c.getDumpOptions())
	return c
}

// EnableDumpAllToFile enables dump for requests fired from the
// client and output to the specified file.
//
// EnableDumpAllToFile 为此 Client 的请求启用完整 dump，并写入指定文件。
func (c *Client) EnableDumpAllToFile(filename string) *Client {
	if c.Dump != nil {
		c.DisableDump()
	} else {
		c.closeDumpOutput()
	}
	file, err := os.Create(filename)
	if err != nil {
		c.log.Errorf("create dump file error: %v", err)
		return c
	}
	c.getDumpOptions().Output = file
	c.dumpOutputCloser = file
	c.EnableDumpAll()
	return c
}

// EnableDumpAllTo enables dump for requests fired from the
// client and output to the specified io.Writer.
//
// EnableDumpAllTo 为此 Client 的请求启用完整 dump，并写入指定 io.Writer。
func (c *Client) EnableDumpAllTo(output io.Writer) *Client {
	c.closeDumpOutput()
	c.getDumpOptions().Output = output
	c.EnableDumpAll()
	return c
}

// EnableDumpAllAsync enables dump for requests fired from the
// client and output asynchronously, can be used for debugging
// in production environment without affecting performance.
//
// EnableDumpAllAsync 异步输出此 Client 请求的完整 dump，适合降低调试输出对请求路径的阻塞。
func (c *Client) EnableDumpAllAsync() *Client {
	o := c.getDumpOptions()
	o.Async = true
	c.EnableDumpAll()
	return c
}

// EnableDumpAllWithoutRequestBody enables dump for requests fired
// from the client without request body, can be used in the upload
// request to avoid dumping the unreadable binary content.
//
// EnableDumpAllWithoutRequestBody 启用完整 dump 但省略请求正文，适合上传二进制内容时避免记录不可读或敏感数据。
func (c *Client) EnableDumpAllWithoutRequestBody() *Client {
	o := c.getDumpOptions()
	o.RequestBody = false
	c.EnableDumpAll()
	return c
}

// EnableDumpAllWithoutResponseBody enables dump for requests fired
// from the client without response body, can be used in the download
// request to avoid dumping the unreadable binary content.
//
// EnableDumpAllWithoutResponseBody 启用完整 dump 但省略响应正文，适合下载二进制内容时避免记录不可读或敏感数据。
func (c *Client) EnableDumpAllWithoutResponseBody() *Client {
	o := c.getDumpOptions()
	o.ResponseBody = false
	c.EnableDumpAll()
	return c
}

// EnableDumpAllWithoutResponse enables dump for requests fired from
// the client without response, can be used if you only care about
// the request.
//
// EnableDumpAllWithoutResponse 启用 dump 但不记录响应，仅记录请求。
func (c *Client) EnableDumpAllWithoutResponse() *Client {
	o := c.getDumpOptions()
	o.ResponseBody = false
	o.ResponseHeader = false
	c.EnableDumpAll()
	return c
}

// EnableDumpAllWithoutRequest enables dump for requests fired from
// the client without request, can be used if you only care about
// the response.
//
// EnableDumpAllWithoutRequest 启用 dump 但不记录请求，仅记录响应。
func (c *Client) EnableDumpAllWithoutRequest() *Client {
	o := c.getDumpOptions()
	o.RequestHeader = false
	o.RequestBody = false
	c.EnableDumpAll()
	return c
}

// EnableDumpAllWithoutHeader enables dump for requests fired from
// the client without header, can be used if you only care about
// the body.
//
// EnableDumpAllWithoutHeader 启用 dump 但省略请求和响应 Header。
func (c *Client) EnableDumpAllWithoutHeader() *Client {
	o := c.getDumpOptions()
	o.RequestHeader = false
	o.ResponseHeader = false
	c.EnableDumpAll()
	return c
}

// EnableDumpAllWithoutBody enables dump for requests fired from
// the client without body, can be used if you only care about
// the header.
//
// EnableDumpAllWithoutBody 启用 dump 但省略请求和响应正文。
func (c *Client) EnableDumpAllWithoutBody() *Client {
	o := c.getDumpOptions()
	o.RequestBody = false
	o.ResponseBody = false
	c.EnableDumpAll()
	return c
}

// EnableDumpEachRequest enables dump at the request-level for each request, and only
// temporarily stores the dump content in memory, call Response.Dump() to get the
// dump content when needed.
//
// EnableDumpEachRequest 为每个 Request 启用 dump；默认包含该请求及其响应的全部内容。
func (c *Client) EnableDumpEachRequest() *Client {
	return c.OnBeforeRequest(func(client *Client, req *Request) error {
		if req.RetryAttempt == 0 { // Ignore on retry, no need to repeat enable dump.
			req.EnableDump()
		}
		return nil
	})
}

// EnableDumpEachRequestWithoutBody enables dump without body at the request-level for
// each request, and only temporarily stores the dump content in memory, call
// Response.Dump() to get the dump content when needed.
//
// EnableDumpEachRequestWithoutBody 为每个 Request 启用 dump，但省略请求和响应正文。
func (c *Client) EnableDumpEachRequestWithoutBody() *Client {
	return c.OnBeforeRequest(func(client *Client, req *Request) error {
		if req.RetryAttempt == 0 { // Ignore on retry, no need to repeat enable dump.
			req.EnableDumpWithoutBody()
		}
		return nil
	})
}

// EnableDumpEachRequestWithoutHeader enables dump without header at the request-level for
// each request, and only temporarily stores the dump content in memory, call
// Response.Dump() to get the dump content when needed.
//
// EnableDumpEachRequestWithoutHeader 为每个 Request 启用 dump，但省略请求和响应 Header。
func (c *Client) EnableDumpEachRequestWithoutHeader() *Client {
	return c.OnBeforeRequest(func(client *Client, req *Request) error {
		if req.RetryAttempt == 0 { // Ignore on retry, no need to repeat enable dump.
			req.EnableDumpWithoutHeader()
		}
		return nil
	})
}

// EnableDumpEachRequestWithoutRequest enables dump without request at the request-level for
// each request, and only temporarily stores the dump content in memory, call
// Response.Dump() to get the dump content when needed.
//
// EnableDumpEachRequestWithoutRequest 为每个 Request 启用 dump，但不记录请求部分。
func (c *Client) EnableDumpEachRequestWithoutRequest() *Client {
	return c.OnBeforeRequest(func(client *Client, req *Request) error {
		if req.RetryAttempt == 0 { // Ignore on retry, no need to repeat enable dump.
			req.EnableDumpWithoutRequest()
		}
		return nil
	})
}

// EnableDumpEachRequestWithoutResponse enables dump without response at the request-level for
// each request, and only temporarily stores the dump content in memory, call
// Response.Dump() to get the dump content when needed.
//
// EnableDumpEachRequestWithoutResponse 为每个 Request 启用 dump，但不记录响应部分。
func (c *Client) EnableDumpEachRequestWithoutResponse() *Client {
	return c.OnBeforeRequest(func(client *Client, req *Request) error {
		if req.RetryAttempt == 0 { // Ignore on retry, no need to repeat enable dump.
			req.EnableDumpWithoutResponse()
		}
		return nil
	})
}

// EnableDumpEachRequestWithoutResponseBody enables dump without response body at the
// request-level for each request, and only temporarily stores the dump content in memory,
// call Response.Dump() to get the dump content when needed.
//
// EnableDumpEachRequestWithoutResponseBody 为每个 Request 启用 dump，但省略响应正文。
func (c *Client) EnableDumpEachRequestWithoutResponseBody() *Client {
	return c.OnBeforeRequest(func(client *Client, req *Request) error {
		if req.RetryAttempt == 0 { // Ignore on retry, no need to repeat enable dump.
			req.EnableDumpWithoutResponseBody()
		}
		return nil
	})
}

// EnableDumpEachRequestWithoutRequestBody enables dump without request body at the
// request-level for each request, and only temporarily stores the dump content in memory,
// call Response.Dump() to get the dump content when needed.
//
// EnableDumpEachRequestWithoutRequestBody 为每个 Request 启用 dump，但省略请求正文。
func (c *Client) EnableDumpEachRequestWithoutRequestBody() *Client {
	return c.OnBeforeRequest(func(client *Client, req *Request) error {
		if req.RetryAttempt == 0 { // Ignore on retry, no need to repeat enable dump.
			req.EnableDumpWithoutRequestBody()
		}
		return nil
	})
}

// NewParallelDownload creates a ParallelDownload for url using c.
//
// NewParallelDownload 使用当前 Client 和 url 创建 ParallelDownload。
func (c *Client) NewParallelDownload(url string) *ParallelDownload {
	return &ParallelDownload{
		url:    url,
		client: c,
	}
}

// DisableAutoReadResponse disables read response body automatically (enabled by default).
//
// DisableAutoReadResponse 关闭普通响应正文的自动读取；未使用 SetOutput 或 SetOutputFile 时，
// 调用方随后负责读取并关闭 Response.Body。下载流程不受此开关影响。
func (c *Client) DisableAutoReadResponse() *Client {
	c.disableAutoReadResponse = true
	return c
}

// EnableAutoReadResponse enables read response body automatically (enabled by default).
//
// EnableAutoReadResponse 恢复普通响应正文的自动读取；这是默认行为，库会读取并缓存正文，
// 使用 SetOutput 或 SetOutputFile 的请求仍由下载流程处理。
func (c *Client) EnableAutoReadResponse() *Client {
	c.disableAutoReadResponse = false
	return c
}

// SetMaxResponseSize sets the maximum allowed size of a response body in bytes.
//
// Enforcement:
//   - When Response.ContentLength is known and greater than the limit, the body
//     is closed without reading and a ResponseBodyTooLargeError is returned.
//     Closing early may prevent connection reuse for that request.
//   - Otherwise the body is wrapped so application reads stop at the limit.
//   - HEAD requests never fail the Content-Length early check (there is no body).
//
// The limit is applied to bytes delivered to the application after the transport
// has handled Content-Encoding (e.g. gzip decompression). For auto-decompressed
// responses ContentLength is typically -1, so only the streaming limit applies.
// Charset auto-decode, if enabled, also runs underneath the limit.
//
// A value of 0 or less disables the limit (default). This is useful for bounding
// memory use and network bandwidth when talking to untrusted or unexpectedly
// large endpoints.
//
// SetMaxResponseSize 设置响应正文允许的最大字节数；该限制同时约束自动读取、下载和调用方
// 对 Response.Body 的读取，超限时返回 ResponseBodyTooLargeError。max 小于或等于 0 时禁用限制，
// HEAD 响应不会仅因 Content-Length 超限而失败。
func (c *Client) SetMaxResponseSize(max int64) *Client {
	if max < 0 {
		max = 0
	}
	c.maxResponseSize = max
	return c
}

// SetUserAgent sets the "User-Agent" header for requests fired from the client.
//
// SetUserAgent 为此 Client 创建的 Request 设置默认 User-Agent。
func (c *Client) SetUserAgent(userAgent string) *Client {
	return c.SetCommonHeader(header.UserAgent, userAgent)
}

// SetCommonBearerAuthToken sets the bearer auth token for requests fired from the client.
//
// SetCommonBearerAuthToken 为此 Client 创建的 Request 设置默认 Bearer Authorization token。
func (c *Client) SetCommonBearerAuthToken(token string) *Client {
	return c.SetCommonHeader(header.Authorization, "Bearer "+token)
}

// SetCommonAuthToken sets the Authorization header using Bearer scheme.
//
// SetCommonAuthToken 为此 Client 创建的 Request 设置默认 Authorization token。
func (c *Client) SetCommonAuthToken(token string) *Client {
	return c.SetCommonAuthSchemeToken("Bearer", token)
}

// SetCommonAuthSchemeToken sets the Authorization header using a custom auth scheme.
//
// SetCommonAuthSchemeToken 为此 Client 创建的 Request 设置默认 Authorization scheme 和 token。
func (c *Client) SetCommonAuthSchemeToken(scheme, token string) *Client {
	return c.SetCommonHeader(header.Authorization, authSchemeTokenValue(scheme, token))
}

// SetCommonBasicAuth sets the basic auth for requests fired from
// the client.
//
// SetCommonBasicAuth 为此 Client 创建的 Request 设置默认 HTTP Basic 认证用户名和密码。
func (c *Client) SetCommonBasicAuth(username, password string) *Client {
	c.SetCommonHeader(header.Authorization, util.BasicAuthHeaderValue(username, password))
	return c
}

// SetCommonDigestAuth sets the Digest Access auth scheme for requests fired from the client. If a server responds with
// 401 and sends a Digest challenge in the WWW-Authenticate Header, requests will be resent with the appropriate
// Authorization Header.
//
// For Example: To set the Digest scheme with user "roc" and password "123456"
//
//	client.SetCommonDigestAuth("roc", "123456")
//
// Information about Digest Access Authentication can be found in RFC7616:
//
//	https://datatracker.ietf.org/doc/html/rfc7616
//
// SetCommonDigestAuth 为此 Client 创建的 Request 设置默认 HTTP Digest 认证用户名和密码。
func (c *Client) SetCommonDigestAuth(username, password string) *Client {
	c.digestAuth = &digestAuth{
		Username:   username,
		Password:   password,
		HttpClient: c.httpClient,
		cache:      make(map[string]*cchal),
	}
	c.Transport.WrapRoundTrip(c.digestAuth.HttpRoundTripWrapper)
	return c
}

// SetCommonHeaders sets headers for requests fired from the client.
//
// SetCommonHeaders 为此 Client 创建的 Request 批量设置默认 HTTP Header；每个键使用给定的单个值。
func (c *Client) SetCommonHeaders(hdrs map[string]string) *Client {
	for k, v := range hdrs {
		c.SetCommonHeader(k, v)
	}
	return c
}

// SetCommonHeaderAny sets a header value converted from any type with fmt.Sprint.
//
// SetCommonHeaderAny 为此 Client 创建的 Request 设置默认 HTTP Header，并用 fmt.Sprint 转换 value。
func (c *Client) SetCommonHeaderAny(key string, value any) *Client {
	return c.SetCommonHeader(key, fmt.Sprint(value))
}

// SetCommonHeaderValues sets multiple values for a common header key.
//
// SetCommonHeaderValues 为此 Client 创建的 Request 为一个默认 HTTP Header 设置一个或多个值。
func (c *Client) SetCommonHeaderValues(key string, values ...string) *Client {
	if c.Headers == nil {
		c.Headers = make(http.Header)
	}
	c.Headers[http.CanonicalHeaderKey(key)] = slices.Clone(values)
	return c
}

// SetCommonHeaderMultiValues sets multiple common headers whose values may contain more than one entry.
//
// SetCommonHeaderMultiValues 为此 Client 创建的 Request 批量设置带多值的默认 HTTP Header。
func (c *Client) SetCommonHeaderMultiValues(hdrs map[string][]string) *Client {
	for k, v := range hdrs {
		c.SetCommonHeaderValues(k, v...)
	}
	return c
}

// SetCommonHeader sets a header for requests fired from the client.
//
// SetCommonHeader 为此 Client 创建的 Request 设置一个默认 HTTP Header 键值对。
func (c *Client) SetCommonHeader(key, value string) *Client {
	if c.Headers == nil {
		c.Headers = make(http.Header)
	}
	c.Headers.Set(key, value)
	return c
}

// SetCommonHeaderNonCanonical sets a header for requests fired from
// the client which key is a non-canonical key (keep case unchanged),
// only valid for HTTP/1.1.
//
// SetCommonHeaderNonCanonical 为此 Client 创建的 Request 设置一个保持原始大小写的默认 HTTP Header。
func (c *Client) SetCommonHeaderNonCanonical(key, value string) *Client {
	if c.Headers == nil {
		c.Headers = make(http.Header)
	}
	c.Headers[key] = append(c.Headers[key], value)
	return c
}

// SetCommonHeadersNonCanonical sets headers for requests fired from the
// client which key is a non-canonical key (keep case unchanged), only
// valid for HTTP/1.1.
//
// SetCommonHeadersNonCanonical 为此 Client 创建的 Request 批量设置保持原始大小写的默认 HTTP Header。
func (c *Client) SetCommonHeadersNonCanonical(hdrs map[string]string) *Client {
	for k, v := range hdrs {
		c.SetCommonHeaderNonCanonical(k, v)
	}
	return c
}

// SetCommonHeaderOrder sets the order of the http header requests fired from the
// client (case-insensitive).
// For example:
//
//	client.R().SetCommonHeaderOrder(
//	    "custom-header",
//	    "cookie",
//	    "user-agent",
//	    "accept-encoding",
//	).Get(url
//
// SetCommonHeaderOrder 设置此 Client 请求的普通 HTTP Header 发送顺序。
func (c *Client) SetCommonHeaderOrder(keys ...string) *Client {
	c.Transport.WrapRoundTrip(func(rt http.RoundTripper) HttpRoundTripFunc {
		return func(req *http.Request) (resp *http.Response, err error) {
			if req.Header == nil {
				req.Header = make(http.Header)
			}
			req.Header[HeaderOderKey] = keys
			return rt.RoundTrip(req)
		}
	})
	return c
}

// SetCommonPseudoHeaderOder sets the order of the pseudo http header requests fired
// from the client (case-insensitive).
// Note this is only valid for http2 and http3.
// For example:
//
//	client.SetCommonPseudoHeaderOder(
//	    ":scheme",
//	    ":authority",
//	    ":path",
//	    ":method",
//	)
//
// SetCommonPseudoHeaderOder 设置 HTTP/2 伪 Header 的发送顺序；方法名中的 Oder 为既有拼写。
func (c *Client) SetCommonPseudoHeaderOder(keys ...string) *Client {
	c.Transport.WrapRoundTrip(func(rt http.RoundTripper) HttpRoundTripFunc {
		return func(req *http.Request) (resp *http.Response, err error) {
			if req.Header == nil {
				req.Header = make(http.Header)
			}
			req.Header[PseudoHeaderOderKey] = keys
			return rt.RoundTrip(req)
		}
	})
	return c
}

// SetCommonContentType sets the `Content-Type` header for requests fired
// from the client.
//
// SetCommonContentType 为此 Client 创建的 Request 设置默认 Content-Type。
func (c *Client) SetCommonContentType(ct string) *Client {
	c.SetCommonHeader(header.ContentType, ct)
	return c
}

// DisableDumpAll disables dump for requests fired from the client.
//
// DisableDumpAll 关闭此 Client 的全局 dump 设置。
func (c *Client) DisableDumpAll() *Client {
	c.DisableDump()
	return c
}

// DisableDump disables dump for requests fired from the client.
//
// DisableDump 关闭此 Client 的 dump 功能。
func (c *Client) DisableDump() {
	c.Transport.DisableDump()
	c.closeDumpOutput()
}

func (c *Client) closeDumpOutput() {
	if c.dumpOutputCloser == nil {
		return
	}
	if err := c.dumpOutputCloser.Close(); err != nil {
		c.log.Warnf("close dump output error: %v", err)
	}
	c.dumpOutputCloser = nil
	if c.dumpOptions != nil {
		c.dumpOptions.Output = nil
	}
}

// SetCommonDumpOptions configures the underlying Transport's DumpOptions
// for requests fired from the client.
//
// SetCommonDumpOptions 设置此 Client 创建的 Request 默认使用的 DumpOptions。
func (c *Client) SetCommonDumpOptions(opt *DumpOptions) *Client {
	if opt == nil {
		return c
	}
	if opt.Output != nil {
		c.closeDumpOutput()
	}
	if opt.Output == nil {
		if c.dumpOptions != nil {
			opt.Output = c.dumpOptions.Output
		} else {
			opt.Output = os.Stdout
		}
	}
	c.dumpOptions = opt
	if c.Dump != nil {
		c.Dump.SetOptions(dumpOptions{opt})
	}
	return c
}

// OnError sets the error hook which will be executed if any error is returned,
// even if it occurs before the request is sent (e.g. an invalid URL).
//
// OnError 设置请求错误 hook，并替换此前设置的 hook；即使错误发生在发送前（例如 URL 无效）
// 也会调用它。
func (c *Client) OnError(hook ErrorHook) *Client {
	c.onError = hook
	return c
}

// OnBeforeRequest adds a request middleware which hooks before request sent.
//
// OnBeforeRequest 注册请求前中间件；每个 Request 发送前按注册顺序执行。
func (c *Client) OnBeforeRequest(m RequestMiddleware) *Client {
	c.udBeforeRequest = append(c.udBeforeRequest, m)
	return c
}

// OnAfterResponse adds response middleware that runs after each request attempt,
// including attempts that finish with an error.
//
// OnAfterResponse 追加响应后中间件；每次请求尝试结束后（包括发生错误时）按注册顺序执行。
func (c *Client) OnAfterResponse(m ResponseMiddleware) *Client {
	c.afterResponse = append(c.afterResponse, m)
	return c
}

// SetProxyURL sets proxy from the proxy URL.
//
// SetProxyURL 解析并设置此 Client 使用的固定代理 URL。
func (c *Client) SetProxyURL(proxyUrl string) *Client {
	if proxyUrl == "" {
		c.log.Warnf("ignore empty proxy url in SetProxyURL")
		return c
	}
	u, err := urlpkg.Parse(proxyUrl)
	if err != nil {
		c.log.Errorf("failed to parse proxy url %s: %v", proxyUrl, err)
		return c
	}
	proxy := http.ProxyURL(u)
	c.SetProxy(proxy)
	return c
}

// DisableTraceAll disables trace for requests fired from the client.
//
// DisableTraceAll 关闭此 Client 所有 Request 的 trace 收集。
func (c *Client) DisableTraceAll() *Client {
	c.trace = false
	return c
}

// EnableTraceAll enables trace for requests fired from the client (http3
// currently does not support trace).
//
// EnableTraceAll 为此 Client 创建的所有 Request 启用 trace 收集。
func (c *Client) EnableTraceAll() *Client {
	c.trace = true
	return c
}

// SetCookieJar sets the cookie jar to the underlying `http.Client`, set to nil if you
// want to disable cookies.
// Note: If you use Client.Clone to clone a new Client, the new client will share the same
// cookie jar as the old Client after cloning. Use SetCookieJarFactory instead if you want
// to create a new CookieJar automatically when cloning a client.
//
// SetCookieJar 设置底层 http.Client 使用的 CookieJar。
func (c *Client) SetCookieJar(jar http.CookieJar) *Client {
	c.cookiejarFactory = nil
	c.httpClient.Jar = jar
	return c
}

// GetCookies get cookies from the underlying `http.Client`'s `CookieJar`.
//
// GetCookies 从此 Client 的 CookieJar 获取指定 URL 当前匹配的 Cookie。
func (c *Client) GetCookies(url string) ([]*http.Cookie, error) {
	if c.httpClient.Jar == nil {
		return nil, errors.New("cookie jar is not enabled")
	}
	u, err := urlpkg.Parse(url)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Jar.Cookies(u), nil
}

// ClearCookies clears all cookies if cookie is enabled, including
// cookies from cookie jar and cookies set by SetCommonCookies.
// Note: The cookie jar will not be cleared if you called SetCookieJar
// instead of SetCookieJarFactory.
//
// ClearCookies 清除由 CookieJar 和 SetCommonCookies 保存的 Cookie；直接通过 SetCookieJar 设置的 jar 不会被清空。
func (c *Client) ClearCookies() *Client {
	c.initCookieJar()
	c.Cookies = nil
	return c
}

// SetJsonMarshal sets the JSON marshal function which will be used
// to marshal request body.
//
// SetJsonMarshal 设置 JSON 编码函数，用于将请求值序列化为 JSON。
func (c *Client) SetJsonMarshal(fn func(v any) ([]byte, error)) *Client {
	c.jsonMarshal = fn
	return c
}

// SetJsonUnmarshal sets the JSON unmarshal function which will be used
// to unmarshal response body.
//
// SetJsonUnmarshal 设置 JSON 解码函数，用于将响应数据反序列化为 JSON。
func (c *Client) SetJsonUnmarshal(fn func(data []byte, v any) error) *Client {
	c.jsonUnmarshal = fn
	return c
}

// SetXmlMarshal sets the XML marshal function which will be used
// to marshal request body.
//
// SetXmlMarshal 设置 XML 编码函数，用于将请求值序列化为 XML。
func (c *Client) SetXmlMarshal(fn func(v any) ([]byte, error)) *Client {
	c.xmlMarshal = fn
	return c
}

// SetXmlUnmarshal sets the XML unmarshal function which will be used
// to unmarshal response body.
//
// SetXmlUnmarshal 设置 XML 解码函数，用于将响应数据反序列化为 XML。
func (c *Client) SetXmlUnmarshal(fn func(data []byte, v any) error) *Client {
	c.xmlUnmarshal = fn
	return c
}

// SetTLSFingerprintChrome uses tls fingerprint of Chrome browser.
//
// SetTLSFingerprintChrome 将 TLS ClientHello 指纹设为 Chrome 预设。
func (c *Client) SetTLSFingerprintChrome() *Client {
	return c.SetTLSFingerprint(utls.HelloChrome_133)
}

// SetTLSFingerprintFirefox uses tls fingerprint of Firefox browser.
//
// SetTLSFingerprintFirefox 将 TLS ClientHello 指纹设为 Firefox 预设。
func (c *Client) SetTLSFingerprintFirefox() *Client {
	return c.SetTLSFingerprint(utls.HelloFirefox_120)
}

// SetTLSFingerprintEdge uses tls fingerprint of Edge browser.
//
// SetTLSFingerprintEdge 将 TLS ClientHello 指纹设为 Edge 预设。
func (c *Client) SetTLSFingerprintEdge() *Client {
	return c.SetTLSFingerprint(utls.HelloEdge_Auto)
}

// SetTLSFingerprintQQ uses tls fingerprint of QQ browser.
//
// SetTLSFingerprintQQ 将 TLS ClientHello 指纹设为 QQ 预设。
func (c *Client) SetTLSFingerprintQQ() *Client {
	return c.SetTLSFingerprint(utls.HelloQQ_Auto)
}

// SetTLSFingerprintSafari uses tls fingerprint of Safari browser.
//
// SetTLSFingerprintSafari 将 TLS ClientHello 指纹设为 Safari 预设。
func (c *Client) SetTLSFingerprintSafari() *Client {
	return c.SetTLSFingerprint(utls.HelloSafari_16_0)
}

// SetTLSFingerprint360 uses tls fingerprint of 360 browser.
//
// SetTLSFingerprint360 将 TLS ClientHello 指纹设为 360 浏览器预设。
func (c *Client) SetTLSFingerprint360() *Client {
	return c.SetTLSFingerprint(utls.Hello360_Auto)
}

// SetTLSFingerprintIOS uses tls fingerprint of IOS.
//
// SetTLSFingerprintIOS 将 TLS ClientHello 指纹设为 iOS 预设。
func (c *Client) SetTLSFingerprintIOS() *Client {
	return c.SetTLSFingerprint(utls.HelloIOS_Auto)
}

// SetTLSFingerprintAndroid uses tls fingerprint of Android.
//
// SetTLSFingerprintAndroid 将 TLS ClientHello 指纹设为 Android 预设。
func (c *Client) SetTLSFingerprintAndroid() *Client {
	return c.SetTLSFingerprint(utls.HelloAndroid_11_OkHttp)
}

// SetTLSFingerprintRandomized uses randomized tls fingerprint.
//
// SetTLSFingerprintRandomized 将 TLS ClientHello 指纹设为随机化预设。
func (c *Client) SetTLSFingerprintRandomized() *Client {
	return c.SetTLSFingerprint(utls.HelloRandomized)
}

// SetTLSFingerprint sets the tls fingerprint for tls handshake, will use utls
// (https://github.com/refraction-networking/utls) to perform the tls handshake,
// which uses the specified clientHelloID to simulate the tls fingerprint.
// Note this is valid for HTTP1 and HTTP2, not HTTP3.
//
// SetTLSFingerprint 将 TLS ClientHello 指纹设为指定的 utls.ClientHelloID。
func (c *Client) SetTLSFingerprint(clientHelloID utls.ClientHelloID) *Client {
	return c.setTLSFingerprint(clientHelloID, nil)
}

// SetTLSFingerprintSpecFactory sets the TLS fingerprint from a factory that
// returns a fresh custom ClientHelloSpec for every TLS handshake.
// The factory may be called concurrently and must synchronize any shared state.
// Note this is valid for HTTP1 and HTTP2, not HTTP3.
//
// SetTLSFingerprintSpecFactory 设置 TLS ClientHelloSpec 工厂；每次握手通过该函数获取新的 spec。
func (c *Client) SetTLSFingerprintSpecFactory(fn func() *utls.ClientHelloSpec) *Client {
	return c.setTLSFingerprint(utls.HelloCustom, func(conn *uTLSConn) error {
		if fn == nil {
			return errors.New("req: TLS fingerprint spec factory is nil")
		}
		spec := fn()
		if spec == nil {
			return errors.New("req: TLS fingerprint spec factory returned nil")
		}
		return conn.ApplyPreset(spec)
	})
}

// DisableAllowGetMethodPayload disables sending GET method requests with body.
//
// DisableAllowGetMethodPayload 禁止 GET 请求携带请求正文。
func (c *Client) DisableAllowGetMethodPayload() *Client {
	c.AllowGetMethodPayload = false
	return c
}

// EnableAllowGetMethodPayload allows sending GET method requests with body.
//
// EnableAllowGetMethodPayload 允许 GET 请求携带请求正文。
func (c *Client) EnableAllowGetMethodPayload() *Client {
	c.AllowGetMethodPayload = true
	return c
}

func (c *Client) isPayloadForbid(m string) bool {
	return (m == http.MethodGet && !c.AllowGetMethodPayload) || m == http.MethodHead || m == http.MethodOptions
}

// GetClient returns the underlying `http.Client`.
//
// GetClient 返回底层的 http.Client。
func (c *Client) GetClient() *http.Client {
	return c.httpClient
}

func (c *Client) getRetryOption() *RetryOption {
	if c.retryOption == nil {
		c.retryOption = newDefaultRetryOption()
	}
	return c.retryOption
}

// SetCommonRetryCount enables retry and set the maximum retry count for requests
// fired from the client.
// It will retry infinitely if count is negative.
//
// SetCommonRetryCount 启用重试并设置默认最大重试次数；count 为负数时无限重试。
func (c *Client) SetCommonRetryCount(count int) *Client {
	c.getRetryOption().MaxRetries = count
	return c
}

// SetCommonRetryInterval sets the custom GetRetryIntervalFunc for requests fired
// from the client, you can use this to implement your own backoff retry algorithm.
// For example:
//
//		 req.SetCommonRetryInterval(func(resp *req.Response, attempt int) time.Duration {
//	     sleep := 0.01 * math.Exp2(float64(attempt))
//	     return time.Duration(math.Min(2, sleep)) * time.Second
//		 })
//
// SetCommonRetryInterval 设置默认重试间隔计算函数，可用于实现自定义退避算法。
func (c *Client) SetCommonRetryInterval(getRetryIntervalFunc GetRetryIntervalFunc) *Client {
	c.getRetryOption().GetRetryInterval = getRetryIntervalFunc
	return c
}

// SetCommonRetryFixedInterval sets retry to use a fixed interval for requests
// fired from the client.
//
// SetCommonRetryFixedInterval 设置默认重试使用的固定间隔。
func (c *Client) SetCommonRetryFixedInterval(interval time.Duration) *Client {
	c.getRetryOption().GetRetryInterval = func(resp *Response, attempt int) time.Duration {
		return interval
	}
	return c
}

// SetCommonRetryBackoffInterval sets retry to use a capped exponential backoff
// with jitter for requests fired from the client.
// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
//
// SetCommonRetryBackoffInterval 设置默认重试使用带抖动且有上限的指数退避；min 和 max 分别控制下限与上限。
func (c *Client) SetCommonRetryBackoffInterval(min, max time.Duration) *Client {
	c.getRetryOption().GetRetryInterval = backoffInterval(min, max)
	return c
}

// SetCommonRetryHook sets the retry hook which will be executed before a retry.
// It will override other retry hooks if any been added before.
//
// SetCommonRetryHook 设置默认重试 hook；它在每次重试前执行，并替换此前设置的重试 hooks。
func (c *Client) SetCommonRetryHook(hook RetryHookFunc) *Client {
	c.getRetryOption().RetryHooks = []RetryHookFunc{hook}
	return c
}

// AddCommonRetryHook adds a retry hook for requests fired from the client,
// which will be executed before a retry.
//
// AddCommonRetryHook 追加一个默认重试 hook；它在每次重试前执行。
func (c *Client) AddCommonRetryHook(hook RetryHookFunc) *Client {
	ro := c.getRetryOption()
	ro.RetryHooks = append(ro.RetryHooks, hook)
	return c
}

// SetCommonRetryCondition sets the retry condition, which determines whether the
// request should retry.
// It will override other retry conditions if any been added before.
//
// SetCommonRetryCondition 设置默认重试条件，用于决定请求是否应重试，并替换此前设置的条件。
func (c *Client) SetCommonRetryCondition(condition RetryConditionFunc) *Client {
	c.getRetryOption().RetryConditions = []RetryConditionFunc{condition}
	return c
}

// AddCommonRetryCondition adds a retry condition, which determines whether the
// request should retry.
//
// AddCommonRetryCondition 追加一个默认重试条件，用于决定请求是否应重试。
func (c *Client) AddCommonRetryCondition(condition RetryConditionFunc) *Client {
	ro := c.getRetryOption()
	ro.RetryConditions = append(ro.RetryConditions, condition)
	return c
}

// SetUnixSocket sets client to dial connection use unix socket.
// For example:
//
// client.SetUnixSocket("/var/run/custom.sock")
//
// SetUnixSocket 将此 Client 的拨号目标改为指定 Unix socket 文件。
func (c *Client) SetUnixSocket(file string) *Client {
	c.Transport.SetDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", file)
	})
	return c
}

// SetResolver sets a custom DNS resolver used when dialing HTTP/1 and HTTP/2
// connections. It is implemented via SetDial and a net.Dialer that uses r.
// If r is nil, the default resolver is used.
//
// Only valid for HTTP/1 and HTTP/2 (same limitation as SetDial). Calling
// SetDial, SetHosts, or SetUnixSocket replaces this dialer.
//
// For example, use a specific DNS server:
//
//	r := &net.Resolver{
//		PreferGo: true,
//		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
//			var d net.Dialer
//			return d.DialContext(ctx, "udp", "1.1.1.1:53")
//		},
//	}
//	client.SetResolver(r)
//
// SetResolver 设置 HTTP/1 和 HTTP/2 拨号使用的 DNS Resolver；nil 使用默认解析器，随后调用 SetDial、SetHosts 或 SetUnixSocket 会替换它。
func (c *Client) SetResolver(r *net.Resolver) *Client {
	c.Transport.SetDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := net.Dialer{Resolver: r}
		return d.DialContext(ctx, network, addr)
	})
	return c
}

// SetHosts configures a static hostname-to-IP mapping used when dialing HTTP/1
// and HTTP/2 connections (like a hosts file). Hostnames not present in the map
// fail immediately with a "no such host" DNS error and do not consult the
// system resolver. This avoids long DNS timeouts when maintaining a custom
// host list (e.g. crawlers).
//
// Notes:
//   - Keys are hostnames only (no port). Matching is case-insensitive and
//     IDNA-normalized to match the dial address form used by the transport.
//   - Values must be literal IP addresses (IPv4 or IPv6). Optional surrounding
//     brackets on IPv6 (e.g. "[::1]") are accepted and normalized. Non-IP
//     values never fall through to system DNS; dialing that host returns a
//     clear error instead.
//   - IP-literal request addresses (e.g. https://1.2.3.4/) skip the map and
//     dial directly. Scoped IPv6 literals (e.g. https://[fe80::1%25eth0]/)
//     are supported.
//   - An empty or nil map makes every non-literal hostname fail closed.
//   - Proxy routing is rejected while SetHosts is active because a proxy can
//     resolve the destination remotely and bypass the static mapping.
//   - Only valid for HTTP/1 and HTTP/2 (same limitation as SetDial). SetDialTLS
//     still bypasses this dialer for HTTPS when set. Calling SetDial,
//     SetResolver, or SetUnixSocket replaces this dialer.
//   - The map is copied; later changes to the caller's map are ignored.
//
// For example:
//
//	client.SetHosts(map[string]string{
//		"api.internal": "10.0.0.5",
//		"db.internal":  "10.0.0.6",
//		"v6.internal":  "::1",
//	})
//
// SetHosts 设置 HTTP/1 和 HTTP/2 拨号使用的静态主机名到 IP 映射；未命中的非 IP 主机名会失败且不会查询系统 DNS。
func (c *Client) SetHosts(hosts map[string]string) *Client {
	m := make(map[string]string, len(hosts))
	invalid := make(map[string]string)
	for host, ipStr := range hosts {
		key := hostsMapKey(host)
		if key == "" {
			continue
		}
		ip := net.ParseIP(strings.Trim(ipStr, "[]"))
		if ip == nil {
			invalid[key] = ipStr
			continue
		}
		m[key] = ip.String()
	}
	c.SetDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		// IP literals need no DNS and are not subject to the hosts map.
		if _, err := netip.ParseAddr(host); err == nil {
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		}
		key := hostsMapKey(host)
		if raw, bad := invalid[key]; bad {
			return nil, fmt.Errorf("req: SetHosts: invalid IP address %q for host %q", raw, host)
		}
		ip, ok := m[key]
		if !ok {
			return nil, &net.DNSError{
				Err:        "no such host",
				Name:       host,
				IsNotFound: true,
			}
		}
		var d net.Dialer
		return d.DialContext(ctx, network, net.JoinHostPort(ip, port))
	})
	c.Transport.rejectProxyWithSetHosts = true
	return c
}

// hostsMapKey normalizes a hostname for SetHosts lookup: trim, IDNA ToASCII,
// then lowercase so keys match dial addresses produced by the transport.
func hostsMapKey(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if ascii, err := idnaASCII(host); err == nil {
		host = ascii
	}
	return strings.ToLower(host)
}

// Do is compatible with http.Client.Do, which can make req integration easier
// in some scenarios. It should be noted that this will make some req features
// not work properly, such as automatic retry, client middleware, etc.
//
// Do 直接调用底层 http.Client.Do；这会绕过 req 的自动重试和客户端中间件等部分功能。
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// Clone copies and returns the Client.
//
// Clone 返回此 Client 的副本；Transport、http.Client、默认参数、中间件、dump 和重试配置会按需要复制。
func (c *Client) Clone() *Client {
	cc := *c

	// clone Transport
	cc.Transport = c.Transport.Clone()
	cc.initTransport()

	// clone http.Client
	client := *c.httpClient
	client.Transport = cc.Transport
	cc.httpClient = &client
	cc.initCookieJar()

	// clone client middleware
	if len(cc.roundTripWrappers) > 0 {
		cc.wrappedRoundTrip = roundTripImpl{&cc}
		for _, w := range cc.roundTripWrappers {
			cc.wrappedRoundTrip = w(cc.wrappedRoundTrip)
		}
	}

	// clone other fields that may need to be cloned
	cc.PathParams = maps.Clone(c.PathParams)
	cc.RawPathParams = maps.Clone(c.RawPathParams)
	cc.QueryParams = cloneUrlValues(c.QueryParams)
	cc.FormData = cloneUrlValues(c.FormData)
	cc.beforeRequest = slices.Clone(c.beforeRequest)
	cc.udBeforeRequest = slices.Clone(c.udBeforeRequest)
	cc.afterResponse = slices.Clone(c.afterResponse)
	cc.dumpOptions = c.dumpOptions.Clone()
	cc.dumpOutputCloser = nil
	cc.retryOption = c.retryOption.Clone()
	cc.browserProfile = c.browserProfile
	return &cc
}

func memoryCookieJarFactory() http.CookieJar {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return jar
}

// C creates a new client.
//
// C 创建使用默认 Transport、2 分钟超时和内存 CookieJar 的新 Client。
func C() *Client {
	t := T()

	httpClient := &http.Client{
		Transport: t,
		Timeout:   2 * time.Minute,
	}
	beforeRequest := []RequestMiddleware{
		parseRequestHeader,
		parseRequestCookie,
		parseRequestURL,
		parseRequestBody,
		applyBrowserProfileHeaders,
	}
	afterResponse := []ResponseMiddleware{
		parseResponseBody,
		handleDownload,
	}
	c := &Client{
		AllowGetMethodPayload: true,
		beforeRequest:         beforeRequest,
		afterResponse:         afterResponse,
		log:                   createDefaultLogger(),
		httpClient:            httpClient,
		Transport:             t,
		jsonMarshal:           json.Marshal,
		jsonUnmarshal:         json.Unmarshal,
		xmlMarshal:            xml.Marshal,
		xmlUnmarshal:          xml.Unmarshal,
		cookiejarFactory:      memoryCookieJarFactory,
	}
	c.SetRedirectPolicy(DefaultRedirectPolicy())
	c.initCookieJar()

	c.initTransport()
	return c
}

// SetCookieJarFactory sets the factory used to create a cookie jar after cloning.
// SetCookieJarFactory 设置 Client 克隆后创建 CookieJar 的工厂。
func (c *Client) SetCookieJarFactory(factory func() http.CookieJar) *Client {
	c.cookiejarFactory = factory
	c.initCookieJar()
	return c
}

func (c *Client) initCookieJar() {
	if c.cookiejarFactory == nil {
		return
	}
	jar := c.cookiejarFactory()
	if jar != nil {
		c.httpClient.Jar = jar
	}
}

func (c *Client) initTransport() {
	c.Debugf = func(format string, v ...any) {
		if c.DebugLog {
			c.log.Debugf(format, v...)
		}
	}
}

// RoundTripper is the interface of req's Client.
//
// RoundTripper 定义处理 req Request 并返回 Response 的接口。
type RoundTripper interface {
	// RoundTrip processes req and returns its response or an error.
	//
	// RoundTrip 处理 req 并返回响应或错误。
	RoundTrip(*Request) (*Response, error)
}

// RoundTripFunc is a RoundTripper implementation, which is a simple function.
//
// RoundTripFunc 将普通函数适配为 RoundTripper。
type RoundTripFunc func(req *Request) (resp *Response, err error)

// RoundTrip implements RoundTripper.
//
// RoundTrip 调用 fn 处理 req 并返回 Response 或错误。
func (fn RoundTripFunc) RoundTrip(req *Request) (*Response, error) {
	return fn(req)
}

// RoundTripWrapper is client middleware function.
//
// RoundTripWrapper 定义用于包装 RoundTripper 的客户端中间件函数。
type RoundTripWrapper func(rt RoundTripper) RoundTripFunc

type roundTripImpl struct {
	*Client
}

func (r roundTripImpl) RoundTrip(req *Request) (resp *Response, err error) {
	return r.roundTrip(req)
}

// WrapRoundTrip adds a client middleware function that will give the caller
// an opportunity to wrap the underlying http.RoundTripper.
//
// WrapRoundTrip 注册包装底层 RoundTripper 的客户端中间件；未传入 wrapper 时保持现有配置不变。
func (c *Client) WrapRoundTrip(wrappers ...RoundTripWrapper) *Client {
	if len(wrappers) == 0 {
		return c
	}
	if c.wrappedRoundTrip == nil {
		c.roundTripWrappers = wrappers
		c.wrappedRoundTrip = roundTripImpl{c}
	} else {
		c.roundTripWrappers = append(c.roundTripWrappers, wrappers...)
	}
	for _, w := range wrappers {
		c.wrappedRoundTrip = w(c.wrappedRoundTrip)
	}
	return c
}

// RoundTrip implements RoundTripper
func (c *Client) roundTrip(r *Request) (resp *Response, err error) {
	resp = &Response{Request: r}
	defer func() {
		if err != nil {
			resp.Err = err
		} else {
			err = resp.Err
		}
	}()

	// setup trace
	if r.trace == nil && r.client.trace {
		r.trace = &clientTrace{}
	}

	ctx := r.ctx

	if r.trace != nil {
		ctx = r.trace.createContext(r.Context())
	}

	// setup url and host
	var host string
	if h := r.getHeader("Host"); h != "" {
		host = h // Host header override
	} else {
		host = r.URL.Host
	}

	// setup header
	contentLength := int64(len(r.Body))
	if r.contentLengthSet {
		contentLength = r.contentLength
	}

	var reqBody io.ReadCloser
	if r.GetBody != nil {
		reqBody, resp.Err = r.GetBody()
		if resp.Err != nil {
			return
		}
	}
	getBody := r.GetBody
	if r.unReplayableBody != nil {
		getBody = nil
	}
	req := &http.Request{
		Method:        r.Method,
		Header:        r.Headers.Clone(),
		URL:           r.URL,
		Host:          host,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: contentLength,
		Body:          reqBody,
		GetBody:       getBody,
		Close:         r.close,
	}
	for _, cookie := range r.Cookies {
		req.AddCookie(cookie)
	}
	if r.isSaveResponse && r.downloadCallback != nil {
		var wrap wrapResponseBodyFunc = func(rc io.ReadCloser) io.ReadCloser {
			return &callbackReader{
				ReadCloser: rc,
				callback: func(read int64) {
					r.downloadCallback(DownloadInfo{
						Response:       resp,
						DownloadedSize: read,
					})
				},
				lastTime: time.Now(),
				interval: r.downloadCallbackInterval,
			}
		}
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = context.WithValue(ctx, wrapResponseBodyKey, wrap)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	r.RawRequest = req
	r.StartTime = time.Now()

	var httpResponse *http.Response
	httpResponse, resp.Err = c.httpClient.Do(r.RawRequest)
	resp.Response = httpResponse

	// Enforce response body size limit before any body consumption.
	if resp.Err == nil {
		if err := applyMaxResponseSize(r, resp); err != nil {
			resp.Err = err
		}
	}

	// auto-read response body if possible
	if resp.Err == nil && !c.disableAutoReadResponse && !r.isSaveResponse && !r.disableAutoReadResponse && resp.StatusCode > 199 {
		resp.ToBytes()
		// restore body for re-reads
		resp.Body = io.NopCloser(bytes.NewReader(resp.body))
	}

	for _, f := range c.afterResponse {
		if e := f(c, resp); e != nil {
			resp.Err = e
		}
	}
	return
}

// applyMaxResponseSize rejects oversized responses early when Content-Length is
// known, and otherwise wraps the body so reads stop at the configured limit.
func applyMaxResponseSize(r *Request, resp *Response) error {
	max := r.getMaxResponseSize()
	if max <= 0 || resp.Response == nil || resp.Body == nil {
		return nil
	}

	// HEAD keeps Content-Length from the resource header but has no body
	// (see transfer.go). Do not treat that advertised length as a body limit
	// violation — ParallelDownload relies on Head() + ContentLength for sizing.
	if r.Method != http.MethodHead {
		// Known Content-Length over the limit: reject without reading the body so
		// bandwidth and memory are not wasted. Early close may prevent keep-alive reuse.
		if cl := resp.ContentLength; cl > max {
			_ = resp.Body.Close()
			resp.Body = http.NoBody
			return &ResponseBodyTooLargeError{Limit: max, ContentLength: cl}
		}
	}

	resp.Body = &maxResponseBodyReader{
		r:     resp.Body,
		n:     max,
		limit: max,
	}
	return nil
}
