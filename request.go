package req

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/jwwsjlm/req/v3/internal/dump"
	"github.com/jwwsjlm/req/v3/internal/header"
	"github.com/jwwsjlm/req/v3/internal/util"
)

// Request struct is used to compose and fire individual request from
// req client. Request provides lots of chainable settings which can
// override client level settings.
//
// Request 表示由 Client 创建的单次 HTTP 请求；其设置可覆盖 Client 级默认配置。
type Request struct {
	PathParams      map[string]string
	RawPathParams   map[string]string
	QueryParams     urlpkg.Values
	FormData        urlpkg.Values
	OrderedFormData []string
	Headers         http.Header
	Cookies         []*http.Cookie
	Result          any
	Error           any
	RawRequest      *http.Request
	StartTime       time.Time
	RetryAttempt    int
	RawURL          string // read only
	Method          string
	Body            []byte
	GetBody         GetContentFunc
	// URL is an auto-generated field, and is nil in request middleware (OnBeforeRequest),
	// consider using RawURL if you want, it's not nil in client middleware (WrapRoundTripFunc)
	URL *urlpkg.URL

	isMultiPart              bool
	contentLength            int64
	contentLengthSet         bool
	disableAutoReadResponse  bool
	forceChunkedEncoding     bool
	isSaveResponse           bool
	close                    bool
	error                    error
	client                   *Client
	uploadCallback           UploadCallback
	uploadCallbackInterval   time.Duration
	downloadCallback         DownloadCallback
	downloadCallbackInterval time.Duration
	// maxResponseSize, when non-nil, overrides Client.maxResponseSize for this
	// request. A pointed-to value of 0 means no limit for this request.
	maxResponseSize    *int64
	unReplayableBody   io.ReadCloser
	retryOption        *RetryOption
	bodyReadCloser     io.ReadCloser
	dumpOptions        *DumpOptions
	marshalBody        any
	ctx                context.Context
	uploadFiles        []*FileUpload
	uploadReader       []io.ReadCloser
	outputFile         string
	output             io.Writer
	trace              *clientTrace
	dumpBuffer         *bytes.Buffer
	dumpOutputCloser   io.Closer
	responseReturnTime time.Time
	afterResponse      []ResponseMiddleware
}

// GetContentFunc returns a new readable and closable request-content stream.
//
// GetContentFunc 定义用于生成可关闭请求内容读取器的函数。
type GetContentFunc func() (io.ReadCloser, error)

func (r *Request) getHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	return r.Headers.Get(key)
}

// TraceInfo returns the trace information, only available if trace is enabled
// (see Request.EnableTrace and Client.EnableTraceAll).
//
// TraceInfo 返回本次请求的链路追踪数据；仅在 Request.EnableTrace 或
// Client.EnableTraceAll 开启追踪后有效，未开启时返回零值 TraceInfo。
func (r *Request) TraceInfo() TraceInfo {
	ct := r.trace

	if ct == nil {
		return TraceInfo{}
	}
	ts := ct.snapshot()

	ti := TraceInfo{
		IsConnReused:  ts.gotConnInfo.Reused,
		IsConnWasIdle: ts.gotConnInfo.WasIdle,
		ConnIdleTime:  ts.gotConnInfo.IdleTime,
	}

	endTime := ts.endTime
	if endTime.IsZero() { // in case timeout
		endTime = r.responseReturnTime
	}

	if !ts.tlsHandshakeStart.IsZero() {
		if !ts.tlsHandshakeDone.IsZero() {
			ti.TLSHandshakeTime = traceDuration(ts.tlsHandshakeDone, ts.tlsHandshakeStart)
		} else {
			ti.TLSHandshakeTime = traceDuration(endTime, ts.tlsHandshakeStart)
		}
	}

	traceStart := ts.getConn
	if traceStart.IsZero() {
		traceStart = r.StartTime
	}
	if ts.gotConnInfo.Reused {
		ti.TotalTime = traceDuration(endTime, traceStart)
	} else {
		if ts.dnsStart.IsZero() {
			ti.TotalTime = traceDuration(endTime, r.StartTime)
		} else {
			ti.TotalTime = traceDuration(endTime, ts.dnsStart)
		}
	}

	dnsDone := ts.dnsDone
	if dnsDone.IsZero() {
		dnsDone = endTime
	}

	if !ts.dnsStart.IsZero() {
		ti.DNSLookupTime = traceDuration(dnsDone, ts.dnsStart)
	}

	// Only calculate on successful connections
	if !ts.connectDone.IsZero() {
		ti.TCPConnectTime = traceDuration(ts.connectDone, dnsDone)
	}

	// Only calculate on successful connections
	if !ts.gotConn.IsZero() {
		ti.ConnectTime = traceDuration(ts.gotConn, traceStart)
	}

	// Only calculate on successful connections
	if !ts.gotFirstResponseByte.IsZero() {
		firstResponseStart := ts.gotConn
		if firstResponseStart.IsZero() {
			firstResponseStart = traceStart
		}
		ti.FirstResponseTime = traceDuration(ts.gotFirstResponseByte, firstResponseStart)
		ti.ResponseTime = traceDuration(endTime, ts.gotFirstResponseByte)
	}

	// Capture remote address info when connection is non-nil
	if ts.gotConnInfo.Conn != nil {
		ti.RemoteAddr = ts.gotConnInfo.Conn.RemoteAddr()
		ti.LocalAddr = ts.gotConnInfo.Conn.LocalAddr()
	}

	return ti
}

func traceDuration(end, start time.Time) time.Duration {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start)
}

// HeaderToString get all header as string.
//
// HeaderToString 将当前请求的全部 HTTP 头格式化为字符串并返回。
func (r *Request) HeaderToString() string {
	return convertHeaderToString(r.Headers)
}

// SetURL set the url for request.
//
// SetURL 设置请求目标 URL，并返回当前 Request 以便链式调用。
func (r *Request) SetURL(url string) *Request {
	r.RawURL = url
	return r
}

// SetFormDataFromValues set the form data from url.Values, will not
// been used if request method does not allow payload.
//
// SetFormDataFromValues 将 data 中的每个值追加到表单数据；不允许携带请求体的
// HTTP 方法不会发送这些数据。
func (r *Request) SetFormDataFromValues(data urlpkg.Values) *Request {
	if r.FormData == nil {
		r.FormData = urlpkg.Values{}
	}
	for k, v := range data {
		for _, kv := range v {
			r.FormData.Add(k, kv)
		}
	}
	return r
}

// SetFormData set the form data from a map, will not been used
// if request method does not allow payload.
//
// SetFormData 用 data 中的值设置表单字段，同名字段会被覆盖；不允许携带请求体的
// HTTP 方法不会发送这些数据。
func (r *Request) SetFormData(data map[string]string) *Request {
	if r.FormData == nil {
		r.FormData = urlpkg.Values{}
	}
	for k, v := range data {
		r.FormData.Set(k, v)
	}
	return r
}

// SetOrderedFormData set the ordered form data from key-values pairs.
//
// SetOrderedFormData 按给定的 key、value 顺序追加表单字段；kvs 的元素数必须为偶数，
// 否则发送请求时会返回错误。
func (r *Request) SetOrderedFormData(kvs ...string) *Request {
	r.OrderedFormData = append(r.OrderedFormData, kvs...)
	return r
}

// SetFormDataAnyType set the form data from a map, which value could be any type,
// will convert to string automatically.
// It will not been used if request method does not allow payload.
//
// SetFormDataAnyType 用 data 设置表单字段，并通过 fmt.Sprint 将每个值转换为字符串；
// 不允许携带请求体的 HTTP 方法不会发送这些数据。
func (r *Request) SetFormDataAnyType(data map[string]any) *Request {
	if r.FormData == nil {
		r.FormData = urlpkg.Values{}
	}
	for k, v := range data {
		r.FormData.Set(k, fmt.Sprint(v))
	}
	return r
}

// SetFormDataAny is an alias of SetFormDataAnyType.
//
// SetFormDataAny 是 SetFormDataAnyType 的别名，用于设置可包含任意类型值的表单数据。
func (r *Request) SetFormDataAny(data map[string]any) *Request {
	return r.SetFormDataAnyType(data)
}

