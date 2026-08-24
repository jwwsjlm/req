package req

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SetURL creates a Request from the package-level default Client and calls Request.SetURL.
//
// SetURL 使用包级默认 Client 创建 Request 并调用其 SetURL 方法。
func SetURL(url string) *Request {
	return defaultClient.R().SetURL(url)
}

// SetFormDataFromValues creates a Request from the package-level default Client and calls Request.SetFormDataFromValues.
//
// SetFormDataFromValues 使用包级默认 Client 创建 Request 并调用其 SetFormDataFromValues 方法。
func SetFormDataFromValues(data url.Values) *Request {
	return defaultClient.R().SetFormDataFromValues(data)
}

// SetFormData creates a Request from the package-level default Client and calls Request.SetFormData.
//
// SetFormData 使用包级默认 Client 创建 Request 并调用其 SetFormData 方法。
func SetFormData(data map[string]string) *Request {
	return defaultClient.R().SetFormData(data)
}

// SetOrderedFormData creates a Request from the package-level default Client and calls Request.SetOrderedFormData.
//
// SetOrderedFormData 使用包级默认 Client 创建 Request 并调用其 SetOrderedFormData 方法。
func SetOrderedFormData(kvs ...string) *Request {
	return defaultClient.R().SetOrderedFormData(kvs...)
}

// SetFormDataAnyType creates a Request from the package-level default Client and calls Request.SetFormDataAnyType.
//
// SetFormDataAnyType 使用包级默认 Client 创建 Request 并调用其 SetFormDataAnyType 方法。
func SetFormDataAnyType(data map[string]any) *Request {
	return defaultClient.R().SetFormDataAnyType(data)
}

// SetCookies creates a Request from the package-level default Client and calls Request.SetCookies.
//
// SetCookies 使用包级默认 Client 创建 Request 并调用其 SetCookies 方法。
func SetCookies(cookies ...*http.Cookie) *Request {
	return defaultClient.R().SetCookies(cookies...)
}

// SetQueryString creates a Request from the package-level default Client and calls Request.SetQueryString.
//
// SetQueryString 使用包级默认 Client 创建 Request 并调用其 SetQueryString 方法。
func SetQueryString(query string) *Request {
	return defaultClient.R().SetQueryString(query)
}

// SetQueryParamsFromValues creates a Request from the package-level default Client and calls Request.SetQueryParamsFromValues.
//
// SetQueryParamsFromValues 使用包级默认 Client 创建 Request 并调用其 SetQueryParamsFromValues 方法。
func SetQueryParamsFromValues(params url.Values) *Request {
	return defaultClient.R().SetQueryParamsFromValues(params)
}

// SetQueryParamsFromStruct creates a Request from the package-level default Client and calls Request.SetQueryParamsFromStruct.
//
// SetQueryParamsFromStruct 使用包级默认 Client 创建 Request 并调用其 SetQueryParamsFromStruct 方法。
func SetQueryParamsFromStruct(v any) *Request {
	return defaultClient.R().SetQueryParamsFromStruct(v)
}

// SetFileReader creates a Request from the package-level default Client and calls Request.SetFileReader.
//
// SetFileReader 使用包级默认 Client 创建 Request 并调用其 SetFileReader 方法。
func SetFileReader(paramName, filePath string, reader io.Reader) *Request {
	return defaultClient.R().SetFileReader(paramName, filePath, reader)
}

// SetFileBytes creates a Request from the package-level default Client and calls Request.SetFileBytes.
//
// SetFileBytes 使用包级默认 Client 创建 Request 并调用其 SetFileBytes 方法。
func SetFileBytes(paramName, filename string, content []byte) *Request {
	return defaultClient.R().SetFileBytes(paramName, filename, content)
}

// SetFiles creates a Request from the package-level default Client and calls Request.SetFiles.
//
// SetFiles 使用包级默认 Client 创建 Request 并调用其 SetFiles 方法。
func SetFiles(files map[string]string) *Request {
	return defaultClient.R().SetFiles(files)
}

