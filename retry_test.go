package req

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jwwsjlm/req/v3/internal/tests"
)

type trackingBody struct {
	*strings.Reader
	closed *int32
}

func (b *trackingBody) Close() error {
	atomic.AddInt32(b.closed, 1)
	return nil
}

func TestRetryBackOff(t *testing.T) {
	testRetry(t, func(r *Request) {
		r.SetRetryBackoffInterval(10*time.Millisecond, 1*time.Second)
	})
}

func testRetry(t *testing.T, setFunc func(r *Request)) {
	attempt := 0
	r := tc().R().
		SetRetryCount(3).
		SetRetryCondition(func(resp *Response, err error) bool {
			return (err != nil) || (resp.StatusCode == http.StatusTooManyRequests)
		}).
		SetRetryHook(func(resp *Response, err error) {
			attempt++
		})
	setFunc(r)
	resp, err := r.Get("/too-many")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, 3, resp.Request.RetryAttempt)
	tests.AssertEqual(t, 3, attempt)
}

func TestRetryInterval(t *testing.T) {
	testRetry(t, func(r *Request) {
		r.SetRetryInterval(func(resp *Response, attempt int) time.Duration {
			sleep := 0.01 * math.Exp2(float64(attempt))
			return time.Duration(math.Min(2, sleep)) * time.Second
		})
	})
}

func TestRetryFixedInterval(t *testing.T) {
	testRetry(t, func(r *Request) {
		r.SetRetryFixedInterval(1 * time.Millisecond)
	})
}

func TestAddRetryHook(t *testing.T) {
	test := "test1"
	testRetry(t, func(r *Request) {
		r.AddRetryHook(func(resp *Response, err error) {
			test = "test2"
		})
	})
	tests.AssertEqual(t, "test2", test)
}

func TestRetryOverride(t *testing.T) {
	c := tc().
		SetCommonRetryCount(3).
		SetCommonRetryHook(func(resp *Response, err error) {}).
		AddCommonRetryHook(func(resp *Response, err error) {}).
		SetCommonRetryCondition(func(resp *Response, err error) bool {
			return false
		}).SetCommonRetryBackoffInterval(1*time.Millisecond, 10*time.Millisecond)
	test := "test"
	resp, err := c.R().SetRetryFixedInterval(2 * time.Millisecond).
		SetRetryCount(2).
		SetRetryHook(func(resp *Response, err error) {
			test = "test1"
		}).SetRetryCondition(func(resp *Response, err error) bool {
		return err != nil || resp.StatusCode == http.StatusTooManyRequests
	}).Get("/too-many")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, "test1", test)
	tests.AssertEqual(t, 2, resp.Request.RetryAttempt)
}

func TestAddRetryCondition(t *testing.T) {
	attempt := 0
	resp, err := tc().R().
		SetRetryCount(3).
		AddRetryCondition(func(resp *Response, err error) bool {
			return err != nil
		}).
		AddRetryCondition(func(resp *Response, err error) bool {
			return resp.StatusCode == http.StatusServiceUnavailable
		}).
		SetRetryHook(func(resp *Response, err error) {
			attempt++
		}).Get("/too-many")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, 0, attempt)
	tests.AssertEqual(t, 0, resp.Request.RetryAttempt)

	attempt = 0
	resp, err = tc().
		SetCommonRetryCount(3).
		AddCommonRetryCondition(func(resp *Response, err error) bool {
			return err != nil
		}).
		AddCommonRetryCondition(func(resp *Response, err error) bool {
			return resp.StatusCode == http.StatusServiceUnavailable
		}).
		SetCommonRetryHook(func(resp *Response, err error) {
			attempt++
		}).R().Get("/too-many")
	tests.AssertNoError(t, err)
	tests.AssertEqual(t, 0, attempt)
	tests.AssertEqual(t, 0, resp.Request.RetryAttempt)

}

func TestRetryWithUnreplayableBody(t *testing.T) {
	_, err := tc().R().
		SetRetryCount(1).
		SetBody(bytes.NewBufferString("test")).
		Post("/")
	tests.AssertEqual(t, errRetryableWithUnReplayableBody, err)

	_, err = tc().R().
		SetRetryCount(1).
		SetBody(io.NopCloser(bytes.NewBufferString("test"))).
		Post("/")
	tests.AssertEqual(t, errRetryableWithUnReplayableBody, err)
}

func TestRetryWithSetResult(t *testing.T) {
	headers := make(http.Header)
	resp, err := tc().SetCommonCookies(&http.Cookie{
		Name:  "test",
		Value: "test",
	}).R().
		SetRetryCount(1).
		SetResult(&headers).
		Get("/header")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, "test=test", headers.Get("Cookie"))
}

