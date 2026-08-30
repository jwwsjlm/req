package req

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	utls "github.com/refraction-networking/utls"
)

// tlsCompatCA is a short-lived in-memory CA used by the offline TLS tests.
// tlsCompatCA 是离线 TLS 测试使用的短期内存 CA，避免依赖固定证书或公网。
type tlsCompatCA struct {
	certificate *x509.Certificate
	key         *ecdsa.PrivateKey
	der         []byte
	pool        *x509.CertPool
}

type tlsCompatServer struct {
	server   *http.Server
	listener net.Listener
	url      string
}

// tlsCompatSessionCache exposes deterministic ticket/cache signals while the
// standard library LRU cache keeps the actual session state.
//
// tlsCompatSessionCache 在标准库 LRU cache 保存真实会话状态的同时，提供可
// 确定等待的 ticket/cache 信号，避免会话恢复测试依赖固定 sleep。
type tlsCompatSessionCache struct {
	cache     tls.ClientSessionCache
	putSignal chan struct{}
	gets      atomic.Int32
	hits      atomic.Int32
	puts      atomic.Int32
}

func newTLSCompatSessionCache() *tlsCompatSessionCache {
	return &tlsCompatSessionCache{
		cache:     tls.NewLRUClientSessionCache(4),
		putSignal: make(chan struct{}, 4),
	}
}

func (c *tlsCompatSessionCache) Get(sessionKey string) (*tls.ClientSessionState, bool) {
	c.gets.Add(1)
	state, ok := c.cache.Get(sessionKey)
	if ok && state != nil {
		c.hits.Add(1)
	}
	return state, ok
}

func (c *tlsCompatSessionCache) Put(sessionKey string, state *tls.ClientSessionState) {
	c.cache.Put(sessionKey, state)
	if state == nil {
		return
	}
	c.puts.Add(1)
	select {
	case c.putSignal <- struct{}{}:
	default:
	}
}

func newTLSCompatCA(t *testing.T, commonName string) *tlsCompatCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(certificate)
	return &tlsCompatCA{certificate: certificate, key: key, der: der, pool: pool}
}

func newTLSCompatLeaf(t *testing.T, ca *tlsCompatCA, commonName string, dnsNames []string, ipAddresses []net.IP, client bool) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	if client {
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.certificate, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("create leaf certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse leaf certificate: %v", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{der, ca.der},
		PrivateKey:  key,
		Leaf:        certificate,
	}
}

func startTLSCompatServer(network, host string, certificate tls.Certificate, clientAuth tls.ClientAuthType, clientCAs *x509.CertPool, handler http.Handler, getConfigForClient func(*tls.ClientHelloInfo) (*tls.Config, error)) (*tlsCompatServer, error) {
	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   clientAuth,
		ClientCAs:    clientCAs,
		NextProtos:   []string{"http/1.1"},
	}
	if getConfigForClient != nil {
		config.GetConfigForClient = getConfigForClient
	}
	return startTLSCompatServerWithConfig(network, host, config, handler)
}

func startTLSCompatServerWithConfig(network, host string, config *tls.Config, handler http.Handler) (*tlsCompatServer, error) {
	listener, err := net.Listen(network, net.JoinHostPort(host, "0"))
	if err != nil {
		return nil, err
	}
	tlsListener := tls.NewListener(listener, config)
	server := &http.Server{Handler: handler}
	go func() {
		_ = server.Serve(tlsListener)
	}()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		_ = server.Close()
		_ = listener.Close()
		return nil, err
	}
	return &tlsCompatServer{
		server:   server,
		listener: listener,
		url:      "https://" + net.JoinHostPort(host, port),
	}, nil
}

func (s *tlsCompatServer) Close() {
	_ = s.server.Close()
	_ = s.listener.Close()
}

func tlsCompatOKHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

func configuredTLSClient(config *tls.Config) *Client {
	client := C()
	client.Transport.SetProxy(nil)
	client.Transport.SetTLSClientConfig(config)
	return client
}

func tlsCompatClient(rootCAs *x509.CertPool) *Client {
	return configuredTLSClient(&tls.Config{
		RootCAs:    rootCAs,
		NextProtos: []string{"http/1.1"},
	}).
		SetTLSFingerprint(utls.HelloChrome_133)
}

func doTLSCompatGet(t *testing.T, client *Client, endpoint string) {
	t.Helper()
	response, err := client.R().Get(endpoint)
	if err != nil {
		t.Fatalf("GET %s: %v", endpoint, err)
	}
	if response == nil || response.Response == nil {
		t.Fatalf("GET %s returned a nil response", endpoint)
	}
	if response.Response.Body != nil {
		defer response.Response.Body.Close()
	}
	if got := response.String(); got != "ok" {
		t.Fatalf("GET %s body = %q, want %q", endpoint, got, "ok")
	}
}