// SetFile creates a Request from the package-level default Client and calls Request.SetFile.
//
// SetFile 使用包级默认 Client 创建 Request 并调用其 SetFile 方法。
func SetFile(paramName, filePath string) *Request {
	return defaultClient.R().SetFile(paramName, filePath)
}

// SetFileUpload creates a Request from the package-level default Client and calls Request.SetFileUpload.
//
// SetFileUpload 使用包级默认 Client 创建 Request 并调用其 SetFileUpload 方法。
func SetFileUpload(f ...FileUpload) *Request {
	return defaultClient.R().SetFileUpload(f...)
}

// SetResult creates a Request from the package-level default Client and calls Request.SetSuccessResult.
//
// SetResult 使用包级默认 Client 创建 Request 并调用其 SetSuccessResult 方法。
func SetResult(result any) *Request {
	return defaultClient.R().SetSuccessResult(result)
}

// SetSuccessResult creates a Request from the package-level default Client and calls Request.SetSuccessResult.
//
// SetSuccessResult 使用包级默认 Client 创建 Request 并调用其 SetSuccessResult 方法。
func SetSuccessResult(result any) *Request {
	return defaultClient.R().SetSuccessResult(result)
}

// SetError creates a Request from the package-level default Client and calls Request.SetErrorResult.
//
// SetError 使用包级默认 Client 创建 Request 并调用其 SetErrorResult 方法。
func SetError(error any) *Request {
	return defaultClient.R().SetErrorResult(error)
}

// SetErrorResult creates a Request from the package-level default Client and calls Request.SetErrorResult.
//
// SetErrorResult 使用包级默认 Client 创建 Request 并调用其 SetErrorResult 方法。
func SetErrorResult(error any) *Request {
	return defaultClient.R().SetErrorResult(error)
}

// SetBearerAuthToken creates a Request from the package-level default Client and calls Request.SetBearerAuthToken.
//
// SetBearerAuthToken 使用包级默认 Client 创建 Request 并调用其 SetBearerAuthToken 方法。
func SetBearerAuthToken(token string) *Request {
	return defaultClient.R().SetBearerAuthToken(token)
}

// SetBasicAuth creates a Request from the package-level default Client and calls Request.SetBasicAuth.
//
// SetBasicAuth 使用包级默认 Client 创建 Request 并调用其 SetBasicAuth 方法。
func SetBasicAuth(username, password string) *Request {
	return defaultClient.R().SetBasicAuth(username, password)
}

// SetDigestAuth creates a Request from the package-level default Client and calls Request.SetDigestAuth.
//
// SetDigestAuth 使用包级默认 Client 创建 Request 并调用其 SetDigestAuth 方法。
func SetDigestAuth(username, password string) *Request {
	return defaultClient.R().SetDigestAuth(username, password)
}

// SetHeaders creates a Request from the package-level default Client and calls Request.SetHeaders.
//
// SetHeaders 使用包级默认 Client 创建 Request 并调用其 SetHeaders 方法。
func SetHeaders(hdrs map[string]string) *Request {
	return defaultClient.R().SetHeaders(hdrs)
}

// SetHeader creates a Request from the package-level default Client and calls Request.SetHeader.
//
// SetHeader 使用包级默认 Client 创建 Request 并调用其 SetHeader 方法。
func SetHeader(key, value string) *Request {
	return defaultClient.R().SetHeader(key, value)
}

// SetHeaderOrder creates a Request from the package-level default Client and calls Request.SetHeaderOrder.
//
// SetHeaderOrder 使用包级默认 Client 创建 Request 并调用其 SetHeaderOrder 方法。
func SetHeaderOrder(keys ...string) *Request {
	return defaultClient.R().SetHeaderOrder(keys...)
}

// SetPseudoHeaderOrder creates a Request from the package-level default Client and calls Request.SetPseudoHeaderOrder.
//
// SetPseudoHeaderOrder 使用包级默认 Client 创建 Request 并调用其 SetPseudoHeaderOrder 方法。
func SetPseudoHeaderOrder(keys ...string) *Request {
	return defaultClient.R().SetPseudoHeaderOrder(keys...)
}

