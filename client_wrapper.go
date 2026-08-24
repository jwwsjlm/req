package req

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/jwwsjlm/req/v3/http2"
	"github.com/quic-go/quic-go"
	utls "github.com/refraction-networking/utls"
)

// WrapRoundTrip delegates to WrapRoundTrip on the package-level default Client.
//
// WrapRoundTrip 将调用委托给包级默认 Client 的 WrapRoundTrip 方法。
func WrapRoundTrip(wrappers ...RoundTripWrapper) *Client {
	return defaultClient.WrapRoundTrip(wrappers...)
}

// WrapRoundTripFunc delegates to WrapRoundTripFunc on the package-level default Client.
//
// WrapRoundTripFunc 将调用委托给包级默认 Client 的 WrapRoundTripFunc 方法。
func WrapRoundTripFunc(funcs ...RoundTripWrapperFunc) *Client {
	return defaultClient.WrapRoundTripFunc(funcs...)
}

// SetCommonError delegates to SetCommonErrorResult on the package-level default Client.
//
// SetCommonError 将调用委托给包级默认 Client 的 SetCommonErrorResult 方法。
//
// Deprecated: Use SetCommonErrorResult instead.
func SetCommonError(err any) *Client {
	return defaultClient.SetCommonErrorResult(err)
}

// SetCommonErrorResult delegates to SetCommonErrorResult on the package-level default Client.
//
// SetCommonErrorResult 将调用委托给包级默认 Client 的 SetCommonErrorResult 方法。
func SetCommonErrorResult(err any) *Client {
	return defaultClient.SetCommonErrorResult(err)
}

// SetResultStateCheckFunc delegates to SetResultStateCheckFunc on the package-level default Client.
//
// SetResultStateCheckFunc 将调用委托给包级默认 Client 的 SetResultStateCheckFunc 方法。
func SetResultStateCheckFunc(fn func(resp *Response) ResultState) *Client {
	return defaultClient.SetResultStateCheckFunc(fn)
}

// SetCommonFormDataFromValues delegates to SetCommonFormDataFromValues on the package-level default Client.
//
// SetCommonFormDataFromValues 将调用委托给包级默认 Client 的 SetCommonFormDataFromValues 方法。
func SetCommonFormDataFromValues(data url.Values) *Client {
	return defaultClient.SetCommonFormDataFromValues(data)
}

// SetCommonFormData delegates to SetCommonFormData on the package-level default Client.
//
// SetCommonFormData 将调用委托给包级默认 Client 的 SetCommonFormData 方法。
func SetCommonFormData(data map[string]string) *Client {
	return defaultClient.SetCommonFormData(data)
}

// SetCommonFormDataAnyType delegates to SetCommonFormDataAnyType on the package-level default Client.
//
// SetCommonFormDataAnyType 将调用委托给包级默认 Client 的 SetCommonFormDataAnyType 方法。
func SetCommonFormDataAnyType(data map[string]any) *Client {
	return defaultClient.SetCommonFormDataAnyType(data)
}

// SetCommonFormDataAny delegates to SetCommonFormDataAny on the package-level default Client.
//
// SetCommonFormDataAny 将调用委托给包级默认 Client 的 SetCommonFormDataAny 方法。
func SetCommonFormDataAny(data map[string]any) *Client {
	return defaultClient.SetCommonFormDataAny(data)
}

// SetMultipartBoundaryFunc delegates to SetMultipartBoundaryFunc on the package-level default Client.
//
// SetMultipartBoundaryFunc 将调用委托给包级默认 Client 的 SetMultipartBoundaryFunc 方法。
func SetMultipartBoundaryFunc(fn func() string) *Client {
	return defaultClient.SetMultipartBoundaryFunc(fn)
}

// SetBaseURL delegates to SetBaseURL on the package-level default Client.
//
// SetBaseURL 将调用委托给包级默认 Client 的 SetBaseURL 方法。
func SetBaseURL(u string) *Client {
	return defaultClient.SetBaseURL(u)
}

// SetOutputDirectory delegates to SetOutputDirectory on the package-level default Client.
//
// SetOutputDirectory 将调用委托给包级默认 Client 的 SetOutputDirectory 方法。
func SetOutputDirectory(dir string) *Client {
	return defaultClient.SetOutputDirectory(dir)
}

// SetCertFromFile delegates to SetCertFromFile on the package-level default Client.
//
// SetCertFromFile 将调用委托给包级默认 Client 的 SetCertFromFile 方法。
func SetCertFromFile(certFile, keyFile string) *Client {
	return defaultClient.SetCertFromFile(certFile, keyFile)
}

// SetCerts delegates to SetCerts on the package-level default Client.
//
// SetCerts 将调用委托给包级默认 Client 的 SetCerts 方法。
func SetCerts(certs ...tls.Certificate) *Client {
	return defaultClient.SetCerts(certs...)
}

// SetRootCertFromString delegates to SetRootCertFromString on the package-level default Client.
//
// SetRootCertFromString 将调用委托给包级默认 Client 的 SetRootCertFromString 方法。
func SetRootCertFromString(pemContent string) *Client {
	return defaultClient.SetRootCertFromString(pemContent)
}

// SetRootCertsFromFile delegates to SetRootCertsFromFile on the package-level default Client.
//
// SetRootCertsFromFile 将调用委托给包级默认 Client 的 SetRootCertsFromFile 方法。
func SetRootCertsFromFile(pemFiles ...string) *Client {
	return defaultClient.SetRootCertsFromFile(pemFiles...)
}

// GetTLSClientConfig delegates to GetTLSClientConfig on the package-level default Client.
//
// GetTLSClientConfig 将调用委托给包级默认 Client 的 GetTLSClientConfig 方法。
func GetTLSClientConfig() *tls.Config {
	return defaultClient.GetTLSClientConfig()
}

// SetRedirectPolicy delegates to SetRedirectPolicy on the package-level default Client.
//
// SetRedirectPolicy 将调用委托给包级默认 Client 的 SetRedirectPolicy 方法。
func SetRedirectPolicy(policies ...RedirectPolicy) *Client {
	return defaultClient.SetRedirectPolicy(policies...)
}

// DisableKeepAlives delegates to DisableKeepAlives on the package-level default Client.
//
// DisableKeepAlives 将调用委托给包级默认 Client 的 DisableKeepAlives 方法。
func DisableKeepAlives() *Client {
	return defaultClient.DisableKeepAlives()
}

// EnableKeepAlives delegates to EnableKeepAlives on the package-level default Client.
//
// EnableKeepAlives 将调用委托给包级默认 Client 的 EnableKeepAlives 方法。
func EnableKeepAlives() *Client {
	return defaultClient.EnableKeepAlives()
}

// DisableCompression delegates to DisableCompression on the package-level default Client.
//
// DisableCompression 将调用委托给包级默认 Client 的 DisableCompression 方法。
func DisableCompression() *Client {
	return defaultClient.DisableCompression()
}

// EnableCompression delegates to EnableCompression on the package-level default Client.
//
// EnableCompression 将调用委托给包级默认 Client 的 EnableCompression 方法。
func EnableCompression() *Client {
	return defaultClient.EnableCompression()
}

