package req

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jwwsjlm/req/v3/internal/header"
	"github.com/jwwsjlm/req/v3/internal/tests"
	"github.com/jwwsjlm/req/v3/pkg/altsvc"
	"github.com/quic-go/quic-go"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/publicsuffix"
)

func TestRetryCancelledContext(t *testing.T) {
	cancelledCtx, done := context.WithCancel(context.Background())
	done()

	client := tc().
		SetCommonRetryCount(2).
		SetCommonRetryBackoffInterval(1*time.Second, 5*time.Second)

	res, err := client.R().SetContext(cancelledCtx).Get("/")

	tests.AssertEqual(t, 0, res.Request.RetryAttempt)
	tests.AssertNotNil(t, err)
	tests.AssertErrorContains(t, err, "context canceled")
}

func TestWrapRoundTrip(t *testing.T) {
	i, j, a, b := 0, 0, 0, 0
	c := tc().WrapRoundTripFunc(func(rt RoundTripper) RoundTripFunc {
		return func(req *Request) (resp *Response, err error) {
			a = 1
			resp, err = rt.RoundTrip(req)
			b = 1
			return
		}
	})
	c.GetTransport().WrapRoundTripFunc(func(rt http.RoundTripper) HttpRoundTripFunc {
		return func(req *http.Request) (resp *http.Response, err error) {
			i = 1
			resp, err = rt.RoundTrip(req)
			j = 1
			return
		}
	})
	resp, err := c.R().Get("/")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, 1, i)
	tests.AssertEqual(t, 1, j)
	tests.AssertEqual(t, 1, a)
	tests.AssertEqual(t, 1, b)
}

func TestAllowGetMethodPayload(t *testing.T) {
	c := tc()
	resp, err := c.R().SetBody("test").Get("/payload")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test", resp.String())

	c.DisableAllowGetMethodPayload()
	resp, err = c.R().SetBody("test").Get("/payload")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "", resp.String())

	c.EnableAllowGetMethodPayload()
	resp, err = c.R().SetBody("test").Get("/payload")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test", resp.String())
}

func TestSetTLSHandshakeTimeout(t *testing.T) {
	timeout := 2 * time.Second
	c := tc().SetTLSHandshakeTimeout(timeout)
	tests.AssertEqual(t, timeout, c.TLSHandshakeTimeout)
}

func TestSetDial(t *testing.T) {
	testErr := errors.New("test")
	testDial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, testErr
	}
	c := tc().SetDial(testDial)
	_, err := c.DialContext(nil, "", "")
	tests.AssertEqual(t, testErr, err)
}

func TestSetDialTLS(t *testing.T) {
	testErr := errors.New("test")
	testDialTLS := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, testErr
	}
	c := tc().SetDialTLS(testDialTLS)
	_, err := c.DialTLSContext(nil, "", "")
	tests.AssertEqual(t, testErr, err)
}

func TestSetFuncs(t *testing.T) {
	testErr := errors.New("test")
	marshalFunc := func(v any) ([]byte, error) {
		return nil, testErr
	}
	unmarshalFunc := func(data []byte, v any) error {
		return testErr
	}
	c := tc().
		SetJsonMarshal(marshalFunc).
		SetJsonUnmarshal(unmarshalFunc).
		SetXmlMarshal(marshalFunc).
		SetXmlUnmarshal(unmarshalFunc)

	_, err := c.jsonMarshal(nil)
	tests.AssertEqual(t, testErr, err)
	err = c.jsonUnmarshal(nil, nil)
	tests.AssertEqual(t, testErr, err)

	_, err = c.xmlMarshal(nil)
	tests.AssertEqual(t, testErr, err)
	err = c.xmlUnmarshal(nil, nil)
	tests.AssertEqual(t, testErr, err)
}

func TestSetCookieJar(t *testing.T) {
	c := tc().SetCookieJar(nil)
	tests.AssertEqual(t, nil, c.httpClient.Jar)
}

func TestTraceAll(t *testing.T) {
	c := tc().EnableTraceAll()
	resp, err := c.R().Get("/")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, true, resp.TraceInfo().TotalTime > 0)

	c.DisableTraceAll()
	resp, err = c.R().Get("/")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, true, resp.TraceInfo().TotalTime == 0)
}

func TestOnAfterResponse(t *testing.T) {
	c := tc()
	len1 := len(c.afterResponse)
	c.OnAfterResponse(func(client *Client, response *Response) error {
		return nil
	})
	len2 := len(c.afterResponse)
	tests.AssertEqual(t, true, len1+1 == len2)
}

func TestOnBeforeRequest(t *testing.T) {
	c := tc().OnBeforeRequest(func(client *Client, request *Request) error {
		return nil
	})
	tests.AssertEqual(t, true, len(c.udBeforeRequest) == 1)
}

func TestSetProxyURL(t *testing.T) {
	c := tc().SetProxyURL("http://dummy.proxy.local")
	u, err := c.Proxy(nil)
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, "http://dummy.proxy.local", u.String())
}

func TestSetProxy(t *testing.T) {
	u, _ := url.Parse("http://dummy.proxy.local")
	proxy := http.ProxyURL(u)
	c := tc().SetProxy(proxy)
	uu, err := c.Proxy(nil)
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, u.String(), uu.String())
}

func TestSetCommonContentType(t *testing.T) {
	c := tc().SetCommonContentType(header.JsonContentType)
	tests.AssertEqual(t, header.JsonContentType, c.Headers.Get(header.ContentType))
}

func TestSetCommonHeader(t *testing.T) {
	c := tc().SetCommonHeader("my-header", "my-value")
	tests.AssertEqual(t, "my-value", c.Headers.Get("my-header"))
}