// SetOutputFile creates a Request from the package-level default Client and calls Request.SetOutputFile.
//
// SetOutputFile 使用包级默认 Client 创建 Request 并调用其 SetOutputFile 方法。
func SetOutputFile(file string) *Request {
	return defaultClient.R().SetOutputFile(file)
}

// SetOutput creates a Request from the package-level default Client and calls Request.SetOutput.
//
// SetOutput 使用包级默认 Client 创建 Request 并调用其 SetOutput 方法。
func SetOutput(output io.Writer) *Request {
	return defaultClient.R().SetOutput(output)
}

// SetQueryParams creates a Request from the package-level default Client and calls Request.SetQueryParams.
//
// SetQueryParams 使用包级默认 Client 创建 Request 并调用其 SetQueryParams 方法。
func SetQueryParams(params map[string]string) *Request {
	return defaultClient.R().SetQueryParams(params)
}

// SetQueryParamsAnyType creates a Request from the package-level default Client and calls Request.SetQueryParamsAnyType.
//
// SetQueryParamsAnyType 使用包级默认 Client 创建 Request 并调用其 SetQueryParamsAnyType 方法。
func SetQueryParamsAnyType(params map[string]any) *Request {
	return defaultClient.R().SetQueryParamsAnyType(params)
}

// SetQueryParam creates a Request from the package-level default Client and calls Request.SetQueryParam.
//
// SetQueryParam 使用包级默认 Client 创建 Request 并调用其 SetQueryParam 方法。
func SetQueryParam(key, value string) *Request {
	return defaultClient.R().SetQueryParam(key, value)
}

// AddQueryParam creates a Request from the package-level default Client and calls Request.AddQueryParam.
//
// AddQueryParam 使用包级默认 Client 创建 Request 并调用其 AddQueryParam 方法。
func AddQueryParam(key, value string) *Request {
	return defaultClient.R().AddQueryParam(key, value)
}

// AddQueryParams creates a Request from the package-level default Client and calls Request.AddQueryParams.
//
// AddQueryParams 使用包级默认 Client 创建 Request 并调用其 AddQueryParams 方法。
func AddQueryParams(key string, values ...string) *Request {
	return defaultClient.R().AddQueryParams(key, values...)
}

// SetPathParams creates a Request from the package-level default Client and calls Request.SetPathParams.
//
// SetPathParams 使用包级默认 Client 创建 Request 并调用其 SetPathParams 方法。
func SetPathParams(params map[string]string) *Request {
	return defaultClient.R().SetPathParams(params)
}

// SetPathParam creates a Request from the package-level default Client and calls Request.SetPathParam.
//
// SetPathParam 使用包级默认 Client 创建 Request 并调用其 SetPathParam 方法。
func SetPathParam(key, value string) *Request {
	return defaultClient.R().SetPathParam(key, value)
}

// MustGet creates a Request from the package-level default Client and calls Request.MustGet.
//
// MustGet 使用包级默认 Client 创建 Request 并调用其 MustGet 方法。
func MustGet(url string) *Response {
	return defaultClient.R().MustGet(url)
}

// Get creates a Request from the package-level default Client and calls Request.Get.
//
// Get 使用包级默认 Client 创建 Request 并调用其 Get 方法。
func Get(url string) (*Response, error) {
	return defaultClient.R().Get(url)
}

// MustPost creates a Request from the package-level default Client and calls Request.MustPost.
//
// MustPost 使用包级默认 Client 创建 Request 并调用其 MustPost 方法。
func MustPost(url string) *Response {
	return defaultClient.R().MustPost(url)
}

// Post creates a Request from the package-level default Client and calls Request.Post.
//
// Post 使用包级默认 Client 创建 Request 并调用其 Post 方法。
func Post(url string) (*Response, error) {
	return defaultClient.R().Post(url)
}

// MustPut creates a Request from the package-level default Client and calls Request.MustPut.
//
// MustPut 使用包级默认 Client 创建 Request 并调用其 MustPut 方法。
func MustPut(url string) *Response {
	return defaultClient.R().MustPut(url)
}