// SetTLSClientConfig delegates to SetTLSClientConfig on the package-level default Client.
//
// SetTLSClientConfig 将调用委托给包级默认 Client 的 SetTLSClientConfig 方法。
func SetTLSClientConfig(conf *tls.Config) *Client {
	return defaultClient.SetTLSClientConfig(conf)
}

// EnableInsecureSkipVerify delegates to EnableInsecureSkipVerify on the package-level default Client.
//
// EnableInsecureSkipVerify 将调用委托给包级默认 Client 的 EnableInsecureSkipVerify 方法。
func EnableInsecureSkipVerify() *Client {
	return defaultClient.EnableInsecureSkipVerify()
}

// DisableInsecureSkipVerify delegates to DisableInsecureSkipVerify on the package-level default Client.
//
// DisableInsecureSkipVerify 将调用委托给包级默认 Client 的 DisableInsecureSkipVerify 方法。
func DisableInsecureSkipVerify() *Client {
	return defaultClient.DisableInsecureSkipVerify()
}

// SetCommonQueryParams delegates to SetCommonQueryParams on the package-level default Client.
//
// SetCommonQueryParams 将调用委托给包级默认 Client 的 SetCommonQueryParams 方法。
func SetCommonQueryParams(params map[string]string) *Client {
	return defaultClient.SetCommonQueryParams(params)
}

// AddCommonQueryParam delegates to AddCommonQueryParam on the package-level default Client.
//
// AddCommonQueryParam 将调用委托给包级默认 Client 的 AddCommonQueryParam 方法。
func AddCommonQueryParam(key, value string) *Client {
	return defaultClient.AddCommonQueryParam(key, value)
}

// AddCommonQueryParams delegates to AddCommonQueryParams on the package-level default Client.
//
// AddCommonQueryParams 将调用委托给包级默认 Client 的 AddCommonQueryParams 方法。
func AddCommonQueryParams(key string, values ...string) *Client {
	return defaultClient.AddCommonQueryParams(key, values...)
}

// SetCommonQueryParamAny delegates to SetCommonQueryParamAny on the package-level default Client.
//
// SetCommonQueryParamAny 将调用委托给包级默认 Client 的 SetCommonQueryParamAny 方法。
func SetCommonQueryParamAny(key string, value any) *Client {
	return defaultClient.SetCommonQueryParamAny(key, value)
}

// SetCommonPathParam delegates to SetCommonPathParam on the package-level default Client.
//
// SetCommonPathParam 将调用委托给包级默认 Client 的 SetCommonPathParam 方法。
func SetCommonPathParam(key, value string) *Client {
	return defaultClient.SetCommonPathParam(key, value)
}

// SetCommonPathParamAny delegates to SetCommonPathParamAny on the package-level default Client.
//
// SetCommonPathParamAny 将调用委托给包级默认 Client 的 SetCommonPathParamAny 方法。
func SetCommonPathParamAny(key string, value any) *Client {
	return defaultClient.SetCommonPathParamAny(key, value)
}

// SetCommonPathParams delegates to SetCommonPathParams on the package-level default Client.
//
// SetCommonPathParams 将调用委托给包级默认 Client 的 SetCommonPathParams 方法。
func SetCommonPathParams(pathParams map[string]string) *Client {
	return defaultClient.SetCommonPathParams(pathParams)
}

// SetCommonPathRawParam delegates to SetCommonPathRawParam on the package-level default Client.
//
// SetCommonPathRawParam 将调用委托给包级默认 Client 的 SetCommonPathRawParam 方法。
func SetCommonPathRawParam(key, value string) *Client {
	return defaultClient.SetCommonPathRawParam(key, value)
}

// SetCommonPathRawParamAny delegates to SetCommonPathRawParamAny on the package-level default Client.
//
// SetCommonPathRawParamAny 将调用委托给包级默认 Client 的 SetCommonPathRawParamAny 方法。
func SetCommonPathRawParamAny(key string, value any) *Client {
	return defaultClient.SetCommonPathRawParamAny(key, value)
}

// SetCommonPathRawParams delegates to SetCommonPathRawParams on the package-level default Client.
//
// SetCommonPathRawParams 将调用委托给包级默认 Client 的 SetCommonPathRawParams 方法。
func SetCommonPathRawParams(pathParams map[string]string) *Client {
	return defaultClient.SetCommonPathRawParams(pathParams)
}

// SetCommonQueryParam delegates to SetCommonQueryParam on the package-level default Client.
//
// SetCommonQueryParam 将调用委托给包级默认 Client 的 SetCommonQueryParam 方法。
func SetCommonQueryParam(key, value string) *Client {
	return defaultClient.SetCommonQueryParam(key, value)
}

// SetCommonQueryString delegates to SetCommonQueryString on the package-level default Client.
//
// SetCommonQueryString 将调用委托给包级默认 Client 的 SetCommonQueryString 方法。
func SetCommonQueryString(query string) *Client {
	return defaultClient.SetCommonQueryString(query)
}

// SetCommonQueryParamsFromValues delegates to SetCommonQueryParamsFromValues on the package-level default Client.
//
// SetCommonQueryParamsFromValues 将调用委托给包级默认 Client 的 SetCommonQueryParamsFromValues 方法。
func SetCommonQueryParamsFromValues(params url.Values) *Client {
	return defaultClient.SetCommonQueryParamsFromValues(params)
}

// SetCommonQueryParamsFromStruct delegates to SetCommonQueryParamsFromStruct on the package-level default Client.
//
// SetCommonQueryParamsFromStruct 将调用委托给包级默认 Client 的 SetCommonQueryParamsFromStruct 方法。
func SetCommonQueryParamsFromStruct(v any) *Client {
	return defaultClient.SetCommonQueryParamsFromStruct(v)
}

// SetCommonCookies delegates to SetCommonCookies on the package-level default Client.
//
// SetCommonCookies 将调用委托给包级默认 Client 的 SetCommonCookies 方法。
func SetCommonCookies(cookies ...*http.Cookie) *Client {
	return defaultClient.SetCommonCookies(cookies...)
}

// DisableDebugLog delegates to DisableDebugLog on the package-level default Client.
//
// DisableDebugLog 将调用委托给包级默认 Client 的 DisableDebugLog 方法。
func DisableDebugLog() *Client {
	return defaultClient.DisableDebugLog()
}

// EnableDebugLog delegates to EnableDebugLog on the package-level default Client.
//
// EnableDebugLog 将调用委托给包级默认 Client 的 EnableDebugLog 方法。
func EnableDebugLog() *Client {
	return defaultClient.EnableDebugLog()
}

// DevMode delegates to DevMode on the package-level default Client.
//
// DevMode 将调用委托给包级默认 Client 的 DevMode 方法。
func DevMode() *Client {
	return defaultClient.DevMode()
}

// SetScheme delegates to SetScheme on the package-level default Client.
//
// SetScheme 将调用委托给包级默认 Client 的 SetScheme 方法。
func SetScheme(scheme string) *Client {
	return defaultClient.SetScheme(scheme)
}

