package req

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/jwwsjlm/req/v3/http2"
	utls "github.com/refraction-networking/utls"
)

// Identical for both Blink-based browsers (Chrome, Chromium, etc.) and WebKit-based browsers (Safari, etc.)
// Blink implementation: https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/renderer/platform/network/form_data_encoder.cc;drc=1d694679493c7b2f7b9df00e967b4f8699321093;l=130
// WebKit implementation: https://github.com/WebKit/WebKit/blob/47eea119fe9462721e5cc75527a4280c6d5f5214/Source/WebCore/platform/network/FormDataBuilder.cpp#L120
func webkitMultipartBoundaryFunc() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789AB"

	sb := strings.Builder{}
	sb.WriteString("----WebKitFormBoundary")

	for i := 0; i < 16; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters)-1)))
		if err != nil {
			panic(err)
		}

		sb.WriteByte(letters[index.Int64()])
	}

	return sb.String()
}

// Firefox implementation: https://searchfox.org/mozilla-central/source/dom/html/HTMLFormSubmission.cpp#355
func firefoxMultipartBoundaryFunc() string {
	sb := strings.Builder{}
	sb.WriteString("-------------------------")

	for i := 0; i < 3; i++ {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic(err)
		}
		u32 := binary.LittleEndian.Uint32(b[:])
		s := strconv.FormatUint(uint64(u32), 10)

		sb.WriteString(s)
	}

	return sb.String()
}

var (
	chromeHttp2Settings = []http2.Setting{
		{
			ID:  http2.SettingHeaderTableSize,
			Val: 65536,
		},
		{
			ID:  http2.SettingEnablePush,
			Val: 0,
		},
		{
			ID:  http2.SettingInitialWindowSize,
			Val: 6291456,
		},
		{
			ID:  http2.SettingMaxHeaderListSize,
			Val: 262144,
		},
	}

	chromePseudoHeaderOrder = map[string][]string{
		http.MethodGet: {
			":method",
			":authority",
			":scheme",
			":path",
		},
		http.MethodPost: {
			":method",
			":authority",
			":scheme",
			":path",
		},
	}

	chromeHeaderOrder = map[string][]string{
		http.MethodGet: {
			"sec-ch-ua",
			"sec-ch-ua-mobile",
			"sec-ch-ua-platform",
			"authorization",
			"upgrade-insecure-requests",
			"user-agent",
			"accept",
			"sec-fetch-site",
			"sec-fetch-mode",
			"sec-fetch-user",
			"sec-fetch-dest",
			"referer",
			"accept-encoding",
			"accept-language",
			"cookie",
			"priority",
		},
		http.MethodPost: {
			"content-length",
			"pragma",
			"cache-control",
			"sec-ch-ua-platform",
			"authorization",
			"user-agent",
			"sec-ch-ua",
			"content-type",
			"sec-ch-ua-mobile",
			"accept",
			"origin",
			"sec-fetch-site",
			"sec-fetch-mode",
			"sec-fetch-dest",
			"referer",
			"accept-encoding",
			"accept-language",
			"cookie",
			"priority",
		},
	}

	chromeUserAgentByOS = map[BrowserOS]string{
		BrowserOSWindows: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		BrowserOSMacOS:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		BrowserOSLinux:   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		BrowserOSAndroid: "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Mobile Safari/537.36",
		BrowserOSIOS:     "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/133.0.0.0 Mobile/15E148 Safari/604.1",
	}

	chromePlatformByOS = map[BrowserOS]string{
		BrowserOSWindows: `"Windows"`,
		BrowserOSMacOS:   `"macOS"`,
		BrowserOSLinux:   `"Linux"`,
		BrowserOSAndroid: `"Android"`,
		BrowserOSIOS:     `"iOS"`,
	}

	chromeBaseHeaders = map[string]string{
		"sec-ch-ua":       `"Not:A-Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`,
		"accept-encoding": "gzip, deflate, br, zstd",
		"accept-language": "en-US,en;q=0.9",
	}

	chromeGetHeaders = map[string]string{
		"upgrade-insecure-requests": "1",
		"accept":                    "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"sec-fetch-site":            "none",
		"sec-fetch-mode":            "navigate",
		"sec-fetch-user":            "?1",
		"sec-fetch-dest":            "document",
		"priority":                  "u=0, i",
	}

	chromePostHeaders = map[string]string{
		"pragma":         "no-cache",
		"cache-control":  "no-cache",
		"accept":         "*/*",
		"sec-fetch-site": "same-origin",
		"sec-fetch-mode": "cors",
		"sec-fetch-dest": "empty",
		"priority":       "u=1, i",
	}

	chromeHeaderPriority = http2.PriorityParam{
		StreamDep: 0,
		Exclusive: true,
		Weight:    255,
	}
)