// SetCookies set http cookies for the request.
//
// SetCookies 将 cookies 追加到当前请求的 Cookie 列表。
func (r *Request) SetCookies(cookies ...*http.Cookie) *Request {
	r.Cookies = append(r.Cookies, cookies...)
	return r
}

// SetQueryString set URL query parameters for the request using
// raw query string.
//
// SetQueryString 解析去除首尾空白后的原始查询字符串，并将其中的所有值追加到 URL
// 查询参数；解析失败时只记录警告，不应用该字符串中的参数。
func (r *Request) SetQueryString(query string) *Request {
	params, err := urlpkg.ParseQuery(strings.TrimSpace(query))
	if err != nil {
		r.client.log.Warnf("failed to parse query string (%s): %v", query, err)
		return r
	}
	if r.QueryParams == nil {
		r.QueryParams = make(urlpkg.Values)
	}
	for p, v := range params {
		for _, pv := range v {
			r.QueryParams.Add(p, pv)
		}
	}
	return r
}

// SetQueryParamsFromValues sets query parameters from a url.Values map.
// This method allows direct configuration of query parameters from url.Values,
// which is commonly used with libraries like go-querystring.
//
// SetQueryParamsFromValues 将 url.Values 中的全部值追加到当前请求的 URL 查询参数，
// 可直接接收 go-querystring 等库生成的结果。
func (r *Request) SetQueryParamsFromValues(params urlpkg.Values) *Request {
	if r.QueryParams == nil {
		r.QueryParams = make(urlpkg.Values)
	}
	for p, v := range params {
		for _, pv := range v {
			r.QueryParams.Add(p, pv)
		}
	}
	return r
}

// SetQueryParamsFromStruct sets query parameters from a struct using go-querystring.
// This method provides a higher-level abstraction by allowing users to directly pass
// a struct to configure query parameters. The struct should use `url` tags to specify
// parameter names.
//
// SetQueryParamsFromStruct 使用 go-querystring 和结构体的 `url` 标签生成查询参数并追加到请求；
// 转换失败时只记录警告，不修改查询参数。
func (r *Request) SetQueryParamsFromStruct(v any) *Request {
	values, err := query.Values(v)
	if err != nil {
		r.client.log.Warnf("failed to convert struct to query parameters: %v", err)
		return r
	}
	return r.SetQueryParamsFromValues(values)
}

// SetFileReader set up a multipart form with a reader to upload file.
//
// SetFileReader 将 reader 作为名为 paramName、文件名为 filename 的 multipart 文件上传。
// 该读取器不可重放，因此为此请求启用重试会在发送前返回错误。
func (r *Request) SetFileReader(paramName, filename string, reader io.Reader) *Request {
	if rc, ok := reader.(io.ReadCloser); ok {
		r.unReplayableBody = rc
	} else {
		r.unReplayableBody = io.NopCloser(reader)
	}
	r.SetFileUpload(FileUpload{
		ParamName: paramName,
		FileName:  filename,
		GetFileContent: func() (io.ReadCloser, error) {
			return r.unReplayableBody, nil
		},
	})
	return r
}

// SetMultipartField sets up a multipart form part from a reader with an explicit Content-Type.
//
// SetMultipartField 将 reader 作为 multipart 文件字段上传，并显式设置字段名、文件名和
// Content-Type。
func (r *Request) SetMultipartField(paramName, filename, contentType string, reader io.Reader) *Request {
	r.SetFileUpload(FileUpload{
		ParamName:   paramName,
		FileName:    filename,
		ContentType: contentType,
		GetFileContent: func() (io.ReadCloser, error) {
			if rc, ok := reader.(io.ReadCloser); ok {
				return rc, nil
			}
			return io.NopCloser(reader), nil
		},
	})
	return r
}

// SetFileBytes set up a multipart form with given []byte to upload.
//
// SetFileBytes 将 content 作为 multipart 文件字段上传，并根据字节内容自动设置文件大小和
// Content-Type；该内容可在重试时重新读取。
func (r *Request) SetFileBytes(paramName, filename string, content []byte) *Request {
	r.SetFileUpload(FileUpload{
		ParamName:   paramName,
		FileName:    filename,
		FileSize:    int64(len(content)),
		ContentType: http.DetectContentType(content),
		GetFileContent: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(content)), nil
		},
	})
	return r
}

// SetFiles set up a multipart form from a map to upload, which
// key is the parameter name, and value is the file path.
//
// SetFiles 按“表单字段名到本地文件路径”的映射添加多个 multipart 文件；某个文件无法打开、
// 读取或获取信息时，错误会记录到当前 Request，并在发送时返回。
func (r *Request) SetFiles(files map[string]string) *Request {
	for k, v := range files {
		r.SetFile(k, v)
	}
	return r
}

// SetFile set up a multipart form from file path to upload,
// which read file from filePath automatically to upload.
//
// SetFile 将 filePath 指向的本地文件添加为名为 paramName 的 multipart 文件字段；文件名取
// 路径基名，并自动检测大小和 Content-Type。文件访问错误会记录到 Request 并在发送时返回。
func (r *Request) SetFile(paramName, filePath string) *Request {
	file, err := os.Open(filePath)
	if err != nil {
		r.client.log.Errorf("failed to open %s: %v", filePath, err)
		r.appendError(err)
		return r
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		file.Close()
		r.client.log.Errorf("failed to stat file %s: %v", filePath, err)
		r.appendError(err)
		return r
	}
	cbuf := make([]byte, 512)
	n, readErr := file.Read(cbuf)
	file.Close()
	if readErr != nil && readErr != io.EOF {
		r.client.log.Errorf("failed to read %s: %v", filePath, readErr)
		r.appendError(readErr)
		return r
	}
	r.isMultiPart = true
	return r.SetFileUpload(FileUpload{
		ParamName: paramName,
		FileName:  filepath.Base(filePath),
		GetFileContent: func() (io.ReadCloser, error) {
			return os.Open(filePath)
		},
		FileSize:    fileInfo.Size(),
		ContentType: http.DetectContentType(cbuf[:n]),
	})
}

var (
	errMissingParamName   = errors.New("missing param name in multipart file upload")
	errMissingFileName    = errors.New("missing filename in multipart file upload")
	errMissingFileContent = errors.New("missing file content in multipart file upload")
)

// SetFileUpload set the fully customized multipart file upload options.
//
// SetFileUpload 添加完全自定义的 multipart 文件上传项。每项都必须提供 ParamName、FileName
// 和 GetFileContent；缺少任一字段的项不会加入上传列表，错误会记录到当前 Request。
func (r *Request) SetFileUpload(uploads ...FileUpload) *Request {
	r.isMultiPart = true
	for _, upload := range uploads {
		shouldAppend := true
		if upload.ParamName == "" {
			r.appendError(errMissingParamName)
			shouldAppend = false
		}
		if upload.FileName == "" {
			r.appendError(errMissingFileName)
			shouldAppend = false
		}
		if upload.GetFileContent == nil {
			r.appendError(errMissingFileContent)
			shouldAppend = false
		}
		if shouldAppend {
			r.uploadFiles = append(r.uploadFiles, &upload)
		}
	}
	return r
}

// SetUploadCallback sets the UploadCallback used to report multipart upload progress.
// Progress notifications are throttled to once every 200ms, with a final notification
// when a file with a known size is completely written.
//
// SetUploadCallback 设置 multipart 上传进度回调，进度通知默认最多每 200ms 调用一次；
// 已知文件大小时，文件写完还会通知一次。nil 回调会被忽略，非 nil 回调会同时强制分块传输。
func (r *Request) SetUploadCallback(callback UploadCallback) *Request {
	return r.SetUploadCallbackWithInterval(callback, 200*time.Millisecond)
}

// SetUploadCallbackWithInterval sets the UploadCallback used to report multipart upload
// progress. Progress notifications are throttled to once every minInterval, with a final
// notification when a file with a known size is completely written.
//
// SetUploadCallbackWithInterval 设置 multipart 上传进度回调及最小通知间隔 minInterval；
// nil 回调会被忽略，非 nil 回调会同时强制分块传输。
func (r *Request) SetUploadCallbackWithInterval(callback UploadCallback, minInterval time.Duration) *Request {
	if callback == nil {
		return r
	}
	r.forceChunkedEncoding = true
	r.uploadCallback = callback
	r.uploadCallbackInterval = minInterval
	return r
}