func TestUTLSVerifyConnectionAndExplicitServerName(t *testing.T) {
	ca := newTLSCompatCA(t, "verify-connection-ca")
	certificate := newTLSCompatLeaf(t, ca, "verify.test", []string{"verify.test"}, nil, false)
	var serverName atomic.Value
	server, err := startTLSCompatServer("tcp4", "127.0.0.1", certificate, tls.NoClientCert, nil, http.HandlerFunc(tlsCompatOKHandler), func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
		serverName.Store(hello.ServerName)
		return nil, nil
	})
	if err != nil {
		t.Fatalf("start TLS server: %v", err)
	}
	defer server.Close()

	var callbackCount atomic.Int32
	client := configuredTLSClient(&tls.Config{
		RootCAs:    ca.pool,
		ServerName: "verify.test",
		NextProtos: []string{"http/1.1"},
		VerifyConnection: func(state tls.ConnectionState) error {
			callbackCount.Add(1)
			if state.ServerName != "verify.test" {
				return fmt.Errorf("unexpected connection ServerName %q", state.ServerName)
			}
			if len(state.PeerCertificates) == 0 {
				return errors.New("peer certificate missing")
			}
			return nil
		},
	}).SetTLSFingerprint(utls.HelloChrome_133)
	doTLSCompatGet(t, client, server.url)

	if callbackCount.Load() == 0 {
		t.Fatal("VerifyConnection was not called for a uTLS handshake")
	}
	if got, _ := serverName.Load().(string); got != "verify.test" {
		t.Fatalf("server observed SNI %q, want %q", got, "verify.test")
	}
}

func TestUTLSVerifyPeerCertificate(t *testing.T) {
	ca := newTLSCompatCA(t, "verify-peer-ca")
	certificate := newTLSCompatLeaf(t, ca, "peer-check.test", []string{"peer-check.test"}, nil, false)
	server, err := startTLSCompatServer("tcp4", "127.0.0.1", certificate, tls.NoClientCert, nil, http.HandlerFunc(tlsCompatOKHandler), nil)
	if err != nil {
		t.Fatalf("start TLS server: %v", err)
	}
	defer server.Close()

	var callbackCount atomic.Int32
	client := configuredTLSClient(&tls.Config{
		RootCAs:    ca.pool,
		ServerName: "peer-check.test",
		NextProtos: []string{"http/1.1"},
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			callbackCount.Add(1)
			if len(rawCerts) == 0 {
				return errors.New("raw peer certificate missing")
			}
			leaf, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("parse peer certificate: %w", err)
			}
			if leaf.Subject.CommonName != "peer-check.test" {
				return fmt.Errorf("unexpected peer CommonName %q", leaf.Subject.CommonName)
			}
			if len(verifiedChains) == 0 {
				return errors.New("verified chain missing")
			}
			return nil
		},
	}).SetTLSFingerprint(utls.HelloChrome_133)
	doTLSCompatGet(t, client, server.url)

	if callbackCount.Load() == 0 {
		t.Fatal("VerifyPeerCertificate was not called for a uTLS handshake")
	}
}

func TestUTLSMutualTLSClientCertificate(t *testing.T) {
	serverCA := newTLSCompatCA(t, "mTLS-server-ca")
	clientCA := newTLSCompatCA(t, "mTLS-client-ca")
	serverCertificate := newTLSCompatLeaf(t, serverCA, "mtls.test", nil, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	clientCertificate := newTLSCompatLeaf(t, clientCA, "req-client", nil, nil, true)
	var clientCertificateSeen atomic.Int32
	server, err := startTLSCompatServer("tcp4", "127.0.0.1", serverCertificate, tls.RequireAndVerifyClientCert, clientCA.pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client certificate missing", http.StatusUnauthorized)
			return
		}
		if r.TLS.PeerCertificates[0].Subject.CommonName != "req-client" {
			http.Error(w, "unexpected client certificate", http.StatusUnauthorized)
			return
		}
		clientCertificateSeen.Add(1)
		tlsCompatOKHandler(w, r)
	}), nil)
	if err != nil {
		t.Fatalf("start mTLS server: %v", err)
	}
	defer server.Close()

	client := tlsCompatClient(serverCA.pool).SetCerts(clientCertificate)
	doTLSCompatGet(t, client, server.url)
	if clientCertificateSeen.Load() == 0 {
		t.Fatal("server did not observe the client certificate")
	}
}