func TestSetCommonHeaderNonCanonical(t *testing.T) {
	c := tc().SetCommonHeaderNonCanonical("my-Header", "my-value")
	tests.AssertEqual(t, "my-value", c.Headers["my-Header"][0])
}

func TestSetCommonHeaders(t *testing.T) {
	c := tc().SetCommonHeaders(map[string]string{
		"header1": "value1",
		"header2": "value2",
	})
	tests.AssertEqual(t, "value1", c.Headers.Get("header1"))
	tests.AssertEqual(t, "value2", c.Headers.Get("header2"))
}

func TestSetCommonHeadersNonCanonical(t *testing.T) {
	c := tc().SetCommonHeadersNonCanonical(map[string]string{
		"my-Header": "my-value",
	})
	tests.AssertEqual(t, "my-value", c.Headers["my-Header"][0])
}

func TestSetCommonBasicAuth(t *testing.T) {
	c := tc().SetCommonBasicAuth("imroc", "123456")
	tests.AssertEqual(t, "Basic aW1yb2M6MTIzNDU2", c.Headers.Get("Authorization"))
}

func TestSetCommonBearerAuthToken(t *testing.T) {
	c := tc().SetCommonBearerAuthToken("123456")
	tests.AssertEqual(t, "Bearer 123456", c.Headers.Get("Authorization"))
}

func TestSetUserAgent(t *testing.T) {
	c := tc().SetUserAgent("test")
	tests.AssertEqual(t, "test", c.Headers.Get(header.UserAgent))
}

func TestAutoDecode(t *testing.T) {
	c := tc().DisableAutoDecode()
	resp, err := c.R().Get("/gbk")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, toGbk("我是roc"), resp.Bytes())

	resp, err = c.EnableAutoDecode().R().Get("/gbk")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "我是roc", resp.String())

	resp, err = c.SetAutoDecodeContentType("html").R().Get("/gbk")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, toGbk("我是roc"), resp.Bytes())
	resp, err = c.SetAutoDecodeContentType("text").R().Get("/gbk")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "我是roc", resp.String())
	resp, err = c.SetAutoDecodeContentTypeFunc(func(contentType string) bool {
		return strings.Contains(contentType, "text")
	}).R().Get("/gbk")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "我是roc", resp.String())

	resp, err = c.SetAutoDecodeAllContentType().R().Get("/gbk-no-charset")
	assertSuccess(t, resp, err)
	tests.AssertContains(t, resp.String(), "我是roc", true)
}

func TestSetTimeout(t *testing.T) {
	timeout := 100 * time.Second
	c := tc().SetTimeout(timeout)
	tests.AssertEqual(t, timeout, c.httpClient.Timeout)
}

func TestSetLogger(t *testing.T) {
	l := createDefaultLogger()
	c := tc().SetLogger(l)
	tests.AssertEqual(t, l, c.log)

	c.SetLogger(nil)
	tests.AssertEqual(t, &disableLogger{}, c.log)
}

func TestSetScheme(t *testing.T) {
	c := tc().SetScheme("https")
	tests.AssertEqual(t, "https", c.scheme)
}

func TestDebugLog(t *testing.T) {
	c := tc().EnableDebugLog()
	tests.AssertEqual(t, true, c.DebugLog)

	c.DisableDebugLog()
	tests.AssertEqual(t, false, c.DebugLog)
}

func TestSetCommonCookies(t *testing.T) {
	headers := make(http.Header)
	resp, err := tc().SetCommonCookies(&http.Cookie{
		Name:  "test",
		Value: "test",
	}).R().SetSuccessResult(&headers).Get("/header")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test=test", headers.Get("Cookie"))
}

func TestSetCommonQueryString(t *testing.T) {
	resp, err := tc().SetCommonQueryString("test=test").R().Get("/query-parameter")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test=test", resp.String())
}

func TestSetCommonPathParams(t *testing.T) {
	c := tc().SetCommonPathParams(map[string]string{"test": "test"})
	tests.AssertNotNil(t, c.PathParams)
	tests.AssertEqual(t, "test", c.PathParams["test"])
}

func TestSetCommonPathParam(t *testing.T) {
	c := tc().SetCommonPathParam("test", "test")
	tests.AssertNotNil(t, c.PathParams)
	tests.AssertEqual(t, "test", c.PathParams["test"])
}

func TestAddCommonQueryParam(t *testing.T) {
	resp, err := tc().
		AddCommonQueryParam("test", "1").
		AddCommonQueryParam("test", "2").
		R().Get("/query-parameter")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test=1&test=2", resp.String())
}

func TestSetCommonQueryParam(t *testing.T) {
	resp, err := tc().SetCommonQueryParam("test", "test").R().Get("/query-parameter")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test=test", resp.String())
}

func TestSetCommonQueryParams(t *testing.T) {
	resp, err := tc().SetCommonQueryParams(map[string]string{"test": "test"}).R().Get("/query-parameter")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test=test", resp.String())
}

func TestSetCommonQueryParamsFromValues(t *testing.T) {
	values := url.Values{}
	values.Add("test", "test")
	values.Add("key", "value")
	resp, err := tc().SetCommonQueryParamsFromValues(values).R().Get("/query-parameter")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "key=value&test=test", resp.String())
}

func TestSetCommonQueryParamsFromStruct(t *testing.T) {
	type QueryParams struct {
		Test string `url:"test"`
		Key  string `url:"key"`
	}
	params := QueryParams{
		Test: "test",
		Key:  "value",
	}
	resp, err := tc().SetCommonQueryParamsFromStruct(params).R().Get("/query-parameter")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "key=value&test=test", resp.String())
}

func TestInsecureSkipVerify(t *testing.T) {
	c := tc().EnableInsecureSkipVerify()
	tests.AssertEqual(t, true, c.TLSClientConfig.InsecureSkipVerify)

	c.DisableInsecureSkipVerify()
	tests.AssertEqual(t, false, c.TLSClientConfig.InsecureSkipVerify)
}

