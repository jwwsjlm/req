package req

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jwwsjlm/req/v3/internal/header"
	"github.com/jwwsjlm/req/v3/internal/util"
)

// maxResponseBodyPreallocateSize caps how much memory a peer-controlled
// Content-Length can make us allocate before the first read. The hint is only
// used for small, common API responses; larger and unknown bodies keep using
// io.ReadAll's adaptive growth strategy.
//
// 该上限限制不可信 Content-Length 触发的预分配量。仅对常见小响应使用长度提示，
// 大响应和未知长度响应继续使用 io.ReadAll 的自适应扩容，稳定性优先。
const maxResponseBodyPreallocateSize = 8 << 10

// ErrResponseBodyTooLarge is returned when a response body exceeds the limit
// configured via Client.SetMaxResponseSize or Request.SetMaxResponseSize.
// Use errors.Is(err, ErrResponseBodyTooLarge) or errors.As with
// *ResponseBodyTooLargeError for inspection.
var ErrResponseBodyTooLarge = errors.New("req: response body too large")

// ResponseBodyTooLargeError is returned when a response body exceeds the
// configured maximum size. Limit is the configured max in bytes. ContentLength
// is the response's Content-Length when the body was rejected based on headers
// without reading; it is -1 when the limit was exceeded while reading.
type ResponseBodyTooLargeError struct {
	Limit         int64
	ContentLength int64
}

// Error returns a message describing the configured limit and observed response size.
// Error 返回描述响应体限制以及已知响应大小的错误消息。
func (e *ResponseBodyTooLargeError) Error() string {
	if e.ContentLength >= 0 {
		return fmt.Sprintf("req: response body too large: Content-Length %d exceeds limit %d", e.ContentLength, e.Limit)
	}
	return fmt.Sprintf("req: response body too large: exceeds limit of %d bytes", e.Limit)
}

// Is reports whether target is ErrResponseBodyTooLarge.
// Is 报告目标错误是否为 ErrResponseBodyTooLarge。
func (e *ResponseBodyTooLargeError) Is(target error) bool {
	return target == ErrResponseBodyTooLarge
}

// maxResponseBodyReader limits how many bytes can be read from a response body.
// It is similar in spirit to http.MaxBytesReader but for client response bodies:
// when the limit is exceeded it returns *ResponseBodyTooLargeError and subsequent
// reads return the same sticky error. Close closes the underlying body.
type maxResponseBodyReader struct {
	r     io.ReadCloser
	n     int64 // bytes remaining
	limit int64 // original limit, for the error
	err   error // sticky error
}

func (l *maxResponseBodyReader) Read(p []byte) (n int, err error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	// Read at most remaining+1 so we can detect crossing the limit.
	if int64(len(p))-1 > l.n {
		p = p[:l.n+1]
	}
	n, err = l.r.Read(p)

	if int64(n) <= l.n {
		l.n -= int64(n)
		l.err = err
		return n, err
	}

	n = int(l.n)
	l.n = 0
	l.err = &ResponseBodyTooLargeError{Limit: l.limit, ContentLength: -1}
	return n, l.err
}

func (l *maxResponseBodyReader) Close() error {
	return l.r.Close()
}

// Response is the http response.
type Response struct {
	// The underlying http.Response is embed into Response.
	*http.Response
	// Err is the underlying error, not nil if some error occurs.
	// Usually used in the ResponseMiddleware, you can skip logic in
	// ResponseMiddleware that doesn't need to be executed when err occurs.
	Err error
	// Request is the Response's related Request.
	Request    *Request
	body       []byte
	receivedAt time.Time
	error      any
	result     any
}

// IsSuccess method returns true if no error occurs and HTTP status `code >= 200 and <= 299`
// by default, you can also use Client.SetResultStateCheckFunc to customize the result
// state check logic.
// IsSuccess 默认在未发生错误且 HTTP 状态码为 2xx 时返回 true；可通过
// Client.SetResultStateCheckFunc 自定义，该方法已弃用。
//
// Deprecated: Use IsSuccessState instead.
func (r *Response) IsSuccess() bool {
	return r.IsSuccessState()
}