// Put creates a Request from the package-level default Client and calls Request.Put.
//
// Put 使用包级默认 Client 创建 Request 并调用其 Put 方法。
func Put(url string) (*Response, error) {
	return defaultClient.R().Put(url)
}

// MustPatch creates a Request from the package-level default Client and calls Request.MustPatch.
//
// MustPatch 使用包级默认 Client 创建 Request 并调用其 MustPatch 方法。
func MustPatch(url string) *Response {
	return defaultClient.R().MustPatch(url)
}

// Patch creates a Request from the package-level default Client and calls Request.Patch.
//
// Patch 使用包级默认 Client 创建 Request 并调用其 Patch 方法。
func Patch(url string) (*Response, error) {
	return defaultClient.R().Patch(url)
}

// MustDelete creates a Request from the package-level default Client and calls Request.MustDelete.
//
// MustDelete 使用包级默认 Client 创建 Request 并调用其 MustDelete 方法。
func MustDelete(url string) *Response {
	return defaultClient.R().MustDelete(url)
}

// Delete creates a Request from the package-level default Client and calls Request.Delete.
//
// Delete 使用包级默认 Client 创建 Request 并调用其 Delete 方法。
func Delete(url string) (*Response, error) {
	return defaultClient.R().Delete(url)
}

// MustOptions creates a Request from the package-level default Client and calls Request.MustOptions.
//
// MustOptions 使用包级默认 Client 创建 Request 并调用其 MustOptions 方法。
func MustOptions(url string) *Response {
	return defaultClient.R().MustOptions(url)
}

// Options creates a Request from the package-level default Client and calls Request.Options.
//
// Options 使用包级默认 Client 创建 Request 并调用其 Options 方法。
func Options(url string) (*Response, error) {
	return defaultClient.R().Options(url)
}

// MustHead creates a Request from the package-level default Client and calls Request.MustHead.
//
// MustHead 使用包级默认 Client 创建 Request 并调用其 MustHead 方法。
func MustHead(url string) *Response {
	return defaultClient.R().MustHead(url)
}

// Head creates a Request from the package-level default Client and calls Request.Head.
//
// Head 使用包级默认 Client 创建 Request 并调用其 Head 方法。
func Head(url string) (*Response, error) {
	return defaultClient.R().Head(url)
}

// SetBody creates a Request from the package-level default Client and calls Request.SetBody.
//
// SetBody 使用包级默认 Client 创建 Request 并调用其 SetBody 方法。
func SetBody(body any) *Request {
	return defaultClient.R().SetBody(body)
}

// SetBodyBytes creates a Request from the package-level default Client and calls Request.SetBodyBytes.
//
// SetBodyBytes 使用包级默认 Client 创建 Request 并调用其 SetBodyBytes 方法。
func SetBodyBytes(body []byte) *Request {
	return defaultClient.R().SetBodyBytes(body)
}

// SetBodyString creates a Request from the package-level default Client and calls Request.SetBodyString.
//
// SetBodyString 使用包级默认 Client 创建 Request 并调用其 SetBodyString 方法。
func SetBodyString(body string) *Request {
	return defaultClient.R().SetBodyString(body)
}

// SetBodyJsonString creates a Request from the package-level default Client and calls Request.SetBodyJsonString.
//
// SetBodyJsonString 使用包级默认 Client 创建 Request 并调用其 SetBodyJsonString 方法。
func SetBodyJsonString(body string) *Request {
	return defaultClient.R().SetBodyJsonString(body)
}

// SetBodyJsonBytes creates a Request from the package-level default Client and calls Request.SetBodyJsonBytes.
//
// SetBodyJsonBytes 使用包级默认 Client 创建 Request 并调用其 SetBodyJsonBytes 方法。
func SetBodyJsonBytes(body []byte) *Request {
	return defaultClient.R().SetBodyJsonBytes(body)
}