func TestSetTLSClientConfig(t *testing.T) {
	config := &tls.Config{InsecureSkipVerify: true}
	c := tc().SetTLSClientConfig(config)
	tests.AssertEqual(t, config, c.TLSClientConfig)
}

func TestCompression(t *testing.T) {
	c := tc().DisableCompression()
	tests.AssertEqual(t, true, c.Transport.DisableCompression)

	c.EnableCompression()
	tests.AssertEqual(t, false, c.Transport.DisableCompression)
}

func TestKeepAlives(t *testing.T) {
	c := tc().DisableKeepAlives()
	tests.AssertEqual(t, true, c.Transport.DisableKeepAlives)

	c.EnableKeepAlives()
	tests.AssertEqual(t, false, c.Transport.DisableKeepAlives)
}

func TestRedirect(t *testing.T) {
	_, err := tc().SetRedirectPolicy(NoRedirectPolicy()).R().Get("/unlimited-redirect")
	tests.AssertIsNil(t, err)

	_, err = tc().SetRedirectPolicy(MaxRedirectPolicy(3)).R().Get("/unlimited-redirect")
	tests.AssertNotNil(t, err)
	tests.AssertContains(t, err.Error(), "stopped after 3 redirects", true)

	_, err = tc().SetRedirectPolicy(MaxRedirectPolicy(20)).SetRedirectPolicy(DefaultRedirectPolicy()).R().Get("/unlimited-redirect")
	tests.AssertNotNil(t, err)
	tests.AssertContains(t, err.Error(), "stopped after 10 redirects", true)

	_, err = tc().SetRedirectPolicy(SameDomainRedirectPolicy()).R().Get("/redirect-to-other")
	tests.AssertNotNil(t, err)
	tests.AssertContains(t, err.Error(), "different domain name is not allowed", true)

	_, err = tc().SetRedirectPolicy(SameHostRedirectPolicy()).R().Get("/redirect-to-other")
	tests.AssertNotNil(t, err)
	tests.AssertContains(t, err.Error(), "different host name is not allowed", true)

	_, err = tc().SetRedirectPolicy(AllowedHostRedirectPolicy("localhost", "127.0.0.1")).R().Get("/redirect-to-other")
	tests.AssertNotNil(t, err)
	tests.AssertContains(t, err.Error(), "redirect host [dummy.local] is not allowed", true)

	_, err = tc().SetRedirectPolicy(AllowedDomainRedirectPolicy("localhost", "127.0.0.1")).R().Get("/redirect-to-other")
	tests.AssertNotNil(t, err)
	tests.AssertContains(t, err.Error(), "redirect domain [dummy.local] is not allowed", true)

	c := tc().SetRedirectPolicy(AlwaysCopyHeaderRedirectPolicy("Authorization"))
	newHeader := make(http.Header)
	oldHeader := make(http.Header)
	oldHeader.Set("Authorization", "test")
	c.GetClient().CheckRedirect(&http.Request{
		Header: newHeader,
	}, []*http.Request{{
		Header: oldHeader,
	}})
	tests.AssertEqual(t, "test", newHeader.Get("Authorization"))
}

func TestGetTLSClientConfig(t *testing.T) {
	c := tc()
	config := c.GetTLSClientConfig()
	tests.AssertEqual(t, true, c.TLSClientConfig != nil)
	tests.AssertEqual(t, config, c.TLSClientConfig)
}

func TestSetRootCertFromFile(t *testing.T) {
	c := tc().SetRootCertsFromFile(tests.GetTestFilePath("sample-root.pem"))
	tests.AssertEqual(t, true, c.TLSClientConfig.RootCAs != nil)
}

func TestSetRootCertFromString(t *testing.T) {
	c := tc().SetRootCertFromString(string(getTestFileContent(t, "sample-root.pem")))
	tests.AssertEqual(t, true, c.TLSClientConfig.RootCAs != nil)
}

func TestSetCerts(t *testing.T) {
	c := tc().SetCerts(tls.Certificate{}, tls.Certificate{})
	tests.AssertEqual(t, true, len(c.TLSClientConfig.Certificates) == 2)
}

func TestSetCertFromFile(t *testing.T) {
	c := tc().SetCertFromFile(
		tests.GetTestFilePath("sample-client.pem"),
		tests.GetTestFilePath("sample-client-key.pem"),
	)
	tests.AssertEqual(t, true, len(c.TLSClientConfig.Certificates) == 1)
}

func TestSetOutputDirectory(t *testing.T) {
	outFile := "test_output_dir"
	resp, err := tc().
		SetOutputDirectory(testDataPath).
		R().SetOutputFile(outFile).
		Get("/")
	assertSuccess(t, resp, err)
	content := string(getTestFileContent(t, outFile))
	os.Remove(tests.GetTestFilePath(outFile))
	tests.AssertEqual(t, "TestGet: text response", content)
}

func TestSetBaseURL(t *testing.T) {
	baseURL := "http://dummy-req.local/test"
	resp, _ := tc().SetTimeout(time.Nanosecond).SetBaseURL(baseURL).R().Get("/req")
	tests.AssertEqual(t, baseURL+"/req", resp.Request.RawRequest.URL.String())
}

func TestSetCommonFormDataFromValues(t *testing.T) {
	expectedForm := make(url.Values)
	gotForm := make(url.Values)
	expectedForm.Set("test", "test")
	resp, err := tc().
		SetCommonFormDataFromValues(expectedForm).
		R().SetSuccessResult(&gotForm).
		Post("/form")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test", gotForm.Get("test"))
}

func TestSetCommonFormData(t *testing.T) {
	form := make(url.Values)
	resp, err := tc().
		SetCommonFormData(
			map[string]string{
				"test": "test",
			}).R().
		SetSuccessResult(&form).
		Post("/form")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test", form.Get("test"))
}