// IsSuccessState method returns true if no error occurs and HTTP status `code >= 200 and <= 299`
// by default, you can also use Client.SetResultStateCheckFunc to customize the result state
// check logic.
// IsSuccessState 默认在存在底层响应且结果状态为 SuccessState 时返回 true；
// 可通过 Client.SetResultStateCheckFunc 自定义状态判定。
func (r *Response) IsSuccessState() bool {
	if r.Response == nil {
		return false
	}
	return r.ResultState() == SuccessState
}

// IsError method returns true if no error occurs and HTTP status `code >= 400`
// by default, you can also use Client.SetResultStateCheckFunc to customize the result
// state check logic.
// IsError 默认在未发生错误且 HTTP 状态码不小于 400 时返回 true；可通过
// Client.SetResultStateCheckFunc 自定义，该方法已弃用。
//
// Deprecated: Use IsErrorState instead.
func (r *Response) IsError() bool {
	return r.IsErrorState()
}

// IsErrorState method returns true if no error occurs and HTTP status `code >= 400`
// by default, you can also use Client.SetResultStateCheckFunc to customize the result
// state check logic.
// IsErrorState 默认在存在底层响应且结果状态为 ErrorState 时返回 true；
// 可通过 Client.SetResultStateCheckFunc 自定义状态判定。
func (r *Response) IsErrorState() bool {
	if r.Response == nil {
		return false
	}
	return r.ResultState() == ErrorState
}

// GetContentType returns the Content-Type response header, or an empty string without a response.
// GetContentType 返回响应的 Content-Type Header；没有底层响应时返回空字符串。
func (r *Response) GetContentType() string {
	if r.Response == nil {
		return ""
	}
	return r.Header.Get(header.ContentType)
}

// ResultState returns the result state.
// By default, it returns SuccessState if HTTP status `code >= 200 && code <= 299`, and returns
// ErrorState if HTTP status `code >= 400`, otherwise returns UnknownState.
// You can also use Client.SetResultStateCheckFunc to customize the result
// state check logic.
// ResultState 返回结果状态；默认 2xx 为 SuccessState、不小于 400 为
// ErrorState，其余为 UnknownState，也可由 Client.SetResultStateCheckFunc 自定义。
func (r *Response) ResultState() ResultState {
	if r.Response == nil {
		return UnknownState
	}
	var resultStateCheckFunc func(resp *Response) ResultState
	if r.Request.client.resultStateCheckFunc != nil {
		resultStateCheckFunc = r.Request.client.resultStateCheckFunc
	} else {
		resultStateCheckFunc = defaultResultStateChecker
	}
	return resultStateCheckFunc(r)
}

// Result returns the automatically unmarshalled object if Request.SetSuccessResult
// is called and ResultState returns SuccessState.
// Otherwise, return nil.
// Result 返回由 Request.SetSuccessResult 自动反序列化的成功结果；该方法已弃用。
//
// Deprecated: Use SuccessResult instead.
func (r *Response) Result() any {
	return r.SuccessResult()
}

// SuccessResult returns the automatically unmarshalled object if Request.SetSuccessResult
// is called and ResultState returns SuccessState.
// Otherwise, return nil.
// SuccessResult 返回由 Request.SetSuccessResult 自动反序列化的成功结果，
// 未配置结果或状态不是 SuccessState 时返回 nil。
func (r *Response) SuccessResult() any {
	return r.result
}

// Error returns the automatically unmarshalled object when Request.SetErrorResult
// or Client.SetCommonErrorResult is called, and ResultState returns ErrorState.
// Otherwise, return nil.
// Error 返回由 Request.SetErrorResult 或 Client.SetCommonErrorResult 自动反序列化的
// 错误结果；该方法已弃用。
//
// Deprecated: Use ErrorResult instead.
func (r *Response) Error() any {
	return r.error
}

// ErrorResult returns the automatically unmarshalled object when Request.SetErrorResult
// or Client.SetCommonErrorResult is called, and ResultState returns ErrorState.
// Otherwise, return nil.
// ErrorResult 返回自动反序列化的错误结果，未配置错误结果或状态不是 ErrorState 时返回 nil。
func (r *Response) ErrorResult() any {
	return r.error
}

// TraceInfo returns the TraceInfo from Request.
// TraceInfo 返回关联请求收集的链路跟踪信息。
func (r *Response) TraceInfo() TraceInfo {
	return r.Request.TraceInfo()
}