// SetBodyJsonMarshal creates a Request from the package-level default Client and calls Request.SetBodyJsonMarshal.
//
// SetBodyJsonMarshal 使用包级默认 Client 创建 Request 并调用其 SetBodyJsonMarshal 方法。
func SetBodyJsonMarshal(v any) *Request {
	return defaultClient.R().SetBodyJsonMarshal(v)
}

// SetBodyXmlString creates a Request from the package-level default Client and calls Request.SetBodyXmlString.
//
// SetBodyXmlString 使用包级默认 Client 创建 Request 并调用其 SetBodyXmlString 方法。
func SetBodyXmlString(body string) *Request {
	return defaultClient.R().SetBodyXmlString(body)
}

// SetBodyXmlBytes creates a Request from the package-level default Client and calls Request.SetBodyXmlBytes.
//
// SetBodyXmlBytes 使用包级默认 Client 创建 Request 并调用其 SetBodyXmlBytes 方法。
func SetBodyXmlBytes(body []byte) *Request {
	return defaultClient.R().SetBodyXmlBytes(body)
}

// SetBodyXmlMarshal creates a Request from the package-level default Client and calls Request.SetBodyXmlMarshal.
//
// SetBodyXmlMarshal 使用包级默认 Client 创建 Request 并调用其 SetBodyXmlMarshal 方法。
func SetBodyXmlMarshal(v any) *Request {
	return defaultClient.R().SetBodyXmlMarshal(v)
}

// SetContentType creates a Request from the package-level default Client and calls Request.SetContentType.
//
// SetContentType 使用包级默认 Client 创建 Request 并调用其 SetContentType 方法。
func SetContentType(contentType string) *Request {
	return defaultClient.R().SetContentType(contentType)
}

// SetContext creates a Request from the package-level default Client and calls Request.SetContext.
//
// SetContext 使用包级默认 Client 创建 Request 并调用其 SetContext 方法。
func SetContext(ctx context.Context) *Request {
	return defaultClient.R().SetContext(ctx)
}

// DisableTrace creates a Request from the package-level default Client and calls Request.DisableTrace.
//
// DisableTrace 使用包级默认 Client 创建 Request 并调用其 DisableTrace 方法。
func DisableTrace() *Request {
	return defaultClient.R().DisableTrace()
}

// EnableTrace creates a Request from the package-level default Client and calls Request.EnableTrace.
//
// EnableTrace 使用包级默认 Client 创建 Request 并调用其 EnableTrace 方法。
func EnableTrace() *Request {
	return defaultClient.R().EnableTrace()
}

// EnableForceChunkedEncoding creates a Request from the package-level default Client and calls Request.EnableForceChunkedEncoding.
//
// EnableForceChunkedEncoding 使用包级默认 Client 创建 Request 并调用其 EnableForceChunkedEncoding 方法。
func EnableForceChunkedEncoding() *Request {
	return defaultClient.R().EnableForceChunkedEncoding()
}

// DisableForceChunkedEncoding creates a Request from the package-level default Client and calls Request.DisableForceChunkedEncoding.
//
// DisableForceChunkedEncoding 使用包级默认 Client 创建 Request 并调用其 DisableForceChunkedEncoding 方法。
func DisableForceChunkedEncoding() *Request {
	return defaultClient.R().DisableForceChunkedEncoding()
}

// EnableForceMultipart creates a Request from the package-level default Client and calls Request.EnableForceMultipart.
//
// EnableForceMultipart 使用包级默认 Client 创建 Request 并调用其 EnableForceMultipart 方法。
func EnableForceMultipart() *Request {
	return defaultClient.R().EnableForceMultipart()
}

// DisableForceMultipart creates a Request from the package-level default Client and calls Request.DisableForceMultipart.
//
// DisableForceMultipart 使用包级默认 Client 创建 Request 并调用其 DisableForceMultipart 方法。
func DisableForceMultipart() *Request {
	return defaultClient.R().DisableForceMultipart()
}

// EnableDumpTo creates a Request from the package-level default Client and calls Request.EnableDumpTo.
//
// EnableDumpTo 使用包级默认 Client 创建 Request 并调用其 EnableDumpTo 方法。
func EnableDumpTo(output io.Writer) *Request {
	return defaultClient.R().EnableDumpTo(output)
}