func TestUTLSDynamicGetClientCertificate(t *testing.T) {
	serverCA := newTLSCompatCA(t, "dynamic-mTLS-server-ca")
	clientCA := newTLSCompatCA(t, "dynamic-mTLS-client-ca")
	serverCertificate := newTLSCompatLeaf(t, serverCA, "dynamic-mtls.test", nil, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	clientCertificate := newTLSCompatLeaf(t, clientCA, "dynamic-req-client", nil, nil, true)
	var clientCertificateSeen atomic.Int32
	server, err := startTLSCompatServer("tcp4", "127.0.0.1", serverCertificate, tls.RequireAndVerifyClientCert, clientCA.pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "dynamic client certificate missing", http.StatusUnauthorized)
			return
		}
		if r.TLS.PeerCertificates[0].Subject.CommonName != "dynamic-req-client" {
			http.Error(w, "unexpected dynamic client certificate", http.StatusUnauthorized)
			return
		}
		clientCertificateSeen.Add(1)
		tlsCompatOKHandler(w, r)
	}), nil)
	if err != nil {
		t.Fatalf("start dynamic mTLS server: %v", err)
	}
	defer server.Close()

	var callbackCount atomic.Int32
	client := configuredTLSClient(&tls.Config{
		RootCAs:    serverCA.pool,
		NextProtos: []string{"http/1.1"},
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			callbackCount.Add(1)
			if info == nil {
				return nil, errors.New("nil CertificateRequestInfo")
			}
			return &clientCertificate, nil
		},
	}).SetTLSFingerprint(utls.HelloChrome_133)
	doTLSCompatGet(t, client, server.url)
	if callbackCount.Load() == 0 {
		t.Fatal("GetClientCertificate was not called for a uTLS mTLS handshake")
	}
	if clientCertificateSeen.Load() == 0 {
		t.Fatal("server did not observe the dynamically selected client certificate")
	}
}

func TestUTLSTLS13ClientSessionCacheResumption(t *testing.T) {
	ca := newTLSCompatCA(t, "session-resumption-ca")
	certificate := newTLSCompatLeaf(t, ca, "session-resumption.test", nil, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	serverConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		NextProtos:   []string{"http/1.1"},
	}
	var ticketKey [32]byte
	copy(ticketKey[:], "req-tls-utls-session-ticket-key")
	serverConfig.SetSessionTicketKeys([][32]byte{ticketKey})

	type observation struct {
		didResume bool
		remote    string
		version   uint16
	}
	observations := make(chan observation, 2)
	server, err := startTLSCompatServerWithConfig("tcp4", "127.0.0.1", serverConfig, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			http.Error(w, "TLS state missing", http.StatusInternalServerError)
			return
		}
		observations <- observation{
			didResume: r.TLS.DidResume,
			remote:    r.RemoteAddr,
			version:   r.TLS.Version,
		}
		w.Header().Set("Connection", "close")
		tlsCompatOKHandler(w, r)
	}))
	if err != nil {
		t.Fatalf("start TLS 1.3 session server: %v", err)
	}
	defer server.Close()

	cache := newTLSCompatSessionCache()
	client := configuredTLSClient(&tls.Config{
		RootCAs:            ca.pool,
		NextProtos:         []string{"http/1.1"},
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		ClientSessionCache: cache,
	}).
		// Browser parrot presets without a real PSK extension intentionally skip
		// resumption. HelloGolang keeps the request on req's uTLS handshake path
		// while exercising uTLS's fully supported TLS 1.3 PSK implementation.
		//
		// 不带真实 PSK 扩展的浏览器 preset 会有意跳过恢复。HelloGolang 仍
		// 经过 req 的 uTLS 握手路径，同时使用 uTLS 完整支持的 TLS 1.3 PSK。
		SetTLSFingerprint(utls.HelloGolang)
	client.Transport.DisableKeepAlives = true

	doTLSCompatGet(t, client, server.url)
	first := <-observations
	if first.version != tls.VersionTLS13 {
		t.Fatalf("first connection negotiated TLS version %#x, want TLS 1.3", first.version)
	}
	if first.didResume {
		t.Fatal("first TLS 1.3 connection unexpectedly resumed a session")
	}
	select {
	case <-cache.putSignal:
	case <-time.After(3 * time.Second):
		t.Fatalf("TLS 1.3 session ticket was not cached: gets=%d hits=%d puts=%d", cache.gets.Load(), cache.hits.Load(), cache.puts.Load())
	}

	client.CloseIdleConnections()
	doTLSCompatGet(t, client, server.url)
	second := <-observations
	if second.version != tls.VersionTLS13 {
		t.Fatalf("second connection negotiated TLS version %#x, want TLS 1.3", second.version)
	}
	if first.remote == second.remote {
		t.Fatalf("requests reused one connection %q; session resumption requires a new connection", first.remote)
	}
	if !second.didResume {
		t.Fatalf("second TLS 1.3 connection did not resume: gets=%d hits=%d puts=%d", cache.gets.Load(), cache.hits.Load(), cache.puts.Load())
	}
	if cache.hits.Load() == 0 {
		t.Fatalf("second TLS 1.3 connection did not read cached state: gets=%d puts=%d", cache.gets.Load(), cache.puts.Load())
	}
}