// SetLogger delegates to SetLogger on the package-level default Client.
//
// SetLogger 将调用委托给包级默认 Client 的 SetLogger 方法。
func SetLogger(log Logger) *Client {
	return defaultClient.SetLogger(log)
}

// SetTimeout delegates to SetTimeout on the package-level default Client.
//
// SetTimeout 将调用委托给包级默认 Client 的 SetTimeout 方法。
func SetTimeout(d time.Duration) *Client {
	return defaultClient.SetTimeout(d)
}

// EnableDumpAll delegates to EnableDumpAll on the package-level default Client.
//
// EnableDumpAll 将调用委托给包级默认 Client 的 EnableDumpAll 方法。
func EnableDumpAll() *Client {
	return defaultClient.EnableDumpAll()
}

// EnableDumpAllToFile delegates to EnableDumpAllToFile on the package-level default Client.
//
// EnableDumpAllToFile 将调用委托给包级默认 Client 的 EnableDumpAllToFile 方法。
func EnableDumpAllToFile(filename string) *Client {
	return defaultClient.EnableDumpAllToFile(filename)
}

// EnableDumpAllTo delegates to EnableDumpAllTo on the package-level default Client.
//
// EnableDumpAllTo 将调用委托给包级默认 Client 的 EnableDumpAllTo 方法。
func EnableDumpAllTo(output io.Writer) *Client {
	return defaultClient.EnableDumpAllTo(output)
}

// EnableDumpAllAsync delegates to EnableDumpAllAsync on the package-level default Client.
//
// EnableDumpAllAsync 将调用委托给包级默认 Client 的 EnableDumpAllAsync 方法。
func EnableDumpAllAsync() *Client {
	return defaultClient.EnableDumpAllAsync()
}

// EnableDumpAllWithoutRequestBody delegates to EnableDumpAllWithoutRequestBody on the package-level default Client.
//
// EnableDumpAllWithoutRequestBody 将调用委托给包级默认 Client 的 EnableDumpAllWithoutRequestBody 方法。
func EnableDumpAllWithoutRequestBody() *Client {
	return defaultClient.EnableDumpAllWithoutRequestBody()
}

// EnableDumpAllWithoutResponseBody delegates to EnableDumpAllWithoutResponseBody on the package-level default Client.
//
// EnableDumpAllWithoutResponseBody 将调用委托给包级默认 Client 的 EnableDumpAllWithoutResponseBody 方法。
func EnableDumpAllWithoutResponseBody() *Client {
	return defaultClient.EnableDumpAllWithoutResponseBody()
}

// EnableDumpAllWithoutResponse delegates to EnableDumpAllWithoutResponse on the package-level default Client.
//
// EnableDumpAllWithoutResponse 将调用委托给包级默认 Client 的 EnableDumpAllWithoutResponse 方法。
func EnableDumpAllWithoutResponse() *Client {
	return defaultClient.EnableDumpAllWithoutResponse()
}

// EnableDumpAllWithoutRequest delegates to EnableDumpAllWithoutRequest on the package-level default Client.
//
// EnableDumpAllWithoutRequest 将调用委托给包级默认 Client 的 EnableDumpAllWithoutRequest 方法。
func EnableDumpAllWithoutRequest() *Client {
	return defaultClient.EnableDumpAllWithoutRequest()
}

// EnableDumpAllWithoutHeader delegates to EnableDumpAllWithoutHeader on the package-level default Client.
//
// EnableDumpAllWithoutHeader 将调用委托给包级默认 Client 的 EnableDumpAllWithoutHeader 方法。
func EnableDumpAllWithoutHeader() *Client {
	return defaultClient.EnableDumpAllWithoutHeader()
}

// EnableDumpAllWithoutBody delegates to EnableDumpAllWithoutBody on the package-level default Client.
//
// EnableDumpAllWithoutBody 将调用委托给包级默认 Client 的 EnableDumpAllWithoutBody 方法。
func EnableDumpAllWithoutBody() *Client {
	return defaultClient.EnableDumpAllWithoutBody()
}

// EnableDumpEachRequest delegates to EnableDumpEachRequest on the package-level default Client.
//
// EnableDumpEachRequest 将调用委托给包级默认 Client 的 EnableDumpEachRequest 方法。
func EnableDumpEachRequest() *Client {
	return defaultClient.EnableDumpEachRequest()
}

// EnableDumpEachRequestWithoutBody delegates to EnableDumpEachRequestWithoutBody on the package-level default Client.
//
// EnableDumpEachRequestWithoutBody 将调用委托给包级默认 Client 的 EnableDumpEachRequestWithoutBody 方法。
func EnableDumpEachRequestWithoutBody() *Client {
	return defaultClient.EnableDumpEachRequestWithoutBody()
}

// EnableDumpEachRequestWithoutHeader delegates to EnableDumpEachRequestWithoutHeader on the package-level default Client.
//
// EnableDumpEachRequestWithoutHeader 将调用委托给包级默认 Client 的 EnableDumpEachRequestWithoutHeader 方法。
func EnableDumpEachRequestWithoutHeader() *Client {
	return defaultClient.EnableDumpEachRequestWithoutHeader()
}

// EnableDumpEachRequestWithoutResponse delegates to EnableDumpEachRequestWithoutResponse on the package-level default Client.
//
// EnableDumpEachRequestWithoutResponse 将调用委托给包级默认 Client 的 EnableDumpEachRequestWithoutResponse 方法。
func EnableDumpEachRequestWithoutResponse() *Client {
	return defaultClient.EnableDumpEachRequestWithoutResponse()
}

// EnableDumpEachRequestWithoutRequest delegates to EnableDumpEachRequestWithoutRequest on the package-level default Client.
//
// EnableDumpEachRequestWithoutRequest 将调用委托给包级默认 Client 的 EnableDumpEachRequestWithoutRequest 方法。
func EnableDumpEachRequestWithoutRequest() *Client {
	return defaultClient.EnableDumpEachRequestWithoutRequest()
}

// EnableDumpEachRequestWithoutResponseBody delegates to EnableDumpEachRequestWithoutResponseBody on the package-level default Client.
//
// EnableDumpEachRequestWithoutResponseBody 将调用委托给包级默认 Client 的 EnableDumpEachRequestWithoutResponseBody 方法。
func EnableDumpEachRequestWithoutResponseBody() *Client {
	return defaultClient.EnableDumpEachRequestWithoutResponseBody()
}

// EnableDumpEachRequestWithoutRequestBody delegates to EnableDumpEachRequestWithoutRequestBody on the package-level default Client.
//
// EnableDumpEachRequestWithoutRequestBody 将调用委托给包级默认 Client 的 EnableDumpEachRequestWithoutRequestBody 方法。
func EnableDumpEachRequestWithoutRequestBody() *Client {
	return defaultClient.EnableDumpEachRequestWithoutRequestBody()
}

// DisableAutoReadResponse delegates to DisableAutoReadResponse on the package-level default Client.
//
// DisableAutoReadResponse 将调用委托给包级默认 Client 的 DisableAutoReadResponse 方法。
func DisableAutoReadResponse() *Client {
	return defaultClient.DisableAutoReadResponse()
}