func chromeBrowserProfile(os BrowserOS) *browserHeaderProfile {
	os = normalizeBrowserOS(os)
	baseHeaders := cloneMap(chromeBaseHeaders)
	baseHeaders["sec-ch-ua-mobile"] = "?0"
	if browserOSIsMobile(os) {
		baseHeaders["sec-ch-ua-mobile"] = "?1"
	}
	baseHeaders["sec-ch-ua-platform"] = chromePlatformByOS[os]
	baseHeaders["user-agent"] = chromeUserAgentByOS[os]
	return &browserHeaderProfile{
		baseHeaders:        baseHeaders,
		methodHeaders:      map[string]map[string]string{http.MethodGet: chromeGetHeaders, http.MethodPost: chromePostHeaders},
		headerOrders:       chromeHeaderOrder,
		pseudoHeaderOrders: chromePseudoHeaderOrder,
	}
}

// ImpersonateChrome impersonates Chrome browser (version aligned with uTLS Chrome 133).
func (c *Client) ImpersonateChrome() *Client {
	return c.ImpersonateChromeWithOS(BrowserOSMacOS)
}

// ImpersonateChromeWithOS impersonates Chrome browser with the specified OS profile.
func (c *Client) ImpersonateChromeWithOS(os BrowserOS) *Client {
	profile := chromeBrowserProfile(os)
	c.setBrowserProfile(profile)
	c.
		SetTLSFingerprint(utls.HelloChrome_Auto).
		SetHTTP2SettingsFrame(chromeHttp2Settings...).
		SetHTTP2ConnectionFlow(15663105).
		SetHTTP2HeaderPriority(chromeHeaderPriority).
		SetHTTP3TLSChromeProfile().
		SetHTTP3QUICChromeProfile().
		SetHTTP3AdditionalSettings(map[uint64]uint64{
			HTTP3SettingQpackMaxTableCapacity: 65536,
			HTTP3SettingQpackBlockedStreams:   100,
		}).
		SetHTTP3MaxResponseHeaderBytes(262144).
		EnableHTTP3Datagrams().
		SetHTTP3Grease().
		SetMultipartBoundaryFunc(webkitMultipartBoundaryFunc)
	return c
}

// ImpersonateChromeRandomOS impersonates Chrome with a random OS profile.
func (c *Client) ImpersonateChromeRandomOS() *Client {
	return c.ImpersonateChromeWithOS(BrowserOSRandom)
}

var (
	firefoxHttp2Settings = []http2.Setting{
		{
			ID:  http2.SettingHeaderTableSize,
			Val: 65536,
		},
		{
			ID:  http2.SettingEnablePush,
			Val: 0,
		},
		{
			ID:  http2.SettingInitialWindowSize,
			Val: 131072,
		},
		{
			ID:  http2.SettingMaxFrameSize,
			Val: 16384,
		},
	}

	firefoxPseudoHeaderOrder = map[string][]string{
		http.MethodGet: {
			":method",
			":path",
			":authority",
			":scheme",
		},
		http.MethodPost: {
			":method",
			":path",
			":authority",
			":scheme",
		},
	}

	firefoxPseudoHeaderOrderHTTP3 = map[string][]string{
		http.MethodGet: {
			":method",
			":scheme",
			":authority",
			":path",
		},
		http.MethodPost: {
			":method",
			":scheme",
			":authority",
			":path",
		},
	}

	firefoxHeaderOrder = map[string][]string{
		http.MethodGet: {
			"user-agent",
			"accept",
			"accept-language",
			"accept-encoding",
			"referer",
			"authorization",
			"cookie",
			"upgrade-insecure-requests",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"priority",
		},
		http.MethodPost: {
			"user-agent",
			"accept",
			"accept-language",
			"accept-encoding",
			"referer",
			"content-type",
			"authorization",
			"content-length",
			"origin",
			"cookie",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"priority",
			"pragma",
			"cache-control",
		},
	}

	firefoxUserAgentByOS = map[BrowserOS]string{
		BrowserOSWindows: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
		BrowserOSMacOS:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0",
		BrowserOSLinux:   "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
		BrowserOSAndroid: "Mozilla/5.0 (Android 14; Mobile; rv:120.0) Gecko/120.0 Firefox/120.0",
		BrowserOSIOS:     "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/120.0 Mobile/15E148 Safari/605.1.15",
	}

	firefoxBaseHeaders = map[string]string{
		"accept-language": "en-US,en;q=0.5",
		"accept-encoding": "gzip, deflate, br, zstd",
	}

	firefoxGetHeaders = map[string]string{
		"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"upgrade-insecure-requests": "1",
		"sec-fetch-dest":            "document",
		"sec-fetch-mode":            "navigate",
		"sec-fetch-site":            "none",
		"sec-fetch-user":            "?1",
		"priority":                  "u=0, i",
	}

	firefoxPostHeaders = map[string]string{
		"accept":         "*/*",
		"cache-control":  "no-cache",
		"pragma":         "no-cache",
		"sec-fetch-dest": "empty",
		"sec-fetch-mode": "cors",
		"sec-fetch-site": "same-origin",
		"priority":       "u=1, i",
	}

	firefoxHeaderPriority = http2.PriorityParam{
		StreamDep: 0,
		Exclusive: false,
		Weight:    41,
	}
)