// EnableDumpToFile creates a Request from the package-level default Client and calls Request.EnableDumpToFile.
//
// EnableDumpToFile 使用包级默认 Client 创建 Request 并调用其 EnableDumpToFile 方法。
func EnableDumpToFile(filename string) *Request {
	return defaultClient.R().EnableDumpToFile(filename)
}

// SetDumpOptions creates a Request from the package-level default Client and calls Request.SetDumpOptions.
//
// SetDumpOptions 使用包级默认 Client 创建 Request 并调用其 SetDumpOptions 方法。
func SetDumpOptions(opt *DumpOptions) *Request {
	return defaultClient.R().SetDumpOptions(opt)
}

// EnableDump creates a Request from the package-level default Client and calls Request.EnableDump.
//
// EnableDump 使用包级默认 Client 创建 Request 并调用其 EnableDump 方法。
func EnableDump() *Request {
	return defaultClient.R().EnableDump()
}

// EnableDumpWithoutBody creates a Request from the package-level default Client and calls Request.EnableDumpWithoutBody.
//
// EnableDumpWithoutBody 使用包级默认 Client 创建 Request 并调用其 EnableDumpWithoutBody 方法。
func EnableDumpWithoutBody() *Request {
	return defaultClient.R().EnableDumpWithoutBody()
}

// EnableDumpWithoutHeader creates a Request from the package-level default Client and calls Request.EnableDumpWithoutHeader.
//
// EnableDumpWithoutHeader 使用包级默认 Client 创建 Request 并调用其 EnableDumpWithoutHeader 方法。
func EnableDumpWithoutHeader() *Request {
	return defaultClient.R().EnableDumpWithoutHeader()
}

// EnableDumpWithoutResponse creates a Request from the package-level default Client and calls Request.EnableDumpWithoutResponse.
//
// EnableDumpWithoutResponse 使用包级默认 Client 创建 Request 并调用其 EnableDumpWithoutResponse 方法。
func EnableDumpWithoutResponse() *Request {
	return defaultClient.R().EnableDumpWithoutResponse()
}

// EnableDumpWithoutRequest creates a Request from the package-level default Client and calls Request.EnableDumpWithoutRequest.
//
// EnableDumpWithoutRequest 使用包级默认 Client 创建 Request 并调用其 EnableDumpWithoutRequest 方法。
func EnableDumpWithoutRequest() *Request {
	return defaultClient.R().EnableDumpWithoutRequest()
}

// EnableDumpWithoutRequestBody creates a Request from the package-level default Client and calls Request.EnableDumpWithoutRequestBody.
//
// EnableDumpWithoutRequestBody 使用包级默认 Client 创建 Request 并调用其 EnableDumpWithoutRequestBody 方法。
func EnableDumpWithoutRequestBody() *Request {
	return defaultClient.R().EnableDumpWithoutRequestBody()
}

// EnableDumpWithoutResponseBody creates a Request from the package-level default Client and calls Request.EnableDumpWithoutResponseBody.
//
// EnableDumpWithoutResponseBody 使用包级默认 Client 创建 Request 并调用其 EnableDumpWithoutResponseBody 方法。
func EnableDumpWithoutResponseBody() *Request {
	return defaultClient.R().EnableDumpWithoutResponseBody()
}

// SetRetryCount creates a Request from the package-level default Client and calls Request.SetRetryCount.
//
// SetRetryCount 使用包级默认 Client 创建 Request 并调用其 SetRetryCount 方法。
func SetRetryCount(count int) *Request {
	return defaultClient.R().SetRetryCount(count)
}

// SetRetryInterval creates a Request from the package-level default Client and calls Request.SetRetryInterval.
//
// SetRetryInterval 使用包级默认 Client 创建 Request 并调用其 SetRetryInterval 方法。
func SetRetryInterval(getRetryIntervalFunc GetRetryIntervalFunc) *Request {
	return defaultClient.R().SetRetryInterval(getRetryIntervalFunc)
}