// EnableAutoReadResponse delegates to EnableAutoReadResponse on the package-level default Client.
//
// EnableAutoReadResponse 将调用委托给包级默认 Client 的 EnableAutoReadResponse 方法。
func EnableAutoReadResponse() *Client {
	return defaultClient.EnableAutoReadResponse()
}

// SetMaxResponseSize delegates to SetMaxResponseSize on the package-level default Client.
//
// SetMaxResponseSize 将调用委托给包级默认 Client 的 SetMaxResponseSize 方法。
func SetMaxResponseSize(max int64) *Client {
	return defaultClient.SetMaxResponseSize(max)
}

// SetAutoDecodeContentType delegates to SetAutoDecodeContentType on the package-level default Client.
//
// SetAutoDecodeContentType 将调用委托给包级默认 Client 的 SetAutoDecodeContentType 方法。
func SetAutoDecodeContentType(contentTypes ...string) *Client {
	return defaultClient.SetAutoDecodeContentType(contentTypes...)
}

// SetAutoDecodeContentTypeFunc delegates to SetAutoDecodeContentTypeFunc on the package-level default Client.
//
// SetAutoDecodeContentTypeFunc 将调用委托给包级默认 Client 的 SetAutoDecodeContentTypeFunc 方法。
func SetAutoDecodeContentTypeFunc(fn func(contentType string) bool) *Client {
	return defaultClient.SetAutoDecodeContentTypeFunc(fn)
}

// SetAutoDecodeAllContentType delegates to SetAutoDecodeAllContentType on the package-level default Client.
//
// SetAutoDecodeAllContentType 将调用委托给包级默认 Client 的 SetAutoDecodeAllContentType 方法。
func SetAutoDecodeAllContentType() *Client {
	return defaultClient.SetAutoDecodeAllContentType()
}

// DisableAutoDecode delegates to DisableAutoDecode on the package-level default Client.
//
// DisableAutoDecode 将调用委托给包级默认 Client 的 DisableAutoDecode 方法。
func DisableAutoDecode() *Client {
	return defaultClient.DisableAutoDecode()
}

// EnableAutoDecode delegates to EnableAutoDecode on the package-level default Client.
//
// EnableAutoDecode 将调用委托给包级默认 Client 的 EnableAutoDecode 方法。
func EnableAutoDecode() *Client {
	return defaultClient.EnableAutoDecode()
}

// SetUserAgent delegates to SetUserAgent on the package-level default Client.
//
// SetUserAgent 将调用委托给包级默认 Client 的 SetUserAgent 方法。
func SetUserAgent(userAgent string) *Client {
	return defaultClient.SetUserAgent(userAgent)
}

// SetCommonBearerAuthToken delegates to SetCommonBearerAuthToken on the package-level default Client.
//
// SetCommonBearerAuthToken 将调用委托给包级默认 Client 的 SetCommonBearerAuthToken 方法。
func SetCommonBearerAuthToken(token string) *Client {
	return defaultClient.SetCommonBearerAuthToken(token)
}

// SetCommonAuthToken delegates to SetCommonAuthToken on the package-level default Client.
//
// SetCommonAuthToken 将调用委托给包级默认 Client 的 SetCommonAuthToken 方法。
func SetCommonAuthToken(token string) *Client {
	return defaultClient.SetCommonAuthToken(token)
}

// SetCommonAuthSchemeToken delegates to SetCommonAuthSchemeToken on the package-level default Client.
//
// SetCommonAuthSchemeToken 将调用委托给包级默认 Client 的 SetCommonAuthSchemeToken 方法。
func SetCommonAuthSchemeToken(scheme, token string) *Client {
	return defaultClient.SetCommonAuthSchemeToken(scheme, token)
}

// SetCommonBasicAuth delegates to SetCommonBasicAuth on the package-level default Client.
//
// SetCommonBasicAuth 将调用委托给包级默认 Client 的 SetCommonBasicAuth 方法。
func SetCommonBasicAuth(username, password string) *Client {
	return defaultClient.SetCommonBasicAuth(username, password)
}

// SetCommonDigestAuth delegates to SetCommonDigestAuth on the package-level default Client.
//
// SetCommonDigestAuth 将调用委托给包级默认 Client 的 SetCommonDigestAuth 方法。
func SetCommonDigestAuth(username, password string) *Client {
	return defaultClient.SetCommonDigestAuth(username, password)
}

// SetCommonHeaders delegates to SetCommonHeaders on the package-level default Client.
//
// SetCommonHeaders 将调用委托给包级默认 Client 的 SetCommonHeaders 方法。
func SetCommonHeaders(hdrs map[string]string) *Client {
	return defaultClient.SetCommonHeaders(hdrs)
}

// SetCommonHeaderAny delegates to SetCommonHeaderAny on the package-level default Client.
//
// SetCommonHeaderAny 将调用委托给包级默认 Client 的 SetCommonHeaderAny 方法。
func SetCommonHeaderAny(key string, value any) *Client {
	return defaultClient.SetCommonHeaderAny(key, value)
}

// SetCommonHeaderValues delegates to SetCommonHeaderValues on the package-level default Client.
//
// SetCommonHeaderValues 将调用委托给包级默认 Client 的 SetCommonHeaderValues 方法。
func SetCommonHeaderValues(key string, values ...string) *Client {
	return defaultClient.SetCommonHeaderValues(key, values...)
}

// SetCommonHeaderMultiValues delegates to SetCommonHeaderMultiValues on the package-level default Client.
//
// SetCommonHeaderMultiValues 将调用委托给包级默认 Client 的 SetCommonHeaderMultiValues 方法。
func SetCommonHeaderMultiValues(hdrs map[string][]string) *Client {
	return defaultClient.SetCommonHeaderMultiValues(hdrs)
}

// SetCommonHeader delegates to SetCommonHeader on the package-level default Client.
//
// SetCommonHeader 将调用委托给包级默认 Client 的 SetCommonHeader 方法。
func SetCommonHeader(key, value string) *Client {
	return defaultClient.SetCommonHeader(key, value)
}

// SetCommonHeaderOrder delegates to SetCommonHeaderOrder on the package-level default Client.
//
// SetCommonHeaderOrder 将调用委托给包级默认 Client 的 SetCommonHeaderOrder 方法。
func SetCommonHeaderOrder(keys ...string) *Client {
	return defaultClient.SetCommonHeaderOrder(keys...)
}

// SetCommonPseudoHeaderOder delegates to SetCommonPseudoHeaderOder on the package-level default Client.
//
// SetCommonPseudoHeaderOder 将调用委托给包级默认 Client 的 SetCommonPseudoHeaderOder 方法。
func SetCommonPseudoHeaderOder(keys ...string) *Client {
	return defaultClient.SetCommonPseudoHeaderOder(keys...)
}

// SetHTTP2SettingsFrame delegates to SetHTTP2SettingsFrame on the package-level default Client.
//
// SetHTTP2SettingsFrame 将调用委托给包级默认 Client 的 SetHTTP2SettingsFrame 方法。
func SetHTTP2SettingsFrame(settings ...http2.Setting) *Client {
	return defaultClient.SetHTTP2SettingsFrame(settings...)
}