// TotalTime returns the elapsed time from sending the request until response processing completed.
// TotalTime 返回从发送请求到响应处理完成的总耗时。
func (r *Response) TotalTime() time.Duration {
	if r.Request.trace != nil {
		return r.Request.TraceInfo().TotalTime
	}
	if !r.receivedAt.IsZero() {
		return r.receivedAt.Sub(r.Request.StartTime)
	}
	return r.Request.responseReturnTime.Sub(r.Request.StartTime)
}

// ReceivedAt returns the time at which response processing completed.
// ReceivedAt 返回响应处理完成的时间；尚未记录时为零值。
func (r *Response) ReceivedAt() time.Time {
	return r.receivedAt
}

func (r *Response) setReceivedAt() {
	r.receivedAt = time.Now()
	if r.Request.trace != nil {
		r.Request.trace.update(func() {
			r.Request.trace.endTime = r.receivedAt
		})
	}
}

// UnmarshalJson unmarshalls JSON response body into the specified object.
// UnmarshalJson 使用 Client 配置的 JSON 解码器将响应体解析到 v。
func (r *Response) UnmarshalJson(v any) error {
	if r.Err != nil {
		return r.Err
	}
	b, err := r.ToBytes()
	if err != nil {
		return err
	}
	return r.Request.client.jsonUnmarshal(b, v)
}

// UnmarshalXml unmarshalls XML response body into the specified object.
// UnmarshalXml 使用 Client 配置的 XML 解码器将响应体解析到 v。
func (r *Response) UnmarshalXml(v any) error {
	if r.Err != nil {
		return r.Err
	}
	b, err := r.ToBytes()
	if err != nil {
		return err
	}
	return r.Request.client.xmlUnmarshal(b, v)
}

// Unmarshal unmarshalls response body into the specified object according
// to response `Content-Type`.
// Unmarshal 根据响应 Content-Type 将响应体解析到 v；非 XML 类型默认按 JSON 处理，
// ErrorState 响应会返回包含 HTTP 状态的错误。
func (r *Response) Unmarshal(v any) error {
	if r.Err != nil {
		return r.Err
	}
	if r.IsErrorState() {
		return errors.New(r.Status)
	}
	v = util.GetPointer(v)
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "json") {
		return r.UnmarshalJson(v)
	} else if strings.Contains(contentType, "xml") {
		return r.UnmarshalXml(v)
	}
	return r.UnmarshalJson(v)
}

// Into unmarshalls response body into the specified object according
// to response `Content-Type`.
// Into 是 Unmarshal 的别名，根据响应 Content-Type 将响应体解析到 v。
func (r *Response) Into(v any) error {
	return r.Unmarshal(v)
}

// SetBody replaces the cached response body with body.
// SetBody 使用 body 替换缓存的响应体，不会改写底层 http.Response.Body。
func (r *Response) SetBody(body []byte) {
	r.body = body
}

// SetBodyString replaces the cached response body with the bytes of body.
// SetBodyString 使用 body 的字节内容替换缓存的响应体，不会改写底层 http.Response.Body。
func (r *Response) SetBodyString(body string) {
	r.body = []byte(body)
}

// Bytes returns the cached response body without reading from the underlying body.
// It can be nil when the body has not been read. The body is normally cached when:
//  1. Request.SetSuccessResult or Request.SetErrorResult is called.
//  2. `Client.DisableAutoReadResponse` and `Request.DisableAutoReadResponse` is not
//     called, and also `Request.SetOutput` and `Request.SetOutputFile` is not called.
// Bytes 返回已缓存的响应体且不会主动读取底层 body；尚未读取时可能为 nil。
func (r *Response) Bytes() []byte {
	return r.body
}

// String returns the cached response body as a string without reading the underlying body.
// It is empty when the body is empty or has not been read. The body is normally cached when:
//  1. Request.SetSuccessResult or Request.SetErrorResult is called.
//  2. `Client.DisableAutoReadResponse` and `Request.DisableAutoReadResponse` is not
//     called, and also `Request.SetOutput` and `Request.SetOutputFile` is not called.
// String 以字符串返回已缓存的响应体且不会主动读取底层 body；未读取或为空时返回空字符串。
func (r *Response) String() string {
	return string(r.body)
}