func TestSetMultipartBoundaryFunc(t *testing.T) {
	delimiter := "test-delimiter"
	expectedContentType := fmt.Sprintf("multipart/form-data; boundary=%s", delimiter)
	resp, err := tc().
		SetMultipartBoundaryFunc(func() string {
			return delimiter
		}).R().
		EnableForceMultipart().
		SetFormData(
			map[string]string{
				"test": "test",
			}).
		Post("/content-type")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, expectedContentType, resp.String())
}

func TestFirefoxMultipartBoundaryFunc(t *testing.T) {
	r := regexp.MustCompile(`^-------------------------\d{1,10}\d{1,10}\d{1,10}$`)
	b := firefoxMultipartBoundaryFunc()
	tests.AssertEqual(t, true, r.MatchString(b))
}

func TestWebkitMultipartBoundaryFunc(t *testing.T) {
	r := regexp.MustCompile(`^----WebKitFormBoundary[0-9a-zA-Z]{16}$`)
	b := webkitMultipartBoundaryFunc()
	tests.AssertEqual(t, true, r.MatchString(b))
}

func TestClientClone(t *testing.T) {
	c1 := tc().DevMode().
		SetCommonHeader("test", "test").
		SetCommonCookies(&http.Cookie{
			Name:  "test",
			Value: "test",
		}).SetCommonQueryParam("test", "test").
		SetCommonPathParam("test", "test").
		SetCommonRetryCount(2).
		SetCommonFormData(map[string]string{"test": "test"}).
		OnBeforeRequest(func(c *Client, r *Request) error { return nil })

	c2 := c1.Clone()
	assertClone(t, c1, c2)
}

func TestDisableAutoReadResponse(t *testing.T) {
	testWithAllTransport(t, testDisableAutoReadResponse)
}

func testDisableAutoReadResponse(t *testing.T, c *Client) {
	c.DisableAutoReadResponse()
	resp, err := c.R().Get("/")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "", resp.String())
	result, err := resp.ToString()
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, "TestGet: text response", result)

	resp, err = c.R().Get("/")
	assertSuccess(t, resp, err)
	_, err = io.ReadAll(resp.Body)
	tests.AssertNoError(t, err)
}

func testEnableDumpAll(t *testing.T, fn func(c *Client) (de dumpExpected)) {
	testDump := func(c *Client) {
		buff := new(bytes.Buffer)
		c.EnableDumpAllTo(buff)
		r := c.R()
		de := fn(c)
		resp, err := r.SetBody(`test body`).Post("/")
		assertSuccess(t, resp, err)
		dump := buff.String()
		tests.AssertContains(t, dump, "user-agent", de.ReqHeader)
		tests.AssertContains(t, dump, "test body", de.ReqBody)
		tests.AssertContains(t, dump, "date", de.RespHeader)
		tests.AssertContains(t, dump, "testpost: text response", de.RespBody)
	}
	c := tc()
	testDump(c)
	testDump(c.EnableForceHTTP1())
}

func TestEnableDumpAll(t *testing.T) {
	testCases := []func(c *Client) (d dumpExpected){
		func(c *Client) (de dumpExpected) {
			c.EnableDumpAll()
			de.ReqHeader = true
			de.ReqBody = true
			de.RespHeader = true
			de.RespBody = true
			return
		},
		func(c *Client) (de dumpExpected) {
			c.EnableDumpAllWithoutHeader()
			de.ReqBody = true
			de.RespBody = true
			return
		},
		func(c *Client) (de dumpExpected) {
			c.EnableDumpAllWithoutBody()
			de.ReqHeader = true
			de.RespHeader = true
			return
		},
		func(c *Client) (de dumpExpected) {
			c.EnableDumpAllWithoutRequest()
			de.RespHeader = true
			de.RespBody = true
			return
		},
		func(c *Client) (de dumpExpected) {
			c.EnableDumpAllWithoutRequestBody()
			de.ReqHeader = true
			de.RespHeader = true
			de.RespBody = true
			return
		},
		func(c *Client) (de dumpExpected) {
			c.EnableDumpAllWithoutResponse()
			de.ReqHeader = true
			de.ReqBody = true
			return
		},
		func(c *Client) (de dumpExpected) {
			c.EnableDumpAllWithoutResponseBody()
			de.ReqHeader = true
			de.ReqBody = true
			de.RespHeader = true
			return
		},
		func(c *Client) (de dumpExpected) {
			c.SetCommonDumpOptions(&DumpOptions{
				RequestHeader: true,
				RequestBody:   true,
				ResponseBody:  true,
			}).EnableDumpAll()
			de.ReqHeader = true
			de.ReqBody = true
			de.RespBody = true
			return
		},
	}
	for _, fn := range testCases {
		testEnableDumpAll(t, fn)
	}
}

func TestEnableDumpAllToFile(t *testing.T) {
	c := tc()
	dumpFile := "tmp_test_dump_file"
	c.EnableDumpAllToFile(tests.GetTestFilePath(dumpFile))
	resp, err := c.R().SetBody("test body").Post("/")
	assertSuccess(t, resp, err)
	dump := string(getTestFileContent(t, dumpFile))
	os.Remove(tests.GetTestFilePath(dumpFile))
	tests.AssertContains(t, dump, "user-agent", true)
	tests.AssertContains(t, dump, "test body", true)
	tests.AssertContains(t, dump, "date", true)
	tests.AssertContains(t, dump, "testpost: text response", true)
}

func TestEnableDumpAllAsync(t *testing.T) {
	c := tc()
	buf := new(bytes.Buffer)
	c.EnableDumpAllTo(buf).EnableDumpAllAsync()
	tests.AssertEqual(t, true, c.getDumpOptions().Async)
}

