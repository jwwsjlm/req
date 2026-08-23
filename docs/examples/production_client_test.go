package examples

import (
	"context"
	"errors"
	"net/http"
	"time"

	req "github.com/jwwsjlm/req/v3"
)

// newProductionClient returns a long-lived client with bounded response memory.
// Its common retry policy is deliberately limited to read-only methods; writes
// need request-level retries plus an application idempotency guarantee.
// newProductionClient 返回带响应内存上限的长期 client。公共重试仅用于只读方法；
// 写请求必须在具备业务幂等保证时单独配置重试。
func newProductionClient(baseURL string) *req.Client {
	return req.C().
		SetBaseURL(baseURL).
		SetTimeout(30*time.Second).
		SetCommonHeader("Accept", "application/json").
		SetMaxResponseSize(16<<20).
		SetCommonRetryCount(2).
		SetCommonRetryBackoffInterval(200*time.Millisecond, 2*time.Second).
		SetCommonRetryCondition(func(resp *req.Response, err error) bool {
			if !isReadOnlyRequest(resp) {
				return false
			}
			if err != nil {
				return !errors.Is(err, context.Canceled) &&
					!errors.Is(err, context.DeadlineExceeded)
			}
			return resp != nil && (resp.GetStatusCode() == http.StatusTooManyRequests ||
				resp.GetStatusCode() >= http.StatusInternalServerError)
		})
}

// isReadOnlyRequest keeps the common retry policy away from writes.
// isReadOnlyRequest 防止公共重试策略作用于写请求。
func isReadOnlyRequest(resp *req.Response) bool {
	if resp == nil || resp.Request == nil {
		return false
	}
	switch resp.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, "QUERY":
		return true
	default:
		return false
	}
}

func Example_productionClient() {
	client := newProductionClient("https://api.example.com")
	_ = client
	// Output:
}