// SetDownloadCallback sets the DownloadCallback used to report response-body download
// progress. Progress notifications are throttled to once every 200ms, with a final
// notification after the last bytes are read.
//
// SetDownloadCallback 设置响应体下载进度回调，进度通知默认最多每 200ms 调用一次，并在读取
// 完成时通知剩余进度。nil 回调会被忽略；该回调仅在 SetOutput 或 SetOutputFile 下载时生效。
func (r *Request) SetDownloadCallback(callback DownloadCallback) *Request {
	return r.SetDownloadCallbackWithInterval(callback, 200*time.Millisecond)
}

// SetDownloadCallbackWithInterval sets the DownloadCallback used to report response-body
// download progress. Progress notifications are throttled to once every minInterval,
// with a final notification after the last bytes are read.
//
// SetDownloadCallbackWithInterval 设置响应体下载进度回调及最小通知间隔 minInterval；nil 回调
// 会被忽略，且该回调仅在 SetOutput 或 SetOutputFile 下载时生效。
func (r *Request) SetDownloadCallbackWithInterval(callback DownloadCallback, minInterval time.Duration) *Request {
	if callback == nil {
		return r
	}
	r.downloadCallback = callback
	r.downloadCallbackInterval = minInterval
	return r
}

// SetResult set the result that response Body will be unmarshalled to if
// no error occurs and Response.ResultState() returns SuccessState, by default
// it requires HTTP status `code >= 200 && code <= 299`, you can also use
// Client.SetResultStateCheckFunc to customize the result state check logic.
//
// SetResult 为成功状态的响应体设置反序列化目标；它是 SetSuccessResult 的已弃用别名。
// Deprecated: Use SetSuccessResult instead.
func (r *Request) SetResult(result any) *Request {
	return r.SetSuccessResult(result)
}

// SetSuccessResult set the result that response Body will be unmarshalled to if
// no error occurs and Response.ResultState() returns SuccessState, by default
// it requires HTTP status `code >= 200 && code <= 299`, you can also use
// Client.SetResultStateCheckFunc to customize the result state check logic.
//
// SetSuccessResult 设置成功响应体的反序列化目标。默认仅 200 到 299 状态码属于成功状态，
// 可通过 Client.SetResultStateCheckFunc 自定义；result 为 nil 时不做修改。
func (r *Request) SetSuccessResult(result any) *Request {
	if result == nil {
		return r
	}
	r.Result = util.GetPointer(result)
	return r
}

// SetError set the result that response body will be unmarshalled to if
// no error occurs and Response.ResultState() returns ErrorState, by default
// it requires HTTP status `code >= 400`, you can also use
// Client.SetResultStateCheckFunc to customize the result state check logic.
//
// SetError 为错误状态的响应体设置反序列化目标；它是 SetErrorResult 的已弃用别名。
// Deprecated: Use SetErrorResult result.
func (r *Request) SetError(err any) *Request {
	return r.SetErrorResult(err)
}

// SetErrorResult set the result that response body will be unmarshalled to if
// no error occurs and Response.ResultState() returns ErrorState, by default
// it requires HTTP status `code >= 400`, you can also
// use Client.SetResultStateCheckFunc to customize the result state check logic.
//
// SetErrorResult 设置错误响应体的反序列化目标。默认状态码不小于 400 时属于错误状态，
// 可通过 Client.SetResultStateCheckFunc 自定义；err 为 nil 时不做修改。
func (r *Request) SetErrorResult(err any) *Request {
	if err == nil {
		return r
	}
	r.Error = util.GetPointer(err)
	return r
}

// SetBearerAuthToken set bearer auth token for the request.
//
// SetBearerAuthToken 将请求的 Authorization 头设置为 Bearer 方案和给定 token。
func (r *Request) SetBearerAuthToken(token string) *Request {
	return r.SetHeader(header.Authorization, "Bearer "+token)
}

func authSchemeTokenValue(scheme, token string) string {
	scheme = strings.TrimSpace(scheme)
	if scheme == "" {
		return token
	}
	if token == "" {
		return scheme
	}
	return scheme + " " + token
}

// SetAuthToken sets the Authorization header using Bearer scheme.
//
// SetAuthToken 使用 Bearer 认证方案和给定 token 设置 Authorization 请求头。
func (r *Request) SetAuthToken(token string) *Request {
	return r.SetAuthSchemeToken("Bearer", token)
}

// SetAuthSchemeToken sets the Authorization header using a custom auth scheme.
//
// SetAuthSchemeToken 使用自定义认证方案和 token 设置 Authorization 请求头。scheme 会去除
// 首尾空白；scheme 为空时只写入 token，token 为空时只写入 scheme。
func (r *Request) SetAuthSchemeToken(scheme, token string) *Request {
	return r.SetHeader(header.Authorization, authSchemeTokenValue(scheme, token))
}

// SetBasicAuth set basic auth for the request.
//
// SetBasicAuth 使用 username 和 password 设置 HTTP Basic Authorization 请求头。
func (r *Request) SetBasicAuth(username, password string) *Request {
	return r.SetHeader(header.Authorization, util.BasicAuthHeaderValue(username, password))
}

// SetDigestAuth sets the Digest Access auth scheme for the HTTP request. If a server responds with 401 and sends a
// Digest challenge in the WWW-Authenticate Header, the request will be resent with the appropriate Authorization Header.
//
// For Example: To set the Digest scheme with username "roc" and password "123456"
//
//	client.R().SetDigestAuth("roc", "123456")
//
// Information about Digest Access Authentication can be found in RFC7616:
//
//	https://datatracker.ietf.org/doc/html/rfc7616
//
// SetDigestAuth 为当前请求注册 Digest 认证处理：收到 401 Digest challenge 后携带计算出的
// Authorization 头重新发送请求。
// Deprecated: Use Client.SetCommonDigestAuth instead. Request level digest auth is not recommended,
func (r *Request) SetDigestAuth(username, password string) *Request {
	r.OnAfterResponse(handleDigestAuthFunc(username, password))
	return r
}

// OnAfterResponse adds response middleware that runs after each request attempt,
// including attempts that finish with an error.
//
// OnAfterResponse 将响应中间件 m 追加到当前请求；每次请求尝试结束后（包括发生错误时）
// 按注册顺序执行这些中间件。
func (r *Request) OnAfterResponse(m ResponseMiddleware) *Request {
	r.afterResponse = append(r.afterResponse, m)
	return r
}

// SetHeaders set headers from a map for the request.
//
// SetHeaders 用 hdrs 批量设置请求头；每个键会按 http.Header.Set 的规则规范化，
// 同名请求头的旧值会被替换。
func (r *Request) SetHeaders(hdrs map[string]string) *Request {
	for k, v := range hdrs {
		r.SetHeader(k, v)
	}
	return r
}

// SetHeader set a header for the request.
//
// SetHeader 设置单个请求头；键名会规范化，同名请求头的全部旧值会被 value 替换。
func (r *Request) SetHeader(key, value string) *Request {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}
	r.Headers.Set(key, value)
	return r
}

// SetHeaderAny sets a header value converted from any type with fmt.Sprint.
//
// SetHeaderAny 通过 fmt.Sprint 将 value 转为字符串后设置请求头，并替换同名头的旧值。
func (r *Request) SetHeaderAny(key string, value any) *Request {
	return r.SetHeader(key, fmt.Sprint(value))
}

// SetHeaderValues sets multiple values for a header key.
//
// SetHeaderValues 将规范化后的请求头 key 设置为 values 的副本，并替换该键已有的全部值。
func (r *Request) SetHeaderValues(key string, values ...string) *Request {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}
	r.Headers[http.CanonicalHeaderKey(key)] = cloneSlice(values)
	return r
}

// SetHeaderMultiValues sets multiple headers whose values may contain more than one entry.
//
// SetHeaderMultiValues 批量设置可包含多个值的请求头；每个键的旧值都会被给定值列表替换。
func (r *Request) SetHeaderMultiValues(headers map[string][]string) *Request {
	for k, v := range headers {
		r.SetHeaderValues(k, v...)
	}
	return r
}