func TestSetResponseBodyTransformer(t *testing.T) {
	c := tc().SetResponseBodyTransformer(func(rawBody []byte, req *Request, resp *Response) (transformedBody []byte, err error) {
		if resp.IsSuccessState() {
			result, err := url.QueryUnescape(string(rawBody))
			return []byte(result), err
		}
		return rawBody, nil
	})
	user := &UserInfo{}
	resp, err := c.R().SetSuccessResult(user).Get("/urlencode")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, user.Username, "我是roc")
	tests.AssertEqual(t, user.Email, "roc@imroc.cc")
}

func TestSetResultStateCheckFunc(t *testing.T) {
	c := tc().SetResultStateCheckFunc(func(resp *Response) ResultState {
		if resp.StatusCode == http.StatusOK {
			return SuccessState
		} else {
			return ErrorState
		}
	})
	resp, err := c.R().Get("/status?code=200")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, SuccessState, resp.ResultState())

	resp, err = c.R().Get("/status?code=201")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, ErrorState, resp.ResultState())

	resp, err = c.R().Get("/status?code=399")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, ErrorState, resp.ResultState())

	resp, err = c.R().Get("/status?code=404")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, ErrorState, resp.ResultState())
}

func TestCloneCookieJar(t *testing.T) {
	c1 := C()
	c2 := c1.Clone()
	tests.AssertEqual(t, true, c1.httpClient.Jar != c2.httpClient.Jar)

	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	c1.SetCookieJar(jar)
	c2 = c1.Clone()
	tests.AssertEqual(t, true, c1.httpClient.Jar == c2.httpClient.Jar)

	c2.SetCookieJar(nil)
	tests.AssertEqual(t, true, c2.cookiejarFactory == nil)
	tests.AssertEqual(t, true, c2.httpClient.Jar == nil)
}

type customCookieJar struct{}

func (customCookieJar) SetCookies(*url.URL, []*http.Cookie) {}

func (customCookieJar) Cookies(*url.URL) []*http.Cookie {
	return nil
}

func TestSetCookieJarFactoryAcceptsHTTPJar(t *testing.T) {
	jar := &customCookieJar{}
	c := C().SetCookieJarFactory(func() http.CookieJar {
		return jar
	})
	tests.AssertEqual(t, true, c.httpClient.Jar == jar)
}

func TestSetCookieJarFactoryAcceptsLegacyJar(t *testing.T) {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	c := C().SetCookieJarFactory(func() *cookiejar.Jar {
		return jar
	})
	tests.AssertEqual(t, true, c.httpClient.Jar == jar)
}

func TestSetDNSResolver(t *testing.T) {
	resolver := &net.Resolver{PreferGo: true}
	c := C().SetDNSResolver(resolver)
	tests.AssertEqual(t, true, c.Transport.Resolver == resolver)

	c.EnableHTTP3()
	tests.AssertEqual(t, true, c.Transport.t3.Resolver == resolver)

	clone := c.Clone()
	tests.AssertEqual(t, true, clone.Transport.Resolver == resolver)
	tests.AssertEqual(t, true, clone.Transport.t3.Resolver == resolver)
}

func TestSetDNSResolverAfterHTTP3Enabled(t *testing.T) {
	c := C().EnableHTTP3()
	resolver := &net.Resolver{PreferGo: true}
	c.SetDNSResolver(resolver)
	tests.AssertEqual(t, true, c.Transport.t3.Resolver == resolver)
}

func TestNewDNSOverTLSResolver(t *testing.T) {
	resolver := NewDNSOverTLSResolver(DNSOverTLSCloudflare)
	tests.AssertNotNil(t, resolver)
	tests.AssertEqual(t, true, resolver.PreferGo)
	tests.AssertEqual(t, true, resolver.Dial != nil)
}

func TestDNSOverTLSConvenienceMethods(t *testing.T) {
	tests.AssertNotNil(t, C().SetDNSOverTLSCloudflare().Transport.Resolver)
	tests.AssertNotNil(t, C().SetDNSOverTLSGoogle().Transport.Resolver)
	tests.AssertNotNil(t, C().SetDNSOverTLSQuad9().Transport.Resolver)
	tests.AssertNotNil(t, C().SetDNSOverTLSAdGuard().Transport.Resolver)
	tests.AssertNotNil(t, C().SetDNSOverTLSAli().Transport.Resolver)
}

func TestResponseTLSInfo(t *testing.T) {
	resp, err := tc().R().Get("/")
	assertSuccess(t, resp, err)

	info := resp.TLSInfo()
	tests.AssertNotNil(t, info)
	tests.AssertEqual(t, true, info.Version != "")
	tests.AssertEqual(t, 64, len(info.FingerprintSHA256))
	tests.AssertEqual(t, true, strings.Contains(info.FingerprintSHA256OpenSSL, ":"))
	tests.AssertEqual(t, info.FingerprintSHA256, resp.TLSGrabber().FingerprintSHA256)
}

func TestResponseTLSInfoHTTPNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := C().R().Get(server.URL)
	assertSuccess(t, resp, err)
	tests.AssertIsNil(t, resp.TLSInfo())
	tests.AssertIsNil(t, resp.TLSGrabber())
}

func TestSetTLSFingerprintSpec(t *testing.T) {
	clientHelloSpec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
	tests.AssertNoError(t, err)

	c := C().SetTLSFingerprintSpec(&clientHelloSpec)
	tests.AssertEqual(t, true, c.Transport.TLSHandshakeContext != nil)
}