func TestRetryWithModify(t *testing.T) {
	tokens := []string{"badtoken1", "badtoken2", "goodtoken"}
	tokenIndex := 0
	c := tc().
		SetCommonRetryCount(2).
		SetCommonRetryHook(func(resp *Response, err error) {
			tokenIndex++
			resp.Request.SetBearerAuthToken(tokens[tokenIndex])
		}).SetCommonRetryCondition(func(resp *Response, err error) bool {
		return err != nil || resp.StatusCode == http.StatusUnauthorized
	})

	resp, err := c.R().
		SetBearerAuthToken(tokens[tokenIndex]).
		Get("/protected")
	assertSuccess(t, resp, err)
	tests.AssertEqual(t, 2, resp.Request.RetryAttempt)
}

func TestRetryFalse(t *testing.T) {
	resp, err := tc().SetTimeout(2 * time.Second).R().
		SetRetryCount(1).
		SetRetryCondition(func(resp *Response, err error) bool {
			return false
		}).Get("https://non-exists-host.com.cn")
	tests.AssertNotNil(t, err)
	tests.AssertIsNil(t, resp.Response)
	tests.AssertEqual(t, 0, resp.Request.RetryAttempt)
}

func TestRetryClosesPreviousResponseBody(t *testing.T) {
	var retryBodyClosed int32
	var finalBodyClosed int32
	var attempts int32

	c := C()
	c.Transport.WrapRoundTripFunc(func(rt http.RoundTripper) HttpRoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			attempt := atomic.AddInt32(&attempts, 1)
			statusCode := http.StatusServiceUnavailable
			closed := &retryBodyClosed
			if attempt > 1 {
				statusCode = http.StatusOK
				closed = &finalBodyClosed
			}
			body := "retry body"
			return &http.Response{
				StatusCode:    statusCode,
				Status:        fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
				Header:        make(http.Header),
				Body:          &trackingBody{Reader: strings.NewReader(body), closed: closed},
				ContentLength: int64(len(body)),
				Request:       req,
			}, nil
		}
	})

	resp, err := c.R().
		SetRetryCount(1).
		SetRetryFixedInterval(0).
		SetRetryCondition(func(resp *Response, err error) bool {
			return err != nil || resp.StatusCode == http.StatusServiceUnavailable
		}).
		Get("http://example.com")

	assertSuccess(t, resp, err)
	tests.AssertEqual(t, int32(2), atomic.LoadInt32(&attempts))
	tests.AssertEqual(t, int32(1), atomic.LoadInt32(&retryBodyClosed))
	tests.AssertEqual(t, int32(1), atomic.LoadInt32(&finalBodyClosed))
}

func TestRetryBackoffStopsWhenContextCancelled(t *testing.T) {
	var attempts int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := C()
	c.Transport.WrapRoundTripFunc(func(rt http.RoundTripper) HttpRoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&attempts, 1)
			body := "retry body"
			return &http.Response{
				StatusCode:    http.StatusServiceUnavailable,
				Status:        fmt.Sprintf("%d %s", http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable)),
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader(body)),
				ContentLength: int64(len(body)),
				Request:       req,
			}, nil
		}
	})

	start := time.Now()
	resp, err := c.R().
		SetContext(ctx).
		SetRetryCount(3).
		SetRetryFixedInterval(time.Second).
		SetRetryCondition(func(resp *Response, err error) bool {
			return resp.StatusCode == http.StatusServiceUnavailable
		}).
		SetRetryHook(func(resp *Response, err error) {
			cancel()
		}).
		Get("http://example.com")
	elapsed := time.Since(start)

	tests.AssertNotNil(t, err)
	tests.AssertErrorContains(t, err, "context canceled")
	tests.AssertEqual(t, int32(1), atomic.LoadInt32(&attempts))
	tests.AssertEqual(t, 1, resp.Request.RetryAttempt)
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("retry backoff waited too long after context cancellation: %s", elapsed)
	}
}

func TestRetryTurnedOffWhenRetryCountEqZero(t *testing.T) {
	resp, err := tc().SetTimeout(2 * time.Second).R().
		SetRetryCount(0).
		SetRetryCondition(func(resp *Response, err error) bool {
			t.Fatal("retry condition should not be executed")
			return true
		}).
		Get("https://non-exists-host.com.cn")
	tests.AssertNotNil(t, err)
	tests.AssertIsNil(t, resp.Response)
	tests.AssertEqual(t, 0, resp.Request.RetryAttempt)

	resp, err = tc().SetTimeout(2 * time.Second).
		SetCommonRetryCount(0).
		SetCommonRetryCondition(func(resp *Response, err error) bool {
			t.Fatal("retry condition should not be executed")
			return true
		}).
		R().
		Get("https://non-exists-host.com.cn")
	tests.AssertNotNil(t, err)
	tests.AssertIsNil(t, resp.Response)
	tests.AssertEqual(t, 0, resp.Request.RetryAttempt)
}
