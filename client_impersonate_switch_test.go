package req

import (
	"net/http"
	"testing"

	"github.com/jwwsjlm/req/v3/internal/tests"
)

func TestImpersonateFirefoxToChromeClearsTransportProfile(t *testing.T) {
	c := C().ImpersonateFirefox().ImpersonateChromeWithOS(BrowserOSWindows)
	hdr := captureProfileHeaders(t, c, http.MethodGet)

	tests.AssertEqual(t, uint32(0), c.Transport.t2.InitialStreamID)
	tests.AssertEqual(t, false, c.Transport.http3EnableExtendedConnect)
	tests.AssertEqual(t, uint64(100), c.Transport.http3AdditionalSettings[HTTP3SettingQpackBlockedStreams])
	_, hasFirefoxDraftDatagram := c.Transport.http3AdditionalSettings[HTTP3SettingH3DatagramDraft]
	tests.AssertEqual(t, false, hasFirefoxDraftDatagram)
	tests.AssertEqual(t, "sec-ch-ua", hdr[HeaderOderKey][0])
}

func TestImpersonateSafariToChromeClearsHeaderOrder(t *testing.T) {
	c := C().ImpersonateSafari().ImpersonateChrome()
	hdr := captureProfileHeaders(t, c, http.MethodGet)

	tests.AssertEqual(t, "sec-ch-ua", hdr[HeaderOderKey][0])
	tests.AssertEqual(t, ":method", hdr[PseudoHeaderOderKey][0])
	tests.AssertEqual(t, ":authority", hdr[PseudoHeaderOderKey][1])
}

func TestImpersonateChromeToSafariClearsHTTP3Profile(t *testing.T) {
	c := C().ImpersonateChrome().ImpersonateSafari()

	tests.AssertEqual(t, 0, len(c.Transport.http3AdditionalSettings))
	tests.AssertEqual(t, false, c.Transport.http3EnableDatagrams)
	tests.AssertEqual(t, false, c.Transport.http3EnableExtendedConnect)
	tests.AssertEqual(t, 0, c.Transport.http3MaxResponseHeaderBytes)
	tests.AssertIsNil(t, c.Transport.http3QUICConfig)
	tests.AssertIsNil(t, c.Transport.http3TLSClientConfig)
	tests.AssertEqual(t, uint32(0), c.Transport.t2.InitialStreamID)
}