func TestSetTLSFingerprintJA3(t *testing.T) {
	const ja3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-5-10-11-13-16-18-21-23-27-35-43-45-51-17513-65281,29-23-24,0"

	c := C().SetTLSFingerprintJA3(ja3, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0", false)
	tests.AssertEqual(t, true, c.Transport.TLSHandshakeContext != nil)
}

func TestHTTP3AdvancedSettings(t *testing.T) {
	settings := map[uint64]uint64{
		HTTP3SettingQpackMaxTableCapacity: 65536,
	}

	c := C().
		SetHTTP3AdditionalSettings(settings).
		SetHTTP3AdditionalSetting(HTTP3SettingQpackBlockedStreams, 100).
		EnableHTTP3Datagrams().
		EnableHTTP3ExtendedConnect().
		SetHTTP3MaxResponseHeaderBytes(262144).
		SetHTTP3QUICConfig(&quic.Config{}).
		EnableHTTP3()

	settings[HTTP3SettingQpackMaxTableCapacity] = 1
	tests.AssertEqual(t, uint64(65536), c.Transport.http3AdditionalSettings[HTTP3SettingQpackMaxTableCapacity])
	tests.AssertEqual(t, uint64(100), c.Transport.http3AdditionalSettings[HTTP3SettingQpackBlockedStreams])
	tests.AssertEqual(t, true, c.Transport.http3EnableDatagrams)
	tests.AssertEqual(t, true, c.Transport.http3EnableExtendedConnect)
	tests.AssertEqual(t, 262144, c.Transport.http3MaxResponseHeaderBytes)
	tests.AssertEqual(t, true, c.Transport.t3.EnableDatagrams)
	tests.AssertEqual(t, true, c.Transport.t3.EnableExtendedConnect)
	tests.AssertEqual(t, 262144, c.Transport.t3.MaxResponseHeaderBytes)
	tests.AssertEqual(t, true, c.Transport.t3.QUICConfig.EnableDatagrams)

	clone := c.Clone()
	tests.AssertEqual(t, uint64(65536), clone.Transport.http3AdditionalSettings[HTTP3SettingQpackMaxTableCapacity])
	tests.AssertEqual(t, true, clone.Transport.http3EnableDatagrams)
	tests.AssertEqual(t, true, clone.Transport.http3EnableExtendedConnect)
	tests.AssertEqual(t, true, clone.Transport.t3.EnableDatagrams)
	tests.AssertEqual(t, true, clone.Transport.t3.EnableExtendedConnect)
	clone.Transport.http3AdditionalSettings[HTTP3SettingQpackMaxTableCapacity] = 2
	tests.AssertEqual(t, uint64(65536), c.Transport.http3AdditionalSettings[HTTP3SettingQpackMaxTableCapacity])
}

func TestHTTP3TLSConfigPropagation(t *testing.T) {
	base := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "base.example",
		NextProtos:         []string{"h2", "http/1.1"},
	}
	c := C().
		SetTLSClientConfig(base).
		EnableHTTP3()

	tests.AssertEqual(t, base, c.Transport.t3.TLSClientConfig)
	tests.AssertEqual(t, true, c.Transport.t3.TLSClientConfig.InsecureSkipVerify)

	http3Config := &tls.Config{
		ServerName: "h3.example",
		MinVersion: tls.VersionTLS13,
	}
	c.SetHTTP3TLSClientConfig(http3Config)
	http3Config.ServerName = "mutated.example"

	tests.AssertEqual(t, "h3.example", c.Transport.http3TLSClientConfig.ServerName)
	tests.AssertEqual(t, "h3.example", c.Transport.t3.TLSClientConfig.ServerName)

	c.SetHTTP3TLSClientConfig(nil)
	tests.AssertEqual(t, base, c.Transport.t3.TLSClientConfig)
}

func TestHTTP3TLSChromeProfile(t *testing.T) {
	c := C().
		SetHTTP3TLSClientConfig(&tls.Config{ServerName: "h3.example"}).
		SetHTTP3TLSChromeProfile().
		EnableHTTP3()

	cfg := c.Transport.http3TLSClientConfig
	tests.AssertNotNil(t, cfg)
	tests.AssertEqual(t, "h3.example", cfg.ServerName)
	tests.AssertEqual(t, uint16(tls.VersionTLS13), cfg.MinVersion)
	tests.AssertEqual(t, uint16(tls.VersionTLS13), cfg.MaxVersion)
	tests.AssertEqual(t, []string{"h3"}, cfg.NextProtos)
	tests.AssertEqual(t, true, cfg.ClientSessionCache != nil)
	tests.AssertEqual(t, true, hasTLSCurve(cfg.CurvePreferences, tls.X25519MLKEM768))
	tests.AssertEqual(t, true, hasTLSCurve(cfg.CurvePreferences, tls.X25519))
	tests.AssertEqual(t, cfg, c.Transport.t3.TLSClientConfig)

	clone := c.Clone()
	tests.AssertEqual(t, "h3.example", clone.Transport.http3TLSClientConfig.ServerName)
	clone.Transport.http3TLSClientConfig.ServerName = "clone.example"
	tests.AssertEqual(t, "h3.example", c.Transport.http3TLSClientConfig.ServerName)
}

func TestHTTP3FallbackOnError(t *testing.T) {
	c := tc().
		EnableHTTP3FallbackOnError().
		EnableForceHTTP3()

	c.Transport.t3.Dial = func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
		return nil, errors.New("h3 unavailable")
	}

	resp, err := c.R().Get("/")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "GET", resp.GetHeader("Method"))

	clone := c.Clone()
	tests.AssertEqual(t, true, clone.Transport.http3FallbackOnFailure)
	c.DisableHTTP3FallbackOnError()
	tests.AssertEqual(t, false, c.Transport.http3FallbackOnFailure)
}