func TestUTLSIPv6HostNameParsing(t *testing.T) {
	ca := newTLSCompatCA(t, "ipv6-ca")
	certificate := newTLSCompatLeaf(t, ca, "ipv6-loopback", nil, []net.IP{net.ParseIP("::1")}, false)
	server, err := startTLSCompatServer("tcp6", "::1", certificate, tls.NoClientCert, nil, http.HandlerFunc(tlsCompatOKHandler), nil)
	if err != nil {
		t.Skipf("IPv6 loopback is unavailable: %v", err)
	}
	defer server.Close()

	client := tlsCompatClient(ca.pool)
	doTLSCompatGet(t, client, server.url)
}

func TestUTLSCloneUsesIndependentTLSConfig(t *testing.T) {
	caA := newTLSCompatCA(t, "clone-a-ca")
	caB := newTLSCompatCA(t, "clone-b-ca")
	certificateA := newTLSCompatLeaf(t, caA, "clone-a", nil, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	certificateB := newTLSCompatLeaf(t, caB, "clone-b", nil, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	serverA, err := startTLSCompatServer("tcp4", "127.0.0.1", certificateA, tls.NoClientCert, nil, http.HandlerFunc(tlsCompatOKHandler), nil)
	if err != nil {
		t.Fatalf("start clone server A: %v", err)
	}
	defer serverA.Close()
	serverB, err := startTLSCompatServer("tcp4", "127.0.0.1", certificateB, tls.NoClientCert, nil, http.HandlerFunc(tlsCompatOKHandler), nil)
	if err != nil {
		t.Fatalf("start clone server B: %v", err)
	}
	defer serverB.Close()

	base := tlsCompatClient(caA.pool)
	clone := base.Clone()
	clone.Transport.SetTLSClientConfig(&tls.Config{RootCAs: caB.pool, NextProtos: []string{"http/1.1"}})
	doTLSCompatGet(t, base, serverA.url)
	doTLSCompatGet(t, clone, serverB.url)
}

func TestUTLSNilClientHelloSpecFactoryReturnsError(t *testing.T) {
	const childProcessEnv = "REQ_TEST_UTLS_NIL_SPEC_CHILD"
	if os.Getenv(childProcessEnv) != "1" {
		command := exec.Command(os.Args[0], "-test.run=^TestUTLSNilClientHelloSpecFactoryReturnsError$", "-test.v")
		command.Env = append(os.Environ(), childProcessEnv+"=1")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("nil ClientHelloSpec factory subprocess failed (possible panic): %v\n%s", err, output)
		}
		return
	}

	ca := newTLSCompatCA(t, "nil-spec-ca")
	certificate := newTLSCompatLeaf(t, ca, "nil-spec.test", nil, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	server, err := startTLSCompatServer("tcp4", "127.0.0.1", certificate, tls.NoClientCert, nil, http.HandlerFunc(tlsCompatOKHandler), nil)
	if err != nil {
		t.Fatalf("start nil-spec server: %v", err)
	}
	defer server.Close()

	client := configuredTLSClient(&tls.Config{
		InsecureSkipVerify: true, // test server uses a deliberately untrusted in-memory CA.
		NextProtos:         []string{"http/1.1"},
	}).SetTLSFingerprintSpecFactory(func() *utls.ClientHelloSpec {
		return nil
	})

	response, requestErr := client.R().Get(server.url)
	if response != nil && response.Response != nil && response.Response.Body != nil {
		_ = response.Response.Body.Close()
	}
	if requestErr == nil {
		t.Fatal("nil ClientHelloSpec factory returned nil error")
	}
	message := strings.ToLower(requestErr.Error())
	if !strings.Contains(message, "spec") && !strings.Contains(message, "nil") {
		t.Fatalf("nil ClientHelloSpec factory returned an unclear error: %v", requestErr)
	}
}