// SetHTTP2ConnectionFlow delegates to SetHTTP2ConnectionFlow on the package-level default Client.
//
// SetHTTP2ConnectionFlow 将调用委托给包级默认 Client 的 SetHTTP2ConnectionFlow 方法。
func SetHTTP2ConnectionFlow(flow uint32) *Client {
	return defaultClient.SetHTTP2ConnectionFlow(flow)
}

// SetHTTP2InitialStreamID delegates to SetHTTP2InitialStreamID on the package-level default Client.
//
// SetHTTP2InitialStreamID 将调用委托给包级默认 Client 的 SetHTTP2InitialStreamID 方法。
func SetHTTP2InitialStreamID(id uint32) *Client {
	return defaultClient.SetHTTP2InitialStreamID(id)
}

// SetHTTP2HeaderPriority delegates to SetHTTP2HeaderPriority on the package-level default Client.
//
// SetHTTP2HeaderPriority 将调用委托给包级默认 Client 的 SetHTTP2HeaderPriority 方法。
func SetHTTP2HeaderPriority(priority http2.PriorityParam) *Client {
	return defaultClient.SetHTTP2HeaderPriority(priority)
}

// SetHTTP2PriorityFrames delegates to SetHTTP2PriorityFrames on the package-level default Client.
//
// SetHTTP2PriorityFrames 将调用委托给包级默认 Client 的 SetHTTP2PriorityFrames 方法。
func SetHTTP2PriorityFrames(frames ...http2.PriorityFrame) *Client {
	return defaultClient.SetHTTP2PriorityFrames(frames...)
}

// SetHTTP2MaxHeaderListSize delegates to SetHTTP2MaxHeaderListSize on the package-level default Client.
//
// SetHTTP2MaxHeaderListSize 将调用委托给包级默认 Client 的 SetHTTP2MaxHeaderListSize 方法。
func SetHTTP2MaxHeaderListSize(max uint32) *Client {
	return defaultClient.SetHTTP2MaxHeaderListSize(max)
}

// SetHTTP2StrictMaxConcurrentStreams delegates to SetHTTP2StrictMaxConcurrentStreams on the package-level default Client.
//
// SetHTTP2StrictMaxConcurrentStreams 将调用委托给包级默认 Client 的 SetHTTP2StrictMaxConcurrentStreams 方法。
func SetHTTP2StrictMaxConcurrentStreams(strict bool) *Client {
	return defaultClient.SetHTTP2StrictMaxConcurrentStreams(strict)
}

// SetHTTP2ReadIdleTimeout delegates to SetHTTP2ReadIdleTimeout on the package-level default Client.
//
// SetHTTP2ReadIdleTimeout 将调用委托给包级默认 Client 的 SetHTTP2ReadIdleTimeout 方法。
func SetHTTP2ReadIdleTimeout(timeout time.Duration) *Client {
	return defaultClient.SetHTTP2ReadIdleTimeout(timeout)
}

// SetHTTP2PingTimeout delegates to SetHTTP2PingTimeout on the package-level default Client.
//
// SetHTTP2PingTimeout 将调用委托给包级默认 Client 的 SetHTTP2PingTimeout 方法。
func SetHTTP2PingTimeout(timeout time.Duration) *Client {
	return defaultClient.SetHTTP2PingTimeout(timeout)
}

// SetHTTP2WriteByteTimeout delegates to SetHTTP2WriteByteTimeout on the package-level default Client.
//
// SetHTTP2WriteByteTimeout 将调用委托给包级默认 Client 的 SetHTTP2WriteByteTimeout 方法。
func SetHTTP2WriteByteTimeout(timeout time.Duration) *Client {
	return defaultClient.SetHTTP2WriteByteTimeout(timeout)
}

// ImpersonateChrome delegates to ImpersonateChrome on the package-level default Client.
//
// ImpersonateChrome 将调用委托给包级默认 Client 的 ImpersonateChrome 方法。
func ImpersonateChrome() *Client {
	return defaultClient.ImpersonateChrome()
}

// ImpersonateChromeWithOS delegates to ImpersonateChromeWithOS on the package-level default Client.
//
// ImpersonateChromeWithOS 将调用委托给包级默认 Client 的 ImpersonateChromeWithOS 方法。
func ImpersonateChromeWithOS(os BrowserOS) *Client {
	return defaultClient.ImpersonateChromeWithOS(os)
}

// ImpersonateFirefox delegates to ImpersonateFirefox on the package-level default Client.
//
// ImpersonateFirefox 将调用委托给包级默认 Client 的 ImpersonateFirefox 方法。
func ImpersonateFirefox() *Client {
	return defaultClient.ImpersonateFirefox()
}

// ImpersonateFirefoxWithOS delegates to ImpersonateFirefoxWithOS on the package-level default Client.
//
// ImpersonateFirefoxWithOS 将调用委托给包级默认 Client 的 ImpersonateFirefoxWithOS 方法。
func ImpersonateFirefoxWithOS(os BrowserOS) *Client {
	return defaultClient.ImpersonateFirefoxWithOS(os)
}

// ImpersonateSafari delegates to ImpersonateSafari on the package-level default Client.
//
// ImpersonateSafari 将调用委托给包级默认 Client 的 ImpersonateSafari 方法。
func ImpersonateSafari() *Client {
	return defaultClient.ImpersonateSafari()
}

// SetCommonContentType delegates to SetCommonContentType on the package-level default Client.
//
// SetCommonContentType 将调用委托给包级默认 Client 的 SetCommonContentType 方法。
func SetCommonContentType(ct string) *Client {
	return defaultClient.SetCommonContentType(ct)
}

// DisableDumpAll delegates to DisableDumpAll on the package-level default Client.
//
// DisableDumpAll 将调用委托给包级默认 Client 的 DisableDumpAll 方法。
func DisableDumpAll() *Client {
	return defaultClient.DisableDumpAll()
}

// SetCommonDumpOptions delegates to SetCommonDumpOptions on the package-level default Client.
//
// SetCommonDumpOptions 将调用委托给包级默认 Client 的 SetCommonDumpOptions 方法。
func SetCommonDumpOptions(opt *DumpOptions) *Client {
	return defaultClient.SetCommonDumpOptions(opt)
}

// SetProxy delegates to SetProxy on the package-level default Client.
//
// SetProxy 将调用委托给包级默认 Client 的 SetProxy 方法。
func SetProxy(proxy func(*http.Request) (*url.URL, error)) *Client {
	return defaultClient.SetProxy(proxy)
}

// OnBeforeRequest delegates to OnBeforeRequest on the package-level default Client.
//
// OnBeforeRequest 将调用委托给包级默认 Client 的 OnBeforeRequest 方法。
func OnBeforeRequest(m RequestMiddleware) *Client {
	return defaultClient.OnBeforeRequest(m)
}

// OnAfterResponse delegates to OnAfterResponse on the package-level default Client.
//
// OnAfterResponse 将调用委托给包级默认 Client 的 OnAfterResponse 方法。
func OnAfterResponse(m ResponseMiddleware) *Client {
	return defaultClient.OnAfterResponse(m)
}