// SetHeadersNonCanonical set headers from a map for the request which key is a
// non-canonical key (keep case unchanged), only valid for HTTP/1.1.
//
// SetHeadersNonCanonical 批量添加保持原始大小写的请求头；每个值会追加到精确键名下，
// 且此设置仅对 HTTP/1.1 有效。
func (r *Request) SetHeadersNonCanonical(hdrs map[string]string) *Request {
	for k, v := range hdrs {
		r.SetHeaderNonCanonical(k, v)
	}
	return r
}

// SetHeaderNonCanonical set a header for the request which key is a
// non-canonical key (keep case unchanged), only valid for HTTP/1.1.
//
// SetHeaderNonCanonical 使用不经规范化的 key 追加一个请求头值，以保留键名大小写；
// 此设置仅对 HTTP/1.1 有效。
func (r *Request) SetHeaderNonCanonical(key, value string) *Request {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}
	r.Headers[key] = append(r.Headers[key], value)
	return r
}

const (
	// HeaderOderKey is the key of header order, which specifies the order
	// of the http header.
	HeaderOderKey = "__header_order__"
	// PseudoHeaderOderKey is the key of pseudo header order, which specifies
	// the order of the http2 and http3 pseudo header.
	PseudoHeaderOderKey = "__pseudo_header_order__"
)

// SetHeaderOrder set the order of the http header (case-insensitive).
// For example:
//
//	client.R().SetHeaderOrder(
//	    "custom-header",
//	    "cookie",
//	    "user-agent",
//	    "accept-encoding",
//	)
//
// SetHeaderOrder 按给定顺序追加普通 HTTP 请求头的发送顺序；键名匹配不区分大小写。
func (r *Request) SetHeaderOrder(keys ...string) *Request {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}
	r.Headers[HeaderOderKey] = append(r.Headers[HeaderOderKey], keys...)
	return r
}

// SetPseudoHeaderOrder set the order of the pseudo http header (case-insensitive).
// Note this is only valid for http2 and http3.
// For example:
//
//	client.R().SetPseudoHeaderOrder(
//	    ":scheme",
//	    ":authority",
//	    ":path",
//	    ":method",
//	)
//
// SetPseudoHeaderOrder 按给定顺序追加伪首部的发送顺序；键名匹配不区分大小写，
// 且仅对 HTTP/2 和 HTTP/3 有效。
func (r *Request) SetPseudoHeaderOrder(keys ...string) *Request {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}
	r.Headers[PseudoHeaderOderKey] = append(r.Headers[PseudoHeaderOderKey], keys...)
	return r
}

// SetOutputFile set the file that response Body will be downloaded to.
//
// SetOutputFile 将响应体下载到 file。相对路径会基于 Client 配置的输出目录解析，
// 下载时会创建缺失的父目录并覆盖同名文件。
func (r *Request) SetOutputFile(file string) *Request {
	r.isSaveResponse = true
	r.outputFile = file
	return r
}

// SetOutput set the io.Writer that response Body will be downloaded to.
//
// SetOutput 将响应体下载到 output；output 为 nil 时只记录警告并保持原配置不变。
func (r *Request) SetOutput(output io.Writer) *Request {
	if output == nil {
		r.client.log.Warnf("nil io.Writer is not allowed in SetOutput")
		return r
	}
	r.output = output
	r.isSaveResponse = true
	return r
}

// SetQueryParams set URL query parameters from a map for the request.
//
// SetQueryParams 用 params 批量设置 URL 查询参数，每个键的已有值会被给定值替换。
func (r *Request) SetQueryParams(params map[string]string) *Request {
	for k, v := range params {
		r.SetQueryParam(k, v)
	}
	return r
}

// SetQueryParamsAnyType set URL query parameters from a map for the request.
// The value of map is any type, will be convert to string automatically.
//
// SetQueryParamsAnyType 用 params 批量设置 URL 查询参数，并通过 fmt.Sprint 将每个值
// 转换为字符串；每个键的已有值会被替换。
func (r *Request) SetQueryParamsAnyType(params map[string]any) *Request {
	for k, v := range params {
		r.SetQueryParam(k, fmt.Sprint(v))
	}
	return r
}

// SetQueryParam set an URL query parameter for the request.
//
// SetQueryParam 设置一个 URL 查询参数，并用 value 替换该键已有的全部值。
func (r *Request) SetQueryParam(key, value string) *Request {
	if r.QueryParams == nil {
		r.QueryParams = make(urlpkg.Values)
	}
	r.QueryParams.Set(key, value)
	return r
}

// SetQueryParamAny sets a query parameter value converted from any type with fmt.Sprint.
//
// SetQueryParamAny 通过 fmt.Sprint 将 value 转为字符串后设置一个 URL 查询参数，
// 并替换该键已有的全部值。
func (r *Request) SetQueryParamAny(key string, value any) *Request {
	return r.SetQueryParam(key, fmt.Sprint(value))
}

// AddQueryParam add a URL query parameter for the request.
//
// AddQueryParam 为 URL 查询参数 key 追加一个 value，保留该键已有的值。
func (r *Request) AddQueryParam(key, value string) *Request {
	if r.QueryParams == nil {
		r.QueryParams = make(urlpkg.Values)
	}
	r.QueryParams.Add(key, value)
	return r
}

// AddQueryParams add one or more values of specified URL query parameter for the request.
//
// AddQueryParams 为 URL 查询参数 key 按给定顺序追加一个或多个值，保留已有值。
func (r *Request) AddQueryParams(key string, values ...string) *Request {
	if r.QueryParams == nil {
		r.QueryParams = make(urlpkg.Values)
	}
	vs := r.QueryParams[key]
	vs = append(vs, values...)
	r.QueryParams[key] = vs
	return r
}

// SetPathParams set URL path parameters from a map for the request.
//
// SetPathParams 批量设置路径参数；发送请求时，URL 中的每个 `{key}` 会替换为经过
// url.PathEscape 转义的值。
func (r *Request) SetPathParams(params map[string]string) *Request {
	for key, value := range params {
		r.SetPathParam(key, value)
	}
	return r
}

// SetPathParamAny sets a path parameter value converted from any type with fmt.Sprint.
//
// SetPathParamAny 通过 fmt.Sprint 将 value 转为字符串后设置路径参数；替换 URL 中
// `{key}` 时会进行 url.PathEscape 转义。
func (r *Request) SetPathParamAny(key string, value any) *Request {
	return r.SetPathParam(key, fmt.Sprint(value))
}

// SetPathParam set a URL path parameter for the request.
//
// SetPathParam 设置路径参数；发送请求时，URL 中的每个 `{key}` 会替换为经过
// url.PathEscape 转义的 value。
func (r *Request) SetPathParam(key, value string) *Request {
	if r.PathParams == nil {
		r.PathParams = make(map[string]string)
	}
	r.PathParams[key] = value
	return r
}

// SetPathRawParam sets a URL path parameter without url.PathEscape.
//
// SetPathRawParam 设置原始路径参数；发送请求时，URL 中的每个 `{key}` 会直接替换为
// value，不执行 url.PathEscape，调用方需自行保证其安全有效。
func (r *Request) SetPathRawParam(key, value string) *Request {
	if r.RawPathParams == nil {
		r.RawPathParams = make(map[string]string)
	}
	r.RawPathParams[key] = value
	return r
}

// SetPathRawParamAny sets a raw path parameter value converted from any type with fmt.Sprint.
//
// SetPathRawParamAny 通过 fmt.Sprint 将 value 转为字符串后设置原始路径参数；替换
// URL 中的 `{key}` 时不执行 url.PathEscape。
func (r *Request) SetPathRawParamAny(key string, value any) *Request {
	return r.SetPathRawParam(key, fmt.Sprint(value))
}

// SetPathRawParams sets multiple URL path parameters without url.PathEscape.
//
// SetPathRawParams 批量设置原始路径参数；发送请求时直接替换 URL 中的 `{key}`，
// 不执行 url.PathEscape。
func (r *Request) SetPathRawParams(params map[string]string) *Request {
	for key, value := range params {
		r.SetPathRawParam(key, value)
	}
	return r
}

