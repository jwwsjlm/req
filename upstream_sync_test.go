package req

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	utls "github.com/refraction-networking/utls"
)

// Digest 接受带空白的 qop 列表和未知指令，但拒绝缺少 nonce 的挑战。
func TestDigestChallengeCompatibility(t *testing.T) {
	for _, tc := range []struct {
		challenge string
		wantErr   bool
	}{
		{`Digest realm="test", nonce="n", qop="future, auth", extension="ignored"`, false},
		{`Digest realm="test", nonce="n", qop="auth, auth-int"`, false},
		{`Digest realm="test", qop="auth"`, true},
		{`Digest extension="ignored"`, true},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				w.Header().Set("WWW-Authenticate", tc.challenge)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !strings.Contains(r.Header.Get("Authorization"), "qop=auth") {
				w.WriteHeader(http.StatusBadRequest)
			}
		}))
		c := C().SetCommonDigestAuth("user", "password")
		resp, err := c.R().Get(server.URL)
		c.CloseIdleConnections()
		server.Close()
		if (err != nil) != tc.wantErr || (!tc.wantErr && resp.StatusCode != http.StatusOK) {
			t.Errorf("challenge %q: response=%v error=%v", tc.challenge, resp, err)
		}
	}
}

// 空路径保留基础 URL 的尾斜杠，非空路径保持已有拼接语义。
func TestBaseURLPreservesTrailingSlash(t *testing.T) {
	for _, tc := range []struct{ base, path, want string }{
		{"https://example.test/api/", "", "https://example.test/api/"},
		{"https://example.test/api", "", "https://example.test/api"},
		{"https://example.test/api/", "/", "https://example.test/api/"},
		{"https://example.test/api/", "users", "https://example.test/api/users"},
		{"https://example.test/api/", "/users", "https://example.test/api/users"},
		{"https://example.test/api/", "?page=2", "https://example.test/api/?page=2"},
		{"https://example.test/api/", "https://other.test/", "https://other.test/"},
	} {
		c := C().SetBaseURL(tc.base)
		r := c.R()
		r.RawURL = tc.path
		if err := parseRequestURL(c, r); err != nil {
			t.Fatal(err)
		}
		if got := r.URL.String(); got != tc.want {
			t.Errorf("base %q path %q: got %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

// 预留容量会暴露共享切片问题：两个请求的 Add 不应互相覆盖。
func TestRequestHeadersDoNotAliasClient(t *testing.T) {
	c := C()
	c.Headers = http.Header{"X-Values": make([]string, 3, 4)}
	copy(c.Headers["X-Values"], []string{"a", "b", "c"})
	first, second := c.R(), c.R()
	for _, r := range []*Request{first, second} {
		if err := parseRequestHeader(c, r); err != nil {
			t.Fatal(err)
		}
	}
	first.Headers.Add("X-Values", "first")
	second.Headers.Add("X-Values", "second")
	first.Headers["X-Values"][0] = "changed"
	if first.Headers["X-Values"][3] != "first" || second.Headers["X-Values"][0] != "a" || c.Headers["X-Values"][0] != "a" {
		t.Fatal("request headers share client or sibling backing storage")
	}
}

// 校验实际编码后的 ClientHello；扩展顺序逐次变化，根证书标识字节保持固定。
func TestChrome152ClientHello(t *testing.T) {
	orders := make(map[string]bool)
	for range 5 {
		uconn := utls.UClient(nil, &utls.Config{ServerName: "example.test", OmitEmptyPsk: true}, utls.HelloCustom)
		if err := uconn.ApplyPreset(chrome152Spec()); err != nil {
			t.Fatal(err)
		}
		if err := uconn.BuildHandshakeState(); err != nil {
			t.Fatal(err)
		}
		raw := uconn.HandshakeState.Hello.Raw
		record := append([]byte{22, 3, 1, byte(len(raw) >> 8), byte(len(raw))}, raw...)
		fp := utls.Fingerprinter{AllowBluntMimicry: true}
		spec, err := fp.RawClientHello(record)
		if err != nil {
			t.Fatal(err)
		}
		var order string
		var anchorsFound, signaturesFound bool
		for _, ext := range spec.Extensions {
			order += fmt.Sprintf("%T,", ext)
			switch ext := ext.(type) {
			case *utls.GenericExtension:
				if ext.Id == 0xca34 {
					anchorsFound = true
					if len(ext.Data) != 206 || binary.BigEndian.Uint16(ext.Data) != 204 || fmt.Sprintf("%x", sha256.Sum256(ext.Data)) != "fe2dad53a54c3bed0033dbf687247b4654627612c27dfb235a78dc927cf7234a" {
						t.Fatal("Chrome 152 trust-anchor bytes changed")
					}
				}
			case *utls.SignatureAlgorithmsExtension:
				signaturesFound = true
				sig := ext.SupportedSignatureAlgorithms
				if len(sig) != 12 || uint16(sig[0])&0x0f0f != 0x0a0a || sig[1] != 0x0904 || sig[2] != 0x0905 || sig[3] != 0x0906 {
					t.Fatalf("unexpected signature algorithms: %v", sig)
				}
			}
		}
		if !anchorsFound || !signaturesFound {
			t.Fatal("Chrome 152 extensions are missing")
		}
		orders[order] = true
	}
	if len(orders) == 1 {
		t.Fatal("extension order stayed fixed across handshakes")
	}
}

// 用本地 TLS 服务验证新版 preset、协商曲线状态及 Chrome 的第二次连接恢复。
func TestUpdatedBrowserTLSProfiles(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "browser profile")
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	for _, profile := range []struct {
		name  string
		apply func(*Client) *Client
	}{
		{"chrome", (*Client).ImpersonateChrome},
		{"firefox", (*Client).ImpersonateFirefox},
		{"safari", (*Client).ImpersonateSafari},
	} {
		t.Run(profile.name, func(t *testing.T) {
			c := profile.apply(C())
			c.Transport.SetTLSClientConfig(&tls.Config{
				RootCAs: roots, ClientSessionCache: tls.NewLRUClientSessionCache(4),
			})
			c.Transport.DisableKeepAlives = true
			defer c.CloseIdleConnections()
			for attempt := range 2 {
				resp, err := c.R().Get(server.URL)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(resp.Bytes(), []byte("browser profile")) || resp.TLS.CurveID == 0 {
					t.Fatal("TLS response or negotiated curve was lost")
				}
				if profile.name == "chrome" && attempt == 1 && !resp.TLS.DidResume {
					t.Fatal("Chrome did not resume its TLS session")
				}
			}
		})
	}
	state := tlsConnectionStateFromUTLS(utls.ConnectionState{CurveID: utls.X25519, HelloRetryRequest: true})
	if state.CurveID != tls.X25519 || !state.HelloRetryRequest {
		t.Fatal("TLS state conversion lost curve or HelloRetryRequest")
	}
}