// SetProxyURL delegates to SetProxyURL on the package-level default Client.
//
// SetProxyURL 将调用委托给包级默认 Client 的 SetProxyURL 方法。
func SetProxyURL(proxyUrl string) *Client {
	return defaultClient.SetProxyURL(proxyUrl)
}

// DisableTraceAll delegates to DisableTraceAll on the package-level default Client.
//
// DisableTraceAll 将调用委托给包级默认 Client 的 DisableTraceAll 方法。
func DisableTraceAll() *Client {
	return defaultClient.DisableTraceAll()
}

// EnableTraceAll delegates to EnableTraceAll on the package-level default Client.
//
// EnableTraceAll 将调用委托给包级默认 Client 的 EnableTraceAll 方法。
func EnableTraceAll() *Client {
	return defaultClient.EnableTraceAll()
}

// SetCookieJar delegates to SetCookieJar on the package-level default Client.
//
// SetCookieJar 将调用委托给包级默认 Client 的 SetCookieJar 方法。
func SetCookieJar(jar http.CookieJar) *Client {
	return defaultClient.SetCookieJar(jar)
}

// GetCookies delegates to GetCookies on the package-level default Client.
//
// GetCookies 将调用委托给包级默认 Client 的 GetCookies 方法。
func GetCookies(url string) ([]*http.Cookie, error) {
	return defaultClient.GetCookies(url)
}

// ClearCookies delegates to ClearCookies on the package-level default Client.
//
// ClearCookies 将调用委托给包级默认 Client 的 ClearCookies 方法。
func ClearCookies() *Client {
	return defaultClient.ClearCookies()
}

// SetJsonMarshal delegates to SetJsonMarshal on the package-level default Client.
//
// SetJsonMarshal 将调用委托给包级默认 Client 的 SetJsonMarshal 方法。
func SetJsonMarshal(fn func(v any) ([]byte, error)) *Client {
	return defaultClient.SetJsonMarshal(fn)
}

// SetJsonUnmarshal delegates to SetJsonUnmarshal on the package-level default Client.
//
// SetJsonUnmarshal 将调用委托给包级默认 Client 的 SetJsonUnmarshal 方法。
func SetJsonUnmarshal(fn func(data []byte, v any) error) *Client {
	return defaultClient.SetJsonUnmarshal(fn)
}

// SetXmlMarshal delegates to SetXmlMarshal on the package-level default Client.
//
// SetXmlMarshal 将调用委托给包级默认 Client 的 SetXmlMarshal 方法。
func SetXmlMarshal(fn func(v any) ([]byte, error)) *Client {
	return defaultClient.SetXmlMarshal(fn)
}

// SetXmlUnmarshal delegates to SetXmlUnmarshal on the package-level default Client.
//
// SetXmlUnmarshal 将调用委托给包级默认 Client 的 SetXmlUnmarshal 方法。
func SetXmlUnmarshal(fn func(data []byte, v any) error) *Client {
	return defaultClient.SetXmlUnmarshal(fn)
}

// SetDialTLS delegates to SetDialTLS on the package-level default Client.
//
// SetDialTLS 将调用委托给包级默认 Client 的 SetDialTLS 方法。
func SetDialTLS(fn func(ctx context.Context, network, addr string) (net.Conn, error)) *Client {
	return defaultClient.SetDialTLS(fn)
}

// SetDial delegates to SetDial on the package-level default Client.
//
// SetDial 将调用委托给包级默认 Client 的 SetDial 方法。
func SetDial(fn func(ctx context.Context, network, addr string) (net.Conn, error)) *Client {
	return defaultClient.SetDial(fn)
}

// SetTLSHandshakeTimeout delegates to SetTLSHandshakeTimeout on the package-level default Client.
//
// SetTLSHandshakeTimeout 将调用委托给包级默认 Client 的 SetTLSHandshakeTimeout 方法。
func SetTLSHandshakeTimeout(timeout time.Duration) *Client {
	return defaultClient.SetTLSHandshakeTimeout(timeout)
}

// EnableForceHTTP1 delegates to EnableForceHTTP1 on the package-level default Client.
//
// EnableForceHTTP1 将调用委托给包级默认 Client 的 EnableForceHTTP1 方法。
func EnableForceHTTP1() *Client {
	return defaultClient.EnableForceHTTP1()
}

// EnableForceHTTP2 delegates to EnableForceHTTP2 on the package-level default Client.
//
// EnableForceHTTP2 将调用委托给包级默认 Client 的 EnableForceHTTP2 方法。
func EnableForceHTTP2() *Client {
	return defaultClient.EnableForceHTTP2()
}

// EnableForceHTTP3 delegates to EnableForceHTTP3 on the package-level default Client.
//
// EnableForceHTTP3 将调用委托给包级默认 Client 的 EnableForceHTTP3 方法。
func EnableForceHTTP3() *Client {
	return defaultClient.EnableForceHTTP3()
}

// EnableHTTP3 delegates to EnableHTTP3 on the package-level default Client.
//
// EnableHTTP3 将调用委托给包级默认 Client 的 EnableHTTP3 方法。
func EnableHTTP3() *Client {
	return defaultClient.EnableHTTP3()
}

// SetHTTP3AdditionalSettings delegates to SetHTTP3AdditionalSettings on the package-level default Client.
//
// SetHTTP3AdditionalSettings 将调用委托给包级默认 Client 的 SetHTTP3AdditionalSettings 方法。
func SetHTTP3AdditionalSettings(settings map[uint64]uint64) *Client {
	return defaultClient.SetHTTP3AdditionalSettings(settings)
}

// SetHTTP3AdditionalSetting delegates to SetHTTP3AdditionalSetting on the package-level default Client.
//
// SetHTTP3AdditionalSetting 将调用委托给包级默认 Client 的 SetHTTP3AdditionalSetting 方法。
func SetHTTP3AdditionalSetting(id, value uint64) *Client {
	return defaultClient.SetHTTP3AdditionalSetting(id, value)
}

// SetHTTP3Grease delegates to SetHTTP3Grease on the package-level default Client.
//
// SetHTTP3Grease 将调用委托给包级默认 Client 的 SetHTTP3Grease 方法。
func SetHTTP3Grease() *Client {
	return defaultClient.SetHTTP3Grease()
}

// EnableHTTP3Datagrams delegates to EnableHTTP3Datagrams on the package-level default Client.
//
// EnableHTTP3Datagrams 将调用委托给包级默认 Client 的 EnableHTTP3Datagrams 方法。
func EnableHTTP3Datagrams() *Client {
	return defaultClient.EnableHTTP3Datagrams()
}

// DisableHTTP3Datagrams delegates to DisableHTTP3Datagrams on the package-level default Client.
//
// DisableHTTP3Datagrams 将调用委托给包级默认 Client 的 DisableHTTP3Datagrams 方法。
func DisableHTTP3Datagrams() *Client {
	return defaultClient.DisableHTTP3Datagrams()
}