func (r *Request) appendError(err error) {
	r.error = errors.Join(r.error, err)
}

var errRetryableWithUnReplayableBody = errors.New("retryable request should not have unreplayable Body (io.Reader)")

const maxRetryBodyDrainSize = 2 << 10

func (r *Request) newErrorResponse(err error) *Response {
	resp := &Response{Request: r}
	resp.Err = err
	return resp
}

func closeRetryResponseBody(resp *Response) {
	if resp == nil || resp.Response == nil || resp.Body == nil {
		return
	}
	if resp.ContentLength >= 0 && resp.ContentLength <= maxRetryBodyDrainSize {
		_, _ = io.Copy(io.Discard, resp.Body)
	}
	_ = resp.Body.Close()
}

func (r *Request) waitRetryInterval(interval time.Duration) error {
	if interval <= 0 {
		return r.Context().Err()
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-r.Context().Done():
		return r.Context().Err()
	}
}

// Do fires http request, 0 or 1 context is allowed, and returns the *Response which
// is always not nil, and Response.Err is not nil if error occurs.
//
// Do 发送当前 Request。若传入 context 且首个参数非 nil，它会用于本次请求；返回的
// Response 始终非 nil，发送或处理失败时错误保存在 Response.Err 中。
func (r *Request) Do(ctx ...context.Context) *Response {
	if len(ctx) > 0 && ctx[0] != nil {
		r.ctx = ctx[0]
	}

	defer func() {
		r.responseReturnTime = time.Now()
	}()
	defer r.closeDumpOutput()
	if r.error != nil {
		return r.newErrorResponse(r.error)
	}
	if r.retryOption != nil && r.retryOption.MaxRetries != 0 && r.unReplayableBody != nil { // retryable request should not have unreplayable Body
		return r.newErrorResponse(errRetryableWithUnReplayableBody)
	}
	resp, _ := r.do()
	return resp
}

func (r *Request) shouldRetry(resp *Response, err error) bool {
	if errors.Is(err, context.Canceled) || r.retryOption == nil ||
		(r.RetryAttempt >= r.retryOption.MaxRetries && r.retryOption.MaxRetries >= 0) {
		return false
	}
	needRetry := err != nil
	if l := len(r.retryOption.RetryConditions); l > 0 {
		for i := l - 1; i >= 0; i-- {
			needRetry = r.retryOption.RetryConditions[i](resp, err)
			if needRetry {
				break
			}
		}
	}
	return needRetry
}

func (r *Request) prepareRetry(resp *Response, err error) error {
	r.RetryAttempt++
	if l := len(r.retryOption.RetryHooks); l > 0 {
		for i := l - 1; i >= 0; i-- { // run retry hooks in reverse order
			r.retryOption.RetryHooks[i](resp, err)
		}
	}
	retryInterval := r.retryOption.GetRetryInterval(resp, r.RetryAttempt)
	closeRetryResponseBody(resp)
	if err = r.waitRetryInterval(retryInterval); err != nil {
		return err
	}

	if r.dumpBuffer != nil {
		r.dumpBuffer.Reset()
	}
	if r.trace != nil {
		r.trace = &clientTrace{}
	}
	if resp != nil {
		resp.body = nil
		resp.result = nil
		resp.error = nil
	}
	return nil
}

// tryRetry attempts retry after an error. It returns true if the request loop should continue.
func (r *Request) tryRetry(resp **Response, err error) (bool, error) {
	if *resp == nil {
		*resp = &Response{Request: r}
	}
	if err != nil {
		(*resp).Err = err
	}
	if !r.shouldRetry(*resp, err) {
		return false, nil
	}
	if err = r.prepareRetry(*resp, err); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Request) do() (resp *Response, err error) {
	defer func() {
		if resp == nil {
			resp = &Response{Request: r}
		}
		if err != nil && resp.Err == nil {
			resp.Err = err
		}
	}()

retry:
	for {
		if r.Headers == nil {
			r.Headers = make(http.Header)
		}
		for _, f := range r.client.udBeforeRequest {
			if err = f(r.client, r); err != nil {
				if ok, retryErr := r.tryRetry(&resp, err); retryErr != nil {
					err = retryErr
					return
				} else if ok {
					continue retry
				}
				return
			}
		}
		for _, f := range r.client.beforeRequest {
			if err = f(r.client, r); err != nil {
				if ok, retryErr := r.tryRetry(&resp, err); retryErr != nil {
					err = retryErr
					return
				} else if ok {
					continue retry
				}
				return
			}
		}

		if r.client.wrappedRoundTrip != nil {
			resp, err = r.client.wrappedRoundTrip.RoundTrip(r)
		} else {
			resp, err = r.client.roundTrip(r)
		}

		// Determine if the error is from a canceled context.
		// Store it here so it doesn't get lost when processing the AfterResponse middleware.
		contextCanceled := errors.Is(err, context.Canceled)

		for _, f := range r.afterResponse {
			if err = f(r.client, resp); err != nil {
				return
			}
		}

		if contextCanceled {
			return
		}
		if ok, retryErr := r.tryRetry(&resp, err); retryErr != nil {
			err = retryErr
			return
		} else if !ok {
			return
		}
	}
}

// Send fires http request with specified method and url, returns the
// *Response which is always not nil, and the error is not nil if error occurs.
//
// Send 设置 method 和 url 后发送请求；返回的 Response 始终非 nil，失败时第二个返回值
// 与 Response.Err 相同，并会调用 Client 配置的错误处理函数。
func (r *Request) Send(method, url string) (*Response, error) {
	r.Method = method
	r.RawURL = url
	resp := r.Do()
	if resp.Err != nil && r.client.onError != nil {
		r.client.onError(r.client, r, resp, resp.Err)
	}
	return resp, resp.Err
}

