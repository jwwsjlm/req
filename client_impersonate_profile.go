package req

import (
	"crypto/rand"
	"math/big"
	"net/http"
)

// BrowserOS identifies the operating system used by a browser impersonation profile.
type BrowserOS string

const (
	BrowserOSWindows BrowserOS = "windows"
	BrowserOSMacOS   BrowserOS = "macos"
	BrowserOSLinux   BrowserOS = "linux"
	BrowserOSAndroid BrowserOS = "android"
	BrowserOSIOS     BrowserOS = "ios"
	BrowserOSRandom  BrowserOS = "random"
)

var browserOSValues = []BrowserOS{
	BrowserOSWindows,
	BrowserOSMacOS,
	BrowserOSLinux,
	BrowserOSAndroid,
	BrowserOSIOS,
}

type browserHeaderProfile struct {
	baseHeaders         map[string]string
	methodHeaders       map[string]map[string]string
	headerOrders        map[string][]string
	pseudoHeaderOrders  map[string][]string
	pseudoHeaderOrders3 map[string][]string
}

var browserProfileHeaderKeys = []string{
	HeaderOderKey,
	PseudoHeaderOderKey,
	"pragma",
	"cache-control",
	"sec-ch-ua",
	"sec-ch-ua-mobile",
	"sec-ch-ua-platform",
	"upgrade-insecure-requests",
	"user-agent",
	"accept",
	"accept-encoding",
	"accept-language",
	"sec-fetch-site",
	"sec-fetch-mode",
	"sec-fetch-user",
	"sec-fetch-dest",
	"priority",
}

func (c *Client) setBrowserProfile(profile *browserHeaderProfile) {
	c.resetBrowserTransportProfile()
	c.browserProfile = profile
	c.clearBrowserProfileHeaders()
	c.SetCommonHeaders(profile.baseHeaders)
}

// resetBrowserTransportProfile clears transport settings owned by a browser
// profile before another profile is applied. It only affects future
// connections; callers should configure impersonation before sending requests.
// resetBrowserTransportProfile 会在切换浏览器 profile 前清理其拥有的传输层状态；
// 已建立的连接不会被改写，因此应在发送请求前完成伪装配置。
func (c *Client) resetBrowserTransportProfile() {
	c.
		SetHTTP2InitialStreamID(0).
		SetHTTP2PriorityFrames().
		SetHTTP3AdditionalSettings(nil).
		SetHTTP3MaxResponseHeaderBytes(0).
		SetHTTP3QUICConfig(nil).
		SetHTTP3TLSClientConfig(nil).
		DisableHTTP3Datagrams().
		DisableHTTP3ExtendedConnect()
}

func (c *Client) clearBrowserProfileHeaders() {
	if c.Headers == nil {
		c.Headers = make(http.Header)
	}
	for _, key := range browserProfileHeaderKeys {
		c.Headers.Del(key)
	}
}

func normalizeBrowserOS(os BrowserOS) BrowserOS {
	switch os {
	case BrowserOSWindows, BrowserOSMacOS, BrowserOSLinux, BrowserOSAndroid, BrowserOSIOS:
		return os
	case BrowserOSRandom:
		return RandomBrowserOS()
	default:
		return BrowserOSMacOS
	}
}

// RandomBrowserOS returns a random browser OS profile.
func RandomBrowserOS() BrowserOS {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(browserOSValues))))
	if err != nil {
		return BrowserOSMacOS
	}
	return browserOSValues[n.Int64()]
}

func browserOSIsMobile(os BrowserOS) bool {
	return os == BrowserOSAndroid || os == BrowserOSIOS
}

func browserHeaderClass(method string) string {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return http.MethodPost
	default:
		return http.MethodGet
	}
}

func applyBrowserProfileHeaders(c *Client, r *Request) error {
	if c.browserProfile == nil {
		return nil
	}
	c.browserProfile.apply(c, r)
	return nil
}

func (p *browserHeaderProfile) apply(c *Client, r *Request) {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}

	method := browserHeaderClass(r.Method)
	setHeadersIfAbsent(r.Headers, p.baseHeaders)
	setHeadersIfAbsent(r.Headers, p.methodHeaders[method])

	if len(r.Headers[HeaderOderKey]) == 0 {
		if order := p.headerOrders[method]; len(order) > 0 {
			r.Headers[HeaderOderKey] = cloneSlice(order)
		}
	}

	pseudoOrders := p.pseudoHeaderOrders
	if c.Transport != nil && c.Transport.forceHttpVersion == h3 && len(p.pseudoHeaderOrders3) > 0 {
		pseudoOrders = p.pseudoHeaderOrders3
	}
	if len(r.Headers[PseudoHeaderOderKey]) == 0 {
		if order := pseudoOrders[method]; len(order) > 0 {
			r.Headers[PseudoHeaderOderKey] = cloneSlice(order)
		}
	}
}

func setHeadersIfAbsent(headers http.Header, values map[string]string) {
	for key, value := range values {
		if len(headers.Values(key)) == 0 {
			headers.Set(key, value)
		}
	}
}