func TestHTTP3AltSvcFallbackOnError(t *testing.T) {
	c := tc().
		EnableHTTP3().
		EnableHTTP3FallbackOnError().
		SetHTTP3AltSvcFailureCooldown(time.Minute)

	dialCount := 0
	c.Transport.t3.Dial = func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
		dialCount++
		return nil, errors.New("h3 unavailable")
	}

	baseURL, err := url.Parse(c.BaseURL)
	tests.AssertNoError(t, err)
	addr := baseURL.Scheme + "://" + baseURL.Host
	c.Transport.altSvcJar.SetAltSvc(addr, &altsvc.AltSvc{
		Protocol: "h3",
		Expire:   time.Now().Add(time.Hour),
	})

	resp, err := c.R().Get("/")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "GET", resp.GetHeader("Method"))
	tests.AssertEqual(t, 1, dialCount)
	tests.AssertEqual(t, true, c.Transport.isHTTP3AltSvcCoolingDown(addr))

	resp, err = c.R().Get("/")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, 1, dialCount)
}

func TestHTTP3QUICPerformanceProfile(t *testing.T) {
	c := C().
		SetHTTP3QUICPerformanceProfile().
		EnableHTTP3Datagrams().
		EnableHTTP3()

	cfg := c.Transport.http3QUICConfig
	tests.AssertNotNil(t, cfg)
	tests.AssertEqual(t, 5*time.Second, cfg.HandshakeIdleTimeout)
	tests.AssertEqual(t, 45*time.Second, cfg.MaxIdleTimeout)
	tests.AssertEqual(t, 15*time.Second, cfg.KeepAlivePeriod)
	tests.AssertEqual(t, uint64(512*1024), cfg.InitialStreamReceiveWindow)
	tests.AssertEqual(t, uint64(8*1024*1024), cfg.MaxStreamReceiveWindow)
	tests.AssertEqual(t, uint64(1024*1024), cfg.InitialConnectionReceiveWindow)
	tests.AssertEqual(t, uint64(24*1024*1024), cfg.MaxConnectionReceiveWindow)
	tests.AssertEqual(t, int64(-1), cfg.MaxIncomingStreams)
	tests.AssertEqual(t, int64(100), cfg.MaxIncomingUniStreams)
	tests.AssertEqual(t, uint16(1200), cfg.InitialPacketSize)
	tests.AssertEqual(t, true, cfg.TokenStore != nil)
	tests.AssertEqual(t, true, cfg.EnableDatagrams)
	tests.AssertEqual(t, true, c.Transport.t3.QUICConfig.EnableDatagrams)

	clone := c.Clone()
	tests.AssertEqual(t, 15*time.Second, clone.Transport.http3QUICConfig.KeepAlivePeriod)
	tests.AssertEqual(t, true, clone.Transport.t3.QUICConfig.EnableDatagrams)
}

func hasTLSCurve(curves []tls.CurveID, target tls.CurveID) bool {
	for _, curve := range curves {
		if curve == target {
			return true
		}
	}
	return false
}

func TestHTTP3GreaseSettingUsesQUICVarInt(t *testing.T) {
	const maxVarInt = uint64(1<<62) - 1
	for i := 0; i < 100; i++ {
		id, value := http3GreaseSetting()
		tests.AssertEqual(t, true, id >= 0x21)
		tests.AssertEqual(t, uint64(0), (id-0x21)%0x1f)
		tests.AssertEqual(t, true, value <= maxVarInt)
	}
}

func TestHTTP2InitialStreamIDConfig(t *testing.T) {
	c := C().SetHTTP2InitialStreamID(3)
	tests.AssertEqual(t, uint32(3), c.Transport.t2.InitialStreamID)

	clone := c.Clone()
	tests.AssertEqual(t, uint32(3), clone.Transport.t2.InitialStreamID)
}

func captureProfileHeaders(t *testing.T, c *Client, method string) http.Header {
	t.Helper()
	var captured http.Header
	c.WrapRoundTripFunc(func(rt RoundTripper) RoundTripFunc {
		return func(r *Request) (*Response, error) {
			captured = r.Headers.Clone()
			return &Response{
				Request: r,
				Response: &http.Response{
					StatusCode: http.StatusNoContent,
					Status:     "204 No Content",
					Header:     make(http.Header),
					Body:       http.NoBody,
				},
			}, nil
		}
	})

	resp, err := c.R().Send(method, "https://example.com/")
	tests.AssertNoError(t, err)
	tests.AssertNotNil(t, resp)
	return captured
}

func TestImpersonateChromeAdvancedProfile(t *testing.T) {
	c := C().ImpersonateChromeWithOS(BrowserOSWindows)
	hdr := captureProfileHeaders(t, c, http.MethodGet)

	tests.AssertEqual(t, "gzip, deflate, br, zstd", hdr.Get("Accept-Encoding"))
	tests.AssertEqual(t, "en-US,en;q=0.9", hdr.Get("Accept-Language"))
	tests.AssertEqual(t, `"Windows"`, hdr.Get("Sec-Ch-Ua-Platform"))
	tests.AssertEqual(t, "?0", hdr.Get("Sec-Ch-Ua-Mobile"))
	tests.AssertEqual(t, "u=0, i", hdr.Get("Priority"))
	tests.AssertEqual(t, true, strings.Contains(hdr.Get("User-Agent"), "Windows NT 10.0"))
	tests.AssertEqual(t, true, strings.Contains(hdr.Get("User-Agent"), "Chrome/133.0.0.0"))
	tests.AssertEqual(t, "sec-ch-ua", hdr[HeaderOderKey][0])
	tests.AssertEqual(t, uint64(65536), c.Transport.http3AdditionalSettings[HTTP3SettingQpackMaxTableCapacity])
	tests.AssertEqual(t, uint64(100), c.Transport.http3AdditionalSettings[HTTP3SettingQpackBlockedStreams])
	tests.AssertEqual(t, true, c.Transport.http3EnableDatagrams)
	tests.AssertEqual(t, 262144, c.Transport.http3MaxResponseHeaderBytes)
	tests.AssertNotNil(t, c.Transport.http3TLSClientConfig)
	tests.AssertEqual(t, uint16(tls.VersionTLS13), c.Transport.http3TLSClientConfig.MinVersion)
	tests.AssertEqual(t, true, hasTLSCurve(c.Transport.http3TLSClientConfig.CurvePreferences, tls.X25519MLKEM768))
	tests.AssertNotNil(t, c.Transport.http3QUICConfig)
	tests.AssertEqual(t, 15*time.Second, c.Transport.http3QUICConfig.KeepAlivePeriod)
	tests.AssertEqual(t, true, c.Transport.http3QUICConfig.EnableDatagrams)
}