// MustGet like Get, panic if error happens, should only be used to
// test without error handling.
//
// MustGet 发送 GET 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustGet(url string) *Response {
	resp, err := r.Get(url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Get fires http request with GET method and the specified URL.
//
// Get 使用指定 URL 发送 GET 请求，并返回非 nil 的 Response 及请求错误。
func (r *Request) Get(url string) (*Response, error) {
	return r.Send(http.MethodGet, url)
}

// MustPost like Post, panic if error happens. should only be used to
// test without error handling.
//
// MustPost 发送 POST 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustPost(url string) *Response {
	resp, err := r.Post(url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Post fires http request with POST method and the specified URL.
//
// Post 使用指定 URL 发送 POST 请求，并返回非 nil 的 Response 及请求错误。
func (r *Request) Post(url string) (*Response, error) {
	return r.Send(http.MethodPost, url)
}

// MustPut like Put, panic if error happens, should only be used to
// test without error handling.
//
// MustPut 发送 PUT 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustPut(url string) *Response {
	resp, err := r.Put(url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Put fires http request with PUT method and the specified URL.
//
// Put 使用指定 URL 发送 PUT 请求，并返回非 nil 的 Response 及请求错误。
func (r *Request) Put(url string) (*Response, error) {
	return r.Send(http.MethodPut, url)
}

// MustPatch like Patch, panic if error happens, should only be used
// to test without error handling.
//
// MustPatch 发送 PATCH 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustPatch(url string) *Response {
	resp, err := r.Patch(url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Patch fires http request with PATCH method and the specified URL.
//
// Patch 使用指定 URL 发送 PATCH 请求，并返回非 nil 的 Response 及请求错误。
func (r *Request) Patch(url string) (*Response, error) {
	return r.Send(http.MethodPatch, url)
}

// MustDelete like Delete, panic if error happens, should only be used
// to test without error handling.
//
// MustDelete 发送 DELETE 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustDelete(url string) *Response {
	resp, err := r.Delete(url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Delete fires http request with DELETE method and the specified URL.
//
// Delete 使用指定 URL 发送 DELETE 请求，并返回非 nil 的 Response 及请求错误。
func (r *Request) Delete(url string) (*Response, error) {
	return r.Send(http.MethodDelete, url)
}

// MustOptions like Options, panic if error happens, should only be
// used to test without error handling.
//
// MustOptions 发送 OPTIONS 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustOptions(url string) *Response {
	resp, err := r.Options(url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Options fires http request with OPTIONS method and the specified URL.
//
// Options 使用指定 URL 发送 OPTIONS 请求，并返回非 nil 的 Response 及请求错误。
func (r *Request) Options(url string) (*Response, error) {
	return r.Send(http.MethodOptions, url)
}

// MustHead like Head, panic if error happens, should only be used
// to test without error handling.
//
// MustHead 发送 HEAD 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustHead(url string) *Response {
	resp, err := r.Send(http.MethodHead, url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Head fires http request with HEAD method and the specified URL.
//
// Head 使用指定 URL 发送 HEAD 请求，并返回非 nil 的 Response 及请求错误。
func (r *Request) Head(url string) (*Response, error) {
	return r.Send(http.MethodHead, url)
}

// MustQuery like Query, panic if error happens, should only be used
// to test without error handling.
//
// MustQuery 发送 QUERY 请求并返回 Response；发生任何请求错误时 panic，仅适合无需显式处理错误的测试代码。
func (r *Request) MustQuery(url string) *Response {
	resp, err := r.Query(url)
	if err != nil {
		panic(err)
	}
	return resp
}

// Query fires http request with QUERY method and the specified URL. QUERY is a
// safe, idempotent method that carries the query as request content, defined in
// RFC 10008.
//
// Query 使用指定 URL 发送 RFC 10008 定义的 QUERY 请求；该方法安全且幂等，
// 查询内容通过请求体携带。
func (r *Request) Query(url string) (*Response, error) {
	return r.Send("QUERY", url)
}

// SetBody set the request Body, accepts string, []byte, io.Reader, map and struct.
//
// SetBody 设置请求体。string 和 []byte 会保存为可重放内容；io.Reader/io.ReadCloser
// 作为不可重放流使用；GetContentFunc 可按需创建新流；结构体、映射、切片和数组会在发送时
// 根据 Content-Type 序列化（未指定时使用 JSON）；其他类型通过 fmt.Sprint 转为字符串。
// body 为 nil 时不做修改。
func (r *Request) SetBody(body any) *Request {
	if body == nil {
		return r
	}
	switch b := body.(type) {
	case io.ReadCloser:
		r.unReplayableBody = b
		r.GetBody = func() (io.ReadCloser, error) {
			return r.unReplayableBody, nil
		}
	case io.Reader:
		r.unReplayableBody = io.NopCloser(b)
		r.GetBody = func() (io.ReadCloser, error) {
			return r.unReplayableBody, nil
		}
	case []byte:
		r.SetBodyBytes(b)
	case string:
		r.SetBodyString(b)
	case func() (io.ReadCloser, error):
		r.GetBody = b
	case GetContentFunc:
		r.GetBody = b
	default:
		t := reflect.TypeOf(body)
		switch t.Kind() {
		case reflect.Ptr, reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			r.marshalBody = body
		default:
			r.SetBodyString(fmt.Sprint(body))
		}
	}
	return r
}

// SetBodyBytes set the request Body as []byte.
//
// SetBodyBytes 将 body 设置为请求体，并配置 GetBody 以便重试等场景重新读取这些字节。
func (r *Request) SetBodyBytes(body []byte) *Request {
	r.Body = body
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return r
}

// SetBodyString set the request Body as string.
//
// SetBodyString 将 body 转换为 []byte 后设置为可重放的请求体。
func (r *Request) SetBodyString(body string) *Request {
	return r.SetBodyBytes([]byte(body))
}

// SetBodyJsonString set the request Body as string and set Content-Type header
// as "application/json; charset=utf-8"
//
// SetBodyJsonString 将 body 字符串设置为可重放的请求体，并把 Content-Type 设为
// "application/json; charset=utf-8"；该方法不会验证 JSON 语法。
func (r *Request) SetBodyJsonString(body string) *Request {
	return r.SetBodyJsonBytes([]byte(body))
}

// SetBodyJsonBytes set the request Body as []byte and set Content-Type header
// as "application/json; charset=utf-8"
//
// SetBodyJsonBytes 将 body 字节设置为可重放的请求体，并把 Content-Type 设为
// "application/json; charset=utf-8"；该方法不会验证 JSON 语法。
func (r *Request) SetBodyJsonBytes(body []byte) *Request {
	r.SetContentType(header.JsonContentType)
	return r.SetBodyBytes(body)
}

// SetBodyJsonMarshal set the request Body that marshaled from object, and
// set Content-Type header as "application/json; charset=utf-8"
//
// SetBodyJsonMarshal 使用当前 Client 的 JSON 编码器序列化 v，将结果设为可重放请求体，
// 并设置 JSON Content-Type；序列化错误会记录到 Request 并在发送时返回。
func (r *Request) SetBodyJsonMarshal(v any) *Request {
	b, err := r.client.jsonMarshal(v)
	if err != nil {
		r.appendError(err)
		return r
	}
	return r.SetBodyJsonBytes(b)
}

// SetBodyXmlString set the request Body as string and set Content-Type header
// as "text/xml; charset=utf-8"
//
// SetBodyXmlString 将 body 字符串设置为可重放的请求体，并把 Content-Type 设为
// "text/xml; charset=utf-8"；该方法不会验证 XML 语法。
func (r *Request) SetBodyXmlString(body string) *Request {
	return r.SetBodyXmlBytes([]byte(body))
}

// SetBodyXmlBytes set the request Body as []byte and set Content-Type header
// as "text/xml; charset=utf-8"
//
// SetBodyXmlBytes 将 body 字节设置为可重放的请求体，并把 Content-Type 设为
// "text/xml; charset=utf-8"；该方法不会验证 XML 语法。
func (r *Request) SetBodyXmlBytes(body []byte) *Request {
	r.SetContentType(header.XmlContentType)
	return r.SetBodyBytes(body)
}

// SetBodyXmlMarshal set the request Body that marshaled from object, and
// set Content-Type header as "text/xml; charset=utf-8"
//
// SetBodyXmlMarshal 使用当前 Client 的 XML 编码器序列化 v，将结果设为可重放请求体，
// 并设置 XML Content-Type；序列化错误会记录到 Request 并在发送时返回。
func (r *Request) SetBodyXmlMarshal(v any) *Request {
	b, err := r.client.xmlMarshal(v)
	if err != nil {
		r.appendError(err)
		return r
	}
	return r.SetBodyXmlBytes(b)
}

// SetContentType set the `Content-Type` for the request.
//
// SetContentType 设置请求的 Content-Type 头，并替换已有值。
func (r *Request) SetContentType(contentType string) *Request {
	return r.SetHeader(header.ContentType, contentType)
}

// SetContentLength overrides the request Content-Length.
//
// SetContentLength 显式覆盖请求的 Content-Length；该值会优先于根据请求体计算的长度。
func (r *Request) SetContentLength(length int64) *Request {
	r.contentLength = length
	r.contentLengthSet = true
	return r
}

// Context method returns the Context if its already set in request
// otherwise it creates new one using `context.Background()`.
//
// Context 返回当前请求的 context；尚未设置时创建并保存 context.Background()。
func (r *Request) Context() context.Context {
	if r.ctx == nil {
		r.ctx = context.Background()
	}
	return r.ctx
}

// SetContext method sets the context.Context for current Request. It allows
// to interrupt the request execution if ctx.Done() channel is closed.
// See https://blog.golang.org/context article and the "context" package
// documentation.
//
// Attention: make sure call SetContext before EnableDumpXXX if you want to
// dump at the request level.
//
// SetContext 设置当前请求使用的 context，使请求可在 ctx.Done() 关闭时中断；nil 会被忽略。
// 若还要启用请求级 dump，应先调用本方法，以免替换掉 dump 所保存的 context 值。
func (r *Request) SetContext(ctx context.Context) *Request {
	if ctx != nil {
		r.ctx = ctx
	}
	return r
}

// SetContextData sets the key-value pair data for current Request, so you
// can access some extra context info for current Request in hook or middleware.
//
// SetContextData 使用 context.WithValue 在当前请求 context 中保存 key 和 val，供钩子或
// 中间件读取；key 必须满足 context.WithValue 对可比较类型的要求。
func (r *Request) SetContextData(key, val any) *Request {
	r.ctx = context.WithValue(r.Context(), key, val)
	return r
}

// GetContextData returns the context data of specified key, which set by SetContextData.
//
// GetContextData 返回当前请求 context 中与 key 关联的值；不存在时返回 nil。
func (r *Request) GetContextData(key any) any {
	return r.Context().Value(key)
}

// DisableAutoReadResponse disable read response body automatically (enabled by default).
//
// DisableAutoReadResponse 仅为当前请求关闭响应体自动读取；该功能默认开启，关闭后调用方
// 应自行读取并关闭 Response.Body，SetOutput 或 SetOutputFile 下载不受此开关影响。
func (r *Request) DisableAutoReadResponse() *Request {
	r.disableAutoReadResponse = true
	return r
}

// EnableAutoReadResponse enable read response body automatically (enabled by default).
//
// EnableAutoReadResponse 为当前请求恢复响应体自动读取；是否最终自动读取仍受 Client 级
// DisableAutoReadResponse 配置约束，且输出到文件或 Writer 的请求走下载流程。
func (r *Request) EnableAutoReadResponse() *Request {
	r.disableAutoReadResponse = false
	return r
}

// SetMaxResponseSize sets the maximum allowed size of the response body in bytes
// for this request, overriding Client.SetMaxResponseSize. A value of 0 or less
// disables the limit for this request even if the client has a limit configured.
//
// See Client.SetMaxResponseSize for behavior details.
//
// SetMaxResponseSize 设置当前请求允许的最大响应体字节数，并覆盖 Client 级限制；
// max 小于或等于 0 时禁用本请求的大小限制。
func (r *Request) SetMaxResponseSize(max int64) *Request {
	if max < 0 {
		max = 0
	}
	r.maxResponseSize = &max
	return r
}

// getMaxResponseSize returns the effective max response body size for this
// request (request override, else client setting). 0 means no limit.
func (r *Request) getMaxResponseSize() int64 {
	if r.maxResponseSize != nil {
		return *r.maxResponseSize
	}
	if r.client != nil {
		return r.client.maxResponseSize
	}
	return 0
}

// DisableTrace disables trace.
//
// DisableTrace 关闭当前请求的链路追踪并丢弃已保存的追踪状态。
func (r *Request) DisableTrace() *Request {
	r.trace = nil
	return r
}

// EnableTrace enables trace (http3 currently does not support trace).
//
// EnableTrace 开启当前请求的链路追踪；重复调用不会重置已有追踪状态，HTTP/3 当前不支持追踪。
func (r *Request) EnableTrace() *Request {
	if r.trace == nil {
		r.trace = &clientTrace{}
	}
	return r
}

func (r *Request) getDumpBuffer() *bytes.Buffer {
	if r.dumpBuffer == nil {
		r.dumpBuffer = new(bytes.Buffer)
	}
	return r.dumpBuffer
}

func (r *Request) getDumpOptions() *DumpOptions {
	if r.dumpOptions == nil {
		r.dumpOptions = &DumpOptions{
			RequestHeader:  true,
			RequestBody:    true,
			ResponseHeader: true,
			ResponseBody:   true,
			Output:         r.getDumpBuffer(),
		}
	}
	return r.dumpOptions
}

// EnableDumpTo enables dump and save to the specified io.Writer.
//
// EnableDumpTo 为当前请求启用 dump，并将默认包含请求头、请求体、响应头和响应体的内容
// 写入 output；output 为 nil 时写入标准错误，若此前使用文件作为输出则先关闭该文件。
func (r *Request) EnableDumpTo(output io.Writer) *Request {
	r.closeDumpOutput()
	r.getDumpOptions().Output = output
	return r.EnableDump()
}

// EnableDumpToFile enables dump and save to the specified filename.
//
// EnableDumpToFile 创建或截断 filename 并将当前请求的 dump 写入其中；文件创建失败时
// 错误会记录到 Request 并在发送时返回，已打开的 dump 文件会在请求结束后关闭。
func (r *Request) EnableDumpToFile(filename string) *Request {
	r.closeDumpOutput()
	file, err := os.Create(filename)
	if err != nil {
		r.appendError(err)
		return r
	}
	r.getDumpOptions().Output = file
	r.dumpOutputCloser = file
	return r.EnableDump()
}

func (r *Request) closeDumpOutput() {
	if r.dumpOutputCloser == nil {
		return
	}
	if err := r.dumpOutputCloser.Close(); err != nil {
		r.client.log.Warnf("close dump output error: %v", err)
	}
	r.dumpOutputCloser = nil
	if r.dumpOptions != nil {
		r.dumpOptions.Output = r.getDumpBuffer()
	}
}

// SetDumpOptions sets DumpOptions at request level.
//
// SetDumpOptions 设置当前请求的 DumpOptions。opt 为 nil 时不做修改；Output 为 nil 时使用
// Request 内部缓冲区。若已有选项则复制字段值，否则保存 opt 指针。
func (r *Request) SetDumpOptions(opt *DumpOptions) *Request {
	if opt == nil {
		return r
	}
	r.closeDumpOutput()
	if opt.Output == nil {
		opt.Output = r.getDumpBuffer()
	}
	if r.dumpOptions != nil {
		*r.dumpOptions = *opt
	} else {
		r.dumpOptions = opt
	}
	return r
}

// EnableDump enables dump, including all content for the request and response by default.
//
// EnableDump 为当前请求启用 dump；默认记录请求头、请求体、响应头和响应体，未指定输出时
// 写入 Request 的内部缓冲区。
func (r *Request) EnableDump() *Request {
	return r.SetContext(context.WithValue(r.Context(), dump.DumperKey, newDumper(r.getDumpOptions())))
}

// EnableDumpWithoutBody enables dump only header for the request and response.
//
// EnableDumpWithoutBody 在现有 dump 选项上关闭请求体和响应体记录并启用 dump；
// 其他开关保持不变，新请求默认仍会记录请求头和响应头。
func (r *Request) EnableDumpWithoutBody() *Request {
	o := r.getDumpOptions()
	o.RequestBody = false
	o.ResponseBody = false
	return r.EnableDump()
}

// EnableDumpWithoutHeader enables dump only Body for the request and response.
//
// EnableDumpWithoutHeader 在现有 dump 选项上关闭请求头和响应头记录并启用 dump；
// 其他开关保持不变，新请求默认仍会记录请求体和响应体。
func (r *Request) EnableDumpWithoutHeader() *Request {
	o := r.getDumpOptions()
	o.RequestHeader = false
	o.ResponseHeader = false
	return r.EnableDump()
}

// EnableDumpWithoutResponse enables dump only request.
//
// EnableDumpWithoutResponse 在现有 dump 选项上关闭响应头和响应体记录并启用 dump；
// 其他开关保持不变，新请求默认仍会记录请求内容。
func (r *Request) EnableDumpWithoutResponse() *Request {
	o := r.getDumpOptions()
	o.ResponseHeader = false
	o.ResponseBody = false
	return r.EnableDump()
}

// EnableDumpWithoutRequest enables dump only response.
//
// EnableDumpWithoutRequest 在现有 dump 选项上关闭请求头和请求体记录并启用 dump；
// 其他开关保持不变，新请求默认仍会记录响应内容。
func (r *Request) EnableDumpWithoutRequest() *Request {
	o := r.getDumpOptions()
	o.RequestHeader = false
	o.RequestBody = false
	return r.EnableDump()
}

// EnableDumpWithoutRequestBody enables dump with request Body excluded,
// can be used in upload request to avoid dump the unreadable binary content.
//
// EnableDumpWithoutRequestBody 在现有 dump 选项上关闭请求体记录并启用 dump，其他开关
// 保持不变；适合上传时避免记录二进制文件内容。
func (r *Request) EnableDumpWithoutRequestBody() *Request {
	o := r.getDumpOptions()
	o.RequestBody = false
	return r.EnableDump()
}

// EnableDumpWithoutResponseBody enables dump with response Body excluded,
// can be used in download request to avoid dump the unreadable binary content.
//
// EnableDumpWithoutResponseBody 在现有 dump 选项上关闭响应体记录并启用 dump，其他开关
// 保持不变；适合下载时避免记录二进制文件内容。
func (r *Request) EnableDumpWithoutResponseBody() *Request {
	o := r.getDumpOptions()
	o.ResponseBody = false
	return r.EnableDump()
}

// EnableForceChunkedEncoding enables force using chunked encoding when uploading.
//
// EnableForceChunkedEncoding 强制上传请求使用分块传输编码，而不预先计算 multipart 内容长度。
func (r *Request) EnableForceChunkedEncoding() *Request {
	r.forceChunkedEncoding = true
	return r
}

// DisableForceChunkedEncoding disables force using chunked encoding when uploading.
//
// DisableForceChunkedEncoding 取消强制分块上传；可计算 multipart 内容长度时将发送 Content-Length。
func (r *Request) DisableForceChunkedEncoding() *Request {
	r.forceChunkedEncoding = false
	return r
}

// EnableForceMultipart enables force using multipart to upload form data.
//
// EnableForceMultipart 强制将表单数据编码为 multipart/form-data，即使请求中没有文件上传项。
func (r *Request) EnableForceMultipart() *Request {
	r.isMultiPart = true
	return r
}

// DisableForceMultipart disables force using multipart to upload form data.
//
// DisableForceMultipart 取消强制 multipart；没有文件上传项时，表单数据按普通表单格式编码。
func (r *Request) DisableForceMultipart() *Request {
	r.isMultiPart = false
	return r
}

func (r *Request) getRetryOption() *RetryOption {
	if r.retryOption == nil {
		r.retryOption = newDefaultRetryOption()
	}
	return r.retryOption
}

// GetRetryOption returns the retry configuration of this request.
// It returns nil if retry has not been configured (neither via
// Client.SetCommonRetry* nor Request.SetRetry*).
//
// The returned value is the live option used by this request: mutations
// affect subsequent retries on the same request. Treat it as read-only
// unless you intentionally want to change retry behavior from middleware.
//
// This is useful in middleware to inspect MaxRetries together with
// Request.RetryAttempt, e.g. to report errors only after the configured
// retry budget is exhausted:
//
//	client.OnAfterResponse(func(c *req.Client, resp *req.Response) error {
//	    ro := resp.Request.GetRetryOption()
//	    if ro == nil {
//	        return nil
//	    }
//	    // Cover HTTP error statuses and transport failures (resp.Err with
//	    // no Response). Client OnAfterResponse still runs after failed Do.
//	    failed := resp.IsErrorState() || resp.Err != nil
//	    // RetryAttempt >= MaxRetries only detects budget exhaustion. Retries
//	    // may also stop earlier when a RetryCondition returns false; in that
//	    // case RetryAttempt can be less than MaxRetries on a terminal failure.
//	    if failed && ro.MaxRetries >= 0 &&
//	        resp.Request.RetryAttempt >= ro.MaxRetries {
//	        // report once after final failure
//	    }
//	    return nil
//	})
//
// GetRetryOption 返回当前请求实际持有的重试配置；从未通过 Client 或 Request 配置重试时
// 返回 nil。返回的是可变的实时指针，修改它会影响该请求后续的重试行为。
func (r *Request) GetRetryOption() *RetryOption {
	return r.retryOption
}

// SetRetryCount enables retry and set the maximum retry count.
// It will retry infinitely if count is negative.
//
// SetRetryCount 启用并设置当前请求的最大重试次数，不包括首次请求；count 为 0 时不重试，
// 为负数时无限重试。
func (r *Request) SetRetryCount(count int) *Request {
	r.getRetryOption().MaxRetries = count
	return r
}

// SetRetryInterval sets the custom GetRetryIntervalFunc, you can use this to
// implement your own backoff retry algorithm.
// For example:
//
//	req.SetRetryInterval(func(resp *req.Response, attempt int) time.Duration {
//	    sleep := 0.01 * math.Exp2(float64(attempt))
//	    return time.Duration(math.Min(2, sleep)) * time.Second
//	})
//
// SetRetryInterval 设置函数 getRetryIntervalFunc，用响应和当前重试序号计算下一次重试前的
// 等待时间，可用于实现自定义退避算法；启用重试时该函数不得为 nil。
func (r *Request) SetRetryInterval(getRetryIntervalFunc GetRetryIntervalFunc) *Request {
	r.getRetryOption().GetRetryInterval = getRetryIntervalFunc
	return r
}

// SetRetryFixedInterval set retry to use a fixed interval.
//
// SetRetryFixedInterval 将每次重试前的等待时间固定为 interval；非正值表示不等待。
func (r *Request) SetRetryFixedInterval(interval time.Duration) *Request {
	r.getRetryOption().GetRetryInterval = func(resp *Response, attempt int) time.Duration {
		return interval
	}
	return r
}

// SetRetryBackoffInterval set retry to use a capped exponential backoff with jitter.
// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
//
// SetRetryBackoffInterval 使用带抖动且以 max 为上限的指数退避计算重试间隔，min 为基础间隔。
func (r *Request) SetRetryBackoffInterval(min, max time.Duration) *Request {
	r.getRetryOption().GetRetryInterval = backoffInterval(min, max)
	return r
}

// SetRetryHook set the retry hook which will be executed before a retry.
// It will override other retry hooks if any been added before (including
// client-level retry hooks).
//
// SetRetryHook 将当前请求的重试前钩子替换为唯一的 hook，同时覆盖从 Client 继承及此前添加的钩子；
// hook 为 nil 时，若实际进入重试会 panic。
func (r *Request) SetRetryHook(hook RetryHookFunc) *Request {
	r.getRetryOption().RetryHooks = []RetryHookFunc{hook}
	return r
}

// AddRetryHook adds a retry hook which will be executed before a retry.
//
// AddRetryHook 追加一个重试前钩子；实际重试前，所有钩子按添加顺序的逆序执行；
// hook 为 nil 时，若执行到该钩子会 panic。
func (r *Request) AddRetryHook(hook RetryHookFunc) *Request {
	ro := r.getRetryOption()
	ro.RetryHooks = append(ro.RetryHooks, hook)
	return r
}

// SetRetryCondition sets the retry condition, which determines whether the
// request should retry.
// It will override other retry conditions if any been added before (including
// client-level retry conditions).
//
// SetRetryCondition 将当前请求的重试条件替换为唯一的 condition，同时覆盖从 Client 继承及
// 此前添加的条件；condition 为 nil 时，判断是否重试会 panic。
func (r *Request) SetRetryCondition(condition RetryConditionFunc) *Request {
	r.getRetryOption().RetryConditions = []RetryConditionFunc{condition}
	return r
}

// AddRetryCondition adds a retry condition, which determines whether the
// request should retry.
//
// AddRetryCondition 追加一个判断是否重试的条件；条件按添加顺序的逆序检查，任一返回 true
// 即触发重试，全部返回 false 时不重试；condition 为 nil 时，执行到该条件会 panic。
func (r *Request) AddRetryCondition(condition RetryConditionFunc) *Request {
	ro := r.getRetryOption()
	ro.RetryConditions = append(ro.RetryConditions, condition)
	return r
}

// SetClient change the client of request dynamically.
//
// SetClient 动态切换当前请求使用的 Client；client 为 nil 时保持原 Client 不变。
func (r *Request) SetClient(client *Client) *Request {
	if client != nil {
		r.client = client
	}
	return r
}

// GetClient returns the current client used by request.
//
// GetClient 返回当前请求用于发送和处理响应的 Client。
func (r *Request) GetClient() *Client {
	return r.client
}

// EnableCloseConnection closes the connection after sending this
// request and reading its response if set to true in HTTP/1.1 and
// HTTP/2.
//
// Setting this field prevents reuse of TCP connections between
// requests to the same hosts event if EnableKeepAlives() were called.
//
// EnableCloseConnection 要求 HTTP/1.1 或 HTTP/2 在本次请求及响应完成后关闭连接，
// 从而阻止同一主机的后续请求复用该 TCP 连接，即使 Client 已启用 keep-alive。
func (r *Request) EnableCloseConnection() *Request {
	r.close = true
	return r
}