// EnableHTTP3ExtendedConnect delegates to EnableHTTP3ExtendedConnect on the package-level default Client.
//
// EnableHTTP3ExtendedConnect 将调用委托给包级默认 Client 的 EnableHTTP3ExtendedConnect 方法。
func EnableHTTP3ExtendedConnect() *Client {
	return defaultClient.EnableHTTP3ExtendedConnect()
}

// DisableHTTP3ExtendedConnect delegates to DisableHTTP3ExtendedConnect on the package-level default Client.
//
// DisableHTTP3ExtendedConnect 将调用委托给包级默认 Client 的 DisableHTTP3ExtendedConnect 方法。
func DisableHTTP3ExtendedConnect() *Client {
	return defaultClient.DisableHTTP3ExtendedConnect()
}

// SetHTTP3MaxResponseHeaderBytes delegates to SetHTTP3MaxResponseHeaderBytes on the package-level default Client.
//
// SetHTTP3MaxResponseHeaderBytes 将调用委托给包级默认 Client 的 SetHTTP3MaxResponseHeaderBytes 方法。
func SetHTTP3MaxResponseHeaderBytes(max int) *Client {
	return defaultClient.SetHTTP3MaxResponseHeaderBytes(max)
}

// SetHTTP3QUICConfig delegates to SetHTTP3QUICConfig on the package-level default Client.
//
// SetHTTP3QUICConfig 将调用委托给包级默认 Client 的 SetHTTP3QUICConfig 方法。
func SetHTTP3QUICConfig(cfg *quic.Config) *Client {
	return defaultClient.SetHTTP3QUICConfig(cfg)
}

// SetHTTP3QUICPerformanceProfile delegates to SetHTTP3QUICPerformanceProfile on the package-level default Client.
//
// SetHTTP3QUICPerformanceProfile 将调用委托给包级默认 Client 的 SetHTTP3QUICPerformanceProfile 方法。
func SetHTTP3QUICPerformanceProfile() *Client {
	return defaultClient.SetHTTP3QUICPerformanceProfile()
}

// SetHTTP3QUICChromeProfile delegates to SetHTTP3QUICChromeProfile on the package-level default Client.
//
// SetHTTP3QUICChromeProfile 将调用委托给包级默认 Client 的 SetHTTP3QUICChromeProfile 方法。
func SetHTTP3QUICChromeProfile() *Client {
	return defaultClient.SetHTTP3QUICChromeProfile()
}

// SetHTTP3TLSClientConfig delegates to SetHTTP3TLSClientConfig on the package-level default Client.
//
// SetHTTP3TLSClientConfig 将调用委托给包级默认 Client 的 SetHTTP3TLSClientConfig 方法。
func SetHTTP3TLSClientConfig(cfg *tls.Config) *Client {
	return defaultClient.SetHTTP3TLSClientConfig(cfg)
}

// SetHTTP3TLSChromeProfile delegates to SetHTTP3TLSChromeProfile on the package-level default Client.
//
// SetHTTP3TLSChromeProfile 将调用委托给包级默认 Client 的 SetHTTP3TLSChromeProfile 方法。
func SetHTTP3TLSChromeProfile() *Client {
	return defaultClient.SetHTTP3TLSChromeProfile()
}

// SetHTTP3TLSFirefoxProfile delegates to SetHTTP3TLSFirefoxProfile on the package-level default Client.
//
// SetHTTP3TLSFirefoxProfile 将调用委托给包级默认 Client 的 SetHTTP3TLSFirefoxProfile 方法。
func SetHTTP3TLSFirefoxProfile() *Client {
	return defaultClient.SetHTTP3TLSFirefoxProfile()
}

// EnableHTTP3FallbackOnError delegates to EnableHTTP3FallbackOnError on the package-level default Client.
//
// EnableHTTP3FallbackOnError 将调用委托给包级默认 Client 的 EnableHTTP3FallbackOnError 方法。
func EnableHTTP3FallbackOnError() *Client {
	return defaultClient.EnableHTTP3FallbackOnError()
}

// DisableHTTP3FallbackOnError delegates to DisableHTTP3FallbackOnError on the package-level default Client.
//
// DisableHTTP3FallbackOnError 将调用委托给包级默认 Client 的 DisableHTTP3FallbackOnError 方法。
func DisableHTTP3FallbackOnError() *Client {
	return defaultClient.DisableHTTP3FallbackOnError()
}

// SetHTTP3AltSvcFailureCooldown delegates to SetHTTP3AltSvcFailureCooldown on the package-level default Client.
//
// SetHTTP3AltSvcFailureCooldown 将调用委托给包级默认 Client 的 SetHTTP3AltSvcFailureCooldown 方法。
func SetHTTP3AltSvcFailureCooldown(cooldown time.Duration) *Client {
	return defaultClient.SetHTTP3AltSvcFailureCooldown(cooldown)
}

// DisableForceHttpVersion delegates to DisableForceHttpVersion on the package-level default Client.
//
// DisableForceHttpVersion 将调用委托给包级默认 Client 的 DisableForceHttpVersion 方法。
func DisableForceHttpVersion() *Client {
	return defaultClient.DisableForceHttpVersion()
}

// EnableH2C delegates to EnableH2C on the package-level default Client.
//
// EnableH2C 将调用委托给包级默认 Client 的 EnableH2C 方法。
func EnableH2C() *Client {
	return defaultClient.EnableH2C()
}

// DisableH2C delegates to DisableH2C on the package-level default Client.
//
// DisableH2C 将调用委托给包级默认 Client 的 DisableH2C 方法。
func DisableH2C() *Client {
	return defaultClient.DisableH2C()
}

// DisableAllowGetMethodPayload delegates to DisableAllowGetMethodPayload on the package-level default Client.
//
// DisableAllowGetMethodPayload 将调用委托给包级默认 Client 的 DisableAllowGetMethodPayload 方法。
func DisableAllowGetMethodPayload() *Client {
	return defaultClient.DisableAllowGetMethodPayload()
}

// EnableAllowGetMethodPayload delegates to EnableAllowGetMethodPayload on the package-level default Client.
//
// EnableAllowGetMethodPayload 将调用委托给包级默认 Client 的 EnableAllowGetMethodPayload 方法。
func EnableAllowGetMethodPayload() *Client {
	return defaultClient.EnableAllowGetMethodPayload()
}

// SetCommonRetryCount delegates to SetCommonRetryCount on the package-level default Client.
//
// SetCommonRetryCount 将调用委托给包级默认 Client 的 SetCommonRetryCount 方法。
func SetCommonRetryCount(count int) *Client {
	return defaultClient.SetCommonRetryCount(count)
}

// SetCommonRetryInterval delegates to SetCommonRetryInterval on the package-level default Client.
//
// SetCommonRetryInterval 将调用委托给包级默认 Client 的 SetCommonRetryInterval 方法。
func SetCommonRetryInterval(getRetryIntervalFunc GetRetryIntervalFunc) *Client {
	return defaultClient.SetCommonRetryInterval(getRetryIntervalFunc)
}

// SetCommonRetryFixedInterval delegates to SetCommonRetryFixedInterval on the package-level default Client.
//
// SetCommonRetryFixedInterval 将调用委托给包级默认 Client 的 SetCommonRetryFixedInterval 方法。
func SetCommonRetryFixedInterval(interval time.Duration) *Client {
	return defaultClient.SetCommonRetryFixedInterval(interval)
}