func firefoxBrowserProfile(os BrowserOS) *browserHeaderProfile {
	os = normalizeBrowserOS(os)
	baseHeaders := cloneMap(firefoxBaseHeaders)
	baseHeaders["user-agent"] = firefoxUserAgentByOS[os]
	return &browserHeaderProfile{
		baseHeaders:         baseHeaders,
		methodHeaders:       map[string]map[string]string{http.MethodGet: firefoxGetHeaders, http.MethodPost: firefoxPostHeaders},
		headerOrders:        firefoxHeaderOrder,
		pseudoHeaderOrders:  firefoxPseudoHeaderOrder,
		pseudoHeaderOrders3: firefoxPseudoHeaderOrderHTTP3,
	}
}

// ImpersonateFirefox impersonates Firefox browser (version 120).
func (c *Client) ImpersonateFirefox() *Client {
	return c.ImpersonateFirefoxWithOS(BrowserOSMacOS)
}

// ImpersonateFirefoxWithOS impersonates Firefox browser with the specified OS profile.
func (c *Client) ImpersonateFirefoxWithOS(os BrowserOS) *Client {
	profile := firefoxBrowserProfile(os)
	c.setBrowserProfile(profile)
	c.
		SetTLSFingerprint(utls.HelloFirefox_120).
		SetHTTP2SettingsFrame(firefoxHttp2Settings...).
		SetHTTP2ConnectionFlow(12517377).
		SetHTTP2InitialStreamID(3).
		SetHTTP2HeaderPriority(firefoxHeaderPriority).
		SetHTTP3TLSFirefoxProfile().
		SetHTTP3QUICPerformanceProfile().
		SetHTTP3AdditionalSettings(map[uint64]uint64{
			HTTP3SettingQpackMaxTableCapacity: 65536,
			HTTP3SettingQpackBlockedStreams:   20,
			HTTP3SettingEnableWebTransport:    0,
			HTTP3SettingH3DatagramDraft:       1,
		}).
		EnableHTTP3Datagrams().
		EnableHTTP3ExtendedConnect().
		SetMultipartBoundaryFunc(firefoxMultipartBoundaryFunc)
	return c
}

// ImpersonateFirefoxRandomOS impersonates Firefox with a random OS profile.
func (c *Client) ImpersonateFirefoxRandomOS() *Client {
	return c.ImpersonateFirefoxWithOS(BrowserOSRandom)
}

var (
	safariHttp2Settings = []http2.Setting{
		{
			ID:  http2.SettingInitialWindowSize,
			Val: 4194304,
		},
		{
			ID:  http2.SettingMaxConcurrentStreams,
			Val: 100,
		},
	}

	safariPseudoHeaderOrder = []string{
		":method",
		":scheme",
		":path",
		":authority",
	}

	safariHeaderOrder = []string{
		"accept",
		"sec-fetch-site",
		"cookie",
		"sec-fetch-dest",
		"accept-language",
		"sec-fetch-mode",
		"user-agent",
		"referer",
		"accept-encoding",
	}

	safariHeaders = map[string]string{
		"accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"sec-fetch-site":  "same-origin",
		"sec-fetch-dest":  "document",
		"accept-language": "zh-CN,zh-Hans;q=0.9",
		"sec-fetch-mode":  "navigate",
		"user-agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
	}

	safariHeaderPriority = http2.PriorityParam{
		StreamDep: 0,
		Exclusive: false,
		Weight:    254,
	}
)

// ImpersonateSafari impersonates Safari browser (version 16.6).
func (c *Client) ImpersonateSafari() *Client {
	c.browserProfile = nil
	c.clearBrowserProfileHeaders()
	c.
		SetTLSFingerprint(utls.HelloSafari_16_0).
		SetHTTP2SettingsFrame(safariHttp2Settings...).
		SetHTTP2ConnectionFlow(10485760).
		SetCommonPseudoHeaderOder(safariPseudoHeaderOrder...).
		SetCommonHeaderOrder(safariHeaderOrder...).
		SetCommonHeaders(safariHeaders).
		SetHTTP2HeaderPriority(safariHeaderPriority).
		SetMultipartBoundaryFunc(webkitMultipartBoundaryFunc)
	return c
}