func TestImpersonateChromePostMobileProfile(t *testing.T) {
	c := C().ImpersonateChromeWithOS(BrowserOSAndroid)
	hdr := captureProfileHeaders(t, c, http.MethodPost)

	tests.AssertEqual(t, "?1", hdr.Get("Sec-Ch-Ua-Mobile"))
	tests.AssertEqual(t, `"Android"`, hdr.Get("Sec-Ch-Ua-Platform"))
	tests.AssertEqual(t, true, strings.Contains(hdr.Get("User-Agent"), "Android"))
	tests.AssertEqual(t, true, strings.Contains(hdr.Get("User-Agent"), "Mobile Safari"))
	tests.AssertEqual(t, "*/*", hdr.Get("Accept"))
	tests.AssertEqual(t, "u=1, i", hdr.Get("Priority"))
	tests.AssertEqual(t, "empty", hdr.Get("Sec-Fetch-Dest"))
	tests.AssertEqual(t, "content-length", hdr[HeaderOderKey][0])
}

func TestImpersonateChromeRandomOS(t *testing.T) {
	c := C().ImpersonateChromeRandomOS()
	hdr := captureProfileHeaders(t, c, http.MethodGet)
	tests.AssertEqual(t, true, strings.Contains(hdr.Get("User-Agent"), "133.0.0.0"))
	tests.AssertEqual(t, true, hdr.Get("Sec-Ch-Ua-Platform") != "")
}

func TestRandomBrowserOS(t *testing.T) {
	for i := 0; i < 20; i++ {
		switch RandomBrowserOS() {
		case BrowserOSWindows, BrowserOSMacOS, BrowserOSLinux, BrowserOSAndroid, BrowserOSIOS:
		default:
			t.Fatal("unexpected browser OS")
		}
	}
}

func TestImpersonateFirefoxAdvancedProfile(t *testing.T) {
	c := C().ImpersonateFirefoxWithOS(BrowserOSLinux)
	hdr := captureProfileHeaders(t, c, http.MethodGet)

	tests.AssertEqual(t, "gzip, deflate, br, zstd", hdr.Get("Accept-Encoding"))
	tests.AssertEqual(t, "en-US,en;q=0.5", hdr.Get("Accept-Language"))
	tests.AssertEqual(t, "u=0, i", hdr.Get("Priority"))
	tests.AssertEqual(t, true, strings.Contains(hdr.Get("User-Agent"), "X11; Linux x86_64"))
	tests.AssertEqual(t, "", hdr.Get("Sec-Ch-Ua"))
	tests.AssertEqual(t, uint32(3), c.Transport.t2.InitialStreamID)
	tests.AssertEqual(t, uint64(65536), c.Transport.http3AdditionalSettings[HTTP3SettingQpackMaxTableCapacity])
	tests.AssertEqual(t, uint64(20), c.Transport.http3AdditionalSettings[HTTP3SettingQpackBlockedStreams])
	tests.AssertEqual(t, uint64(1), c.Transport.http3AdditionalSettings[HTTP3SettingH3DatagramDraft])
	tests.AssertEqual(t, true, c.Transport.http3EnableDatagrams)
	tests.AssertEqual(t, true, c.Transport.http3EnableExtendedConnect)
	tests.AssertNotNil(t, c.Transport.http3TLSClientConfig)
	tests.AssertEqual(t, uint16(tls.VersionTLS13), c.Transport.http3TLSClientConfig.MinVersion)
	tests.AssertEqual(t, true, hasTLSCurve(c.Transport.http3TLSClientConfig.CurvePreferences, tls.X25519))
	tests.AssertNotNil(t, c.Transport.http3QUICConfig)
	tests.AssertEqual(t, 15*time.Second, c.Transport.http3QUICConfig.KeepAlivePeriod)
	tests.AssertEqual(t, true, c.Transport.http3QUICConfig.EnableDatagrams)
}

func TestImpersonateFirefoxHTTP3PseudoHeaderOrder(t *testing.T) {
	c := C().ImpersonateFirefox().EnableForceHTTP3()
	hdr := captureProfileHeaders(t, c, http.MethodGet)

	tests.AssertEqual(t, ":method", hdr[PseudoHeaderOderKey][0])
	tests.AssertEqual(t, ":scheme", hdr[PseudoHeaderOderKey][1])
	tests.AssertEqual(t, ":authority", hdr[PseudoHeaderOderKey][2])
	tests.AssertEqual(t, ":path", hdr[PseudoHeaderOderKey][3])
}

func TestImpersonateSwitchClearsChromeClientHints(t *testing.T) {
	c := C().ImpersonateChrome().ImpersonateFirefox()
	hdr := captureProfileHeaders(t, c, http.MethodGet)

	tests.AssertEqual(t, "", hdr.Get("Sec-Ch-Ua"))
	tests.AssertEqual(t, "", hdr.Get("Sec-Ch-Ua-Mobile"))
	tests.AssertEqual(t, true, strings.Contains(hdr.Get("User-Agent"), "Firefox/120.0"))
}