// ToString returns the response body as a string, reading and caching it when necessary.
// ToString 以字符串返回响应体；必要时会读取、关闭并缓存底层 body。
func (r *Response) ToString() (string, error) {
	b, err := r.ToBytes()
	return string(b), err
}

// ToBytes returns the response body, reading, closing, transforming, and caching it when necessary.
// ToBytes 返回响应体；必要时会读取并关闭底层 body、执行已配置的转换器并缓存结果。
func (r *Response) ToBytes() (body []byte, err error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.body != nil {
		return r.body, nil
	}
	if r.Response == nil || r.Response.Body == nil {
		return []byte{}, nil
	}
	defer func() {
		r.Body.Close()
		if err != nil {
			r.Err = err
		}
		r.body = body
	}()
	contentLength := r.ContentLength
	if r.Body == http.NoBody || r.Request != nil && r.Request.Method == http.MethodHead {
		// A HEAD response may advertise the resource size while carrying no body.
		// Keep reading through the normal path so transformers still run, but do
		// not use that advertised size as an allocation hint.
		//
		// HEAD 可保留资源大小却不携带响应体。仍走正常读取流程以执行转换器，
		// 但不把该声明长度用于预分配。
		contentLength = -1
	}
	body, err = readResponseBody(r.Body, contentLength)
	r.setReceivedAt()
	if err == nil && r.Request.client.responseBodyTransformer != nil {
		body, err = r.Request.client.responseBodyTransformer(body, r.Request, r)
	}
	return
}

// readResponseBody reads body to EOF. A small positive contentLength is used
// only as a bounded capacity hint, never as a read limit.
//
// readResponseBody 始终读取到 EOF；较小的正 Content-Length 仅作为有上限的容量提示。
func readResponseBody(body io.Reader, contentLength int64) ([]byte, error) {
	if contentLength <= 0 || contentLength > maxResponseBodyPreallocateSize {
		return io.ReadAll(body)
	}

	// bytes.Buffer.ReadFrom asks for at least bytes.MinRead spare bytes before
	// each read. Reserve that documented standard-library tail so an exact
	// Content-Length response can observe EOF without an extra growth step.
	//
	// 按 bytes.Buffer.ReadFrom 的官方约定额外预留 bytes.MinRead，避免读取 EOF
	// 前再扩容一次；Content-Length 只作为容量提示，不作为读取边界。
	buf := bytes.NewBuffer(make([]byte, 0, int(contentLength)+bytes.MinRead))
	_, err := buf.ReadFrom(body)
	return buf.Bytes(), err
}

// Dump returns the captured request and response dump.
// Request.EnableDump or another request-level dump method must have been called first.
// Dump 返回捕获的请求与响应 dump；调用前必须启用对应的请求级 dump。
func (r *Response) Dump() string {
	return r.Request.getDumpBuffer().String()
}

// GetStatus returns the response status.
// GetStatus 返回响应状态文本；没有底层响应时返回空字符串。
func (r *Response) GetStatus() string {
	if r.Response == nil {
		return ""
	}
	return r.Status
}

// GetStatusCode returns the response status code.
// GetStatusCode 返回响应状态码；没有底层响应时返回 0。
func (r *Response) GetStatusCode() int {
	if r.Response == nil {
		return 0
	}
	return r.StatusCode
}

// GetHeader returns the response header value by key.
// GetHeader 返回指定响应 Header 的首个值；没有底层响应时返回空字符串。
func (r *Response) GetHeader(key string) string {
	if r.Response == nil {
		return ""
	}
	return r.Header.Get(key)
}

// GetHeaderValues returns the response header values by key.
// GetHeaderValues 返回指定响应 Header 的全部值；没有底层响应时返回 nil。
func (r *Response) GetHeaderValues(key string) []string {
	if r.Response == nil {
		return nil
	}
	return r.Header.Values(key)
}

// HeaderToString serializes all response headers as an HTTP header block.
// HeaderToString 将全部响应 Header 序列化为 HTTP Header 文本；没有底层响应时返回空字符串。
func (r *Response) HeaderToString() string {
	if r.Response == nil {
		return ""
	}
	return convertHeaderToString(r.Header)
}