// SetCommonRetryBackoffInterval delegates to SetCommonRetryBackoffInterval on the package-level default Client.
//
// SetCommonRetryBackoffInterval 将调用委托给包级默认 Client 的 SetCommonRetryBackoffInterval 方法。
func SetCommonRetryBackoffInterval(min, max time.Duration) *Client {
	return defaultClient.SetCommonRetryBackoffInterval(min, max)
}

// SetCommonRetryHook delegates to SetCommonRetryHook on the package-level default Client.
//
// SetCommonRetryHook 将调用委托给包级默认 Client 的 SetCommonRetryHook 方法。
func SetCommonRetryHook(hook RetryHookFunc) *Client {
	return defaultClient.SetCommonRetryHook(hook)
}

// AddCommonRetryHook delegates to AddCommonRetryHook on the package-level default Client.
//
// AddCommonRetryHook 将调用委托给包级默认 Client 的 AddCommonRetryHook 方法。
func AddCommonRetryHook(hook RetryHookFunc) *Client {
	return defaultClient.AddCommonRetryHook(hook)
}

// SetCommonRetryCondition delegates to SetCommonRetryCondition on the package-level default Client.
//
// SetCommonRetryCondition 将调用委托给包级默认 Client 的 SetCommonRetryCondition 方法。
func SetCommonRetryCondition(condition RetryConditionFunc) *Client {
	return defaultClient.SetCommonRetryCondition(condition)
}

// AddCommonRetryCondition delegates to AddCommonRetryCondition on the package-level default Client.
//
// AddCommonRetryCondition 将调用委托给包级默认 Client 的 AddCommonRetryCondition 方法。
func AddCommonRetryCondition(condition RetryConditionFunc) *Client {
	return defaultClient.AddCommonRetryCondition(condition)
}

// SetResponseBodyTransformer delegates to SetResponseBodyTransformer on the package-level default Client.
//
// SetResponseBodyTransformer 将调用委托给包级默认 Client 的 SetResponseBodyTransformer 方法。
func SetResponseBodyTransformer(fn func(rawBody []byte, req *Request, resp *Response) (transformedBody []byte, err error)) *Client {
	return defaultClient.SetResponseBodyTransformer(fn)
}

// SetUnixSocket delegates to SetUnixSocket on the package-level default Client.
//
// SetUnixSocket 将调用委托给包级默认 Client 的 SetUnixSocket 方法。
func SetUnixSocket(file string) *Client {
	return defaultClient.SetUnixSocket(file)
}

// SetResolver delegates to SetResolver on the package-level default Client.
//
// SetResolver 将调用委托给包级默认 Client 的 SetResolver 方法。
func SetResolver(r *net.Resolver) *Client {
	return defaultClient.SetResolver(r)
}

// SetHosts delegates to SetHosts on the package-level default Client.
//
// SetHosts 将调用委托给包级默认 Client 的 SetHosts 方法。
func SetHosts(hosts map[string]string) *Client {
	return defaultClient.SetHosts(hosts)
}

// SetTLSFingerprint delegates to SetTLSFingerprint on the package-level default Client.
//
// SetTLSFingerprint 将调用委托给包级默认 Client 的 SetTLSFingerprint 方法。
func SetTLSFingerprint(clientHelloID utls.ClientHelloID) *Client {
	return defaultClient.SetTLSFingerprint(clientHelloID)
}

// SetTLSFingerprintRandomized delegates to SetTLSFingerprintRandomized on the package-level default Client.
//
// SetTLSFingerprintRandomized 将调用委托给包级默认 Client 的 SetTLSFingerprintRandomized 方法。
func SetTLSFingerprintRandomized() *Client {
	return defaultClient.SetTLSFingerprintRandomized()
}

// SetTLSFingerprintChrome delegates to SetTLSFingerprintChrome on the package-level default Client.
//
// SetTLSFingerprintChrome 将调用委托给包级默认 Client 的 SetTLSFingerprintChrome 方法。
func SetTLSFingerprintChrome() *Client {
	return defaultClient.SetTLSFingerprintChrome()
}

// SetTLSFingerprintAndroid delegates to SetTLSFingerprintAndroid on the package-level default Client.
//
// SetTLSFingerprintAndroid 将调用委托给包级默认 Client 的 SetTLSFingerprintAndroid 方法。
func SetTLSFingerprintAndroid() *Client {
	return defaultClient.SetTLSFingerprintAndroid()
}

// SetTLSFingerprint360 delegates to SetTLSFingerprint360 on the package-level default Client.
//
// SetTLSFingerprint360 将调用委托给包级默认 Client 的 SetTLSFingerprint360 方法。
func SetTLSFingerprint360() *Client {
	return defaultClient.SetTLSFingerprint360()
}

// SetTLSFingerprintEdge delegates to SetTLSFingerprintEdge on the package-level default Client.
//
// SetTLSFingerprintEdge 将调用委托给包级默认 Client 的 SetTLSFingerprintEdge 方法。
func SetTLSFingerprintEdge() *Client {
	return defaultClient.SetTLSFingerprintEdge()
}

// SetTLSFingerprintFirefox delegates to SetTLSFingerprintFirefox on the package-level default Client.
//
// SetTLSFingerprintFirefox 将调用委托给包级默认 Client 的 SetTLSFingerprintFirefox 方法。
func SetTLSFingerprintFirefox() *Client {
	return defaultClient.SetTLSFingerprintFirefox()
}

// SetTLSFingerprintQQ delegates to SetTLSFingerprintQQ on the package-level default Client.
//
// SetTLSFingerprintQQ 将调用委托给包级默认 Client 的 SetTLSFingerprintQQ 方法。
func SetTLSFingerprintQQ() *Client {
	return defaultClient.SetTLSFingerprintQQ()
}

// SetTLSFingerprintIOS delegates to SetTLSFingerprintIOS on the package-level default Client.
//
// SetTLSFingerprintIOS 将调用委托给包级默认 Client 的 SetTLSFingerprintIOS 方法。
func SetTLSFingerprintIOS() *Client {
	return defaultClient.SetTLSFingerprintIOS()
}

// SetTLSFingerprintSafari delegates to SetTLSFingerprintSafari on the package-level default Client.
//
// SetTLSFingerprintSafari 将调用委托给包级默认 Client 的 SetTLSFingerprintSafari 方法。
func SetTLSFingerprintSafari() *Client {
	return defaultClient.SetTLSFingerprintSafari()
}

// GetClient delegates to GetClient on the package-level default Client.
//
// GetClient 将调用委托给包级默认 Client 的 GetClient 方法。
func GetClient() *http.Client {
	return defaultClient.GetClient()
}

// NewRequest returns a new Request from the package-level default Client.
//
// NewRequest 从包级默认 Client 创建并返回新的 Request。
func NewRequest() *Request {
	return defaultClient.R()
}

// R delegates to R on the package-level default Client.
//
// R 将调用委托给包级默认 Client 的 R 方法。
func R() *Request {
	return defaultClient.R()
}