// SetRetryFixedInterval creates a Request from the package-level default Client and calls Request.SetRetryFixedInterval.
//
// SetRetryFixedInterval 使用包级默认 Client 创建 Request 并调用其 SetRetryFixedInterval 方法。
func SetRetryFixedInterval(interval time.Duration) *Request {
	return defaultClient.R().SetRetryFixedInterval(interval)
}

// SetRetryBackoffInterval creates a Request from the package-level default Client and calls Request.SetRetryBackoffInterval.
//
// SetRetryBackoffInterval 使用包级默认 Client 创建 Request 并调用其 SetRetryBackoffInterval 方法。
func SetRetryBackoffInterval(min, max time.Duration) *Request {
	return defaultClient.R().SetRetryBackoffInterval(min, max)
}

// SetRetryHook creates a Request from the package-level default Client and calls Request.SetRetryHook.
//
// SetRetryHook 使用包级默认 Client 创建 Request 并调用其 SetRetryHook 方法。
func SetRetryHook(hook RetryHookFunc) *Request {
	return defaultClient.R().SetRetryHook(hook)
}

// AddRetryHook creates a Request from the package-level default Client and calls Request.AddRetryHook.
//
// AddRetryHook 使用包级默认 Client 创建 Request 并调用其 AddRetryHook 方法。
func AddRetryHook(hook RetryHookFunc) *Request {
	return defaultClient.R().AddRetryHook(hook)
}

// SetRetryCondition creates a Request from the package-level default Client and calls Request.SetRetryCondition.
//
// SetRetryCondition 使用包级默认 Client 创建 Request 并调用其 SetRetryCondition 方法。
func SetRetryCondition(condition RetryConditionFunc) *Request {
	return defaultClient.R().SetRetryCondition(condition)
}

// AddRetryCondition creates a Request from the package-level default Client and calls Request.AddRetryCondition.
//
// AddRetryCondition 使用包级默认 Client 创建 Request 并调用其 AddRetryCondition 方法。
func AddRetryCondition(condition RetryConditionFunc) *Request {
	return defaultClient.R().AddRetryCondition(condition)
}

// SetUploadCallback creates a Request from the package-level default Client and calls Request.SetUploadCallback.
//
// SetUploadCallback 使用包级默认 Client 创建 Request 并调用其 SetUploadCallback 方法。
func SetUploadCallback(callback UploadCallback) *Request {
	return defaultClient.R().SetUploadCallback(callback)
}

// SetUploadCallbackWithInterval creates a Request from the package-level default Client and calls Request.SetUploadCallbackWithInterval.
//
// SetUploadCallbackWithInterval 使用包级默认 Client 创建 Request 并调用其 SetUploadCallbackWithInterval 方法。
func SetUploadCallbackWithInterval(callback UploadCallback, minInterval time.Duration) *Request {
	return defaultClient.R().SetUploadCallbackWithInterval(callback, minInterval)
}

// SetDownloadCallback creates a Request from the package-level default Client and calls Request.SetDownloadCallback.
//
// SetDownloadCallback 使用包级默认 Client 创建 Request 并调用其 SetDownloadCallback 方法。
func SetDownloadCallback(callback DownloadCallback) *Request {
	return defaultClient.R().SetDownloadCallback(callback)
}

// SetDownloadCallbackWithInterval creates a Request from the package-level default Client and calls Request.SetDownloadCallbackWithInterval.
//
// SetDownloadCallbackWithInterval 使用包级默认 Client 创建 Request 并调用其 SetDownloadCallbackWithInterval 方法。
func SetDownloadCallbackWithInterval(callback DownloadCallback, minInterval time.Duration) *Request {
	return defaultClient.R().SetDownloadCallbackWithInterval(callback, minInterval)
}

// EnableCloseConnection creates a Request from the package-level default Client and calls Request.EnableCloseConnection.
//
// EnableCloseConnection 使用包级默认 Client 创建 Request 并调用其 EnableCloseConnection 方法。
func EnableCloseConnection() *Request {
	return defaultClient.R().EnableCloseConnection()
}
