package examples

import (
	"context"
	"net"
	"time"

	req "github.com/jwwsjlm/req/v3"
)

// newCustomNetworkClient demonstrates the resolver and dial extension points.
// Usually choose either SetDNSResolver/SetResolver or SetDial; a custom dialer
// is responsible for applying any DNS policy it needs.
// newCustomNetworkClient 展示 resolver 与 dial 扩展点。通常在
// SetDNSResolver/SetResolver 和 SetDial 中择一；自定义 dialer 需自行落实 DNS 策略。
func newCustomNetworkClient() *req.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return req.C().SetDial(func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, address)
	})
}

func Example_customNetwork() {
	client := newCustomNetworkClient()
	_ = client
	// Output:
}
