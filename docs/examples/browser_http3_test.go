package examples

import (
	"time"

	req "github.com/jwwsjlm/req/v3"
)

// newBrowserClient configures a coherent browser profile and enables HTTP/3
// discovery with fallback. Avoid forcing HTTP/3 unless the endpoint is known to
// support it and failure without fallback is desired.
// newBrowserClient 配置完整浏览器 profile，并启用带回退的 HTTP/3 探测；
// 只有确认目标支持 H3 且希望失败直接返回时才应强制 HTTP/3。
func newBrowserClient() *req.Client {
	client := req.C().ImpersonateChromeWithOS(req.BrowserOSWindows)
	client.Transport.EnableHTTP3()
	client.Transport.
		EnableHTTP3FallbackOnError().
		SetHTTP3AltSvcFailureCooldown(30 * time.Second)
	return client
}

func Example_browserHTTP3() {
	client := newBrowserClient()
	_ = client
	// Output:
}
