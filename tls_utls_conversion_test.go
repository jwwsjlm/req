package req

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"

	utls "github.com/refraction-networking/utls"
)

func TestTLSServerNameDerivation(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		endpoint   string
		want       string
	}{
		{name: "explicit override", configured: "sni.example", endpoint: "dial.example:443", want: "sni.example"},
		{name: "host", endpoint: "example.com", want: "example.com"},
		{name: "host port", endpoint: "example.com:443", want: "example.com"},
		{name: "bracketed ipv6", endpoint: "[2001:db8::1]", want: "2001:db8::1"},
		{name: "bracketed ipv6 port", endpoint: "[2001:db8::1]:443", want: "2001:db8::1"},
		{name: "unbracketed ipv6", endpoint: "2001:db8::1", want: "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tlsServerName(tt.configured, tt.endpoint); got != tt.want {
				t.Fatalf("tlsServerName(%q, %q) = %q, want %q", tt.configured, tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestTLSVerifyConnectionKeyingMaterialPanicBecomesError(t *testing.T) {
	converted := tlsConfigToUTLS(&tls.Config{
		VerifyConnection: func(state tls.ConnectionState) error {
			_, _ = state.ExportKeyingMaterial("test", nil, 32)
			return nil
		},
	}, "example.test")

	err := converted.VerifyConnection(utls.ConnectionState{})
	if err == nil || !strings.Contains(err.Error(), "VerifyConnection panicked") {
		t.Fatalf("ExportKeyingMaterial panic error = %v", err)
	}
}

func TestTransportTLSHandshakeOverrideClearsFingerprintCloneBinding(t *testing.T) {
	wantErr := errors.New("custom handshake")
	client := C().SetTLSFingerprint(utls.HelloChrome_133)
	client.Transport.SetTLSHandshake(func(context.Context, string, net.Conn) (net.Conn, *tls.ConnectionState, error) {
		return nil, nil, wantErr
	})

	clone := client.Transport.Clone()
	if clone.tlsFingerprint != nil {
		t.Fatal("Transport.Clone restored a fingerprint after a custom TLS handshake override")
	}
	_, _, err := clone.TLSHandshakeContext(context.Background(), "example.test", nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("cloned TLS handshake error = %v, want %v", err, wantErr)
	}
}

func TestTLSConfigToUTLSPreservesClientSemantics(t *testing.T) {
	wantErr := errors.New("verification stopped")
	cache := tls.NewLRUClientSessionCache(4)
	certificate := tls.Certificate{Certificate: [][]byte{{1, 2, 3}}}
	base := &tls.Config{
		Certificates:                   []tls.Certificate{certificate},
		RootCAs:                        x509.NewCertPool(),
		NextProtos:                     []string{"h2", "http/1.1"},
		ServerName:                     "configured.example",
		InsecureSkipVerify:             true,
		CipherSuites:                   []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		ClientSessionCache:             cache,
		MinVersion:                     tls.VersionTLS12,
		MaxVersion:                     tls.VersionTLS13,
		CurvePreferences:               []tls.CurveID{tls.X25519, tls.CurveP256},
		DynamicRecordSizingDisabled:    true,
		Renegotiation:                  tls.RenegotiateOnceAsClient,
		EncryptedClientHelloConfigList: []byte{0, 1, 2},
		VerifyConnection: func(state tls.ConnectionState) error {
			if !state.ECHAccepted || state.ServerName != "configured.example" {
				t.Fatalf("unexpected bridged ConnectionState: %+v", state)
			}
			return wantErr
		},
	}

	converted := tlsConfigToUTLS(base, "endpoint.example:443")
	if converted.ServerName != base.ServerName {
		t.Fatalf("ServerName = %q, want %q", converted.ServerName, base.ServerName)
	}
	if len(converted.Certificates) != 1 || len(converted.Certificates[0].Certificate) != 1 {
		t.Fatal("client certificates were not converted")
	}
	if converted.ClientSessionCache == nil || !converted.PreferSkipResumptionOnNilExtension {
		t.Fatal("client session cache was not bridged safely")
	}
	if len(converted.CurvePreferences) != 2 || converted.CurvePreferences[0] != utls.X25519 {
		t.Fatalf("CurvePreferences were not preserved: %v", converted.CurvePreferences)
	}
	if converted.Renegotiation != utls.RenegotiateOnceAsClient {
		t.Fatalf("Renegotiation = %v, want %v", converted.Renegotiation, utls.RenegotiateOnceAsClient)
	}
	if converted.VerifyConnection == nil {
		t.Fatal("VerifyConnection was not bridged")
	}
	if err := converted.VerifyConnection(utls.ConnectionState{ServerName: base.ServerName, ECHAccepted: true}); !errors.Is(err, wantErr) {
		t.Fatalf("VerifyConnection error = %v, want %v", err, wantErr)
	}
	if string(converted.EncryptedClientHelloConfigList) != string(base.EncryptedClientHelloConfigList) {
		t.Fatal("ECH config list was not preserved")
	}
}

func TestUTLSPresetOwnsClientHelloShape(t *testing.T) {
	base := &tls.Config{
		NextProtos:       []string{"req-test"},
		CipherSuites:     []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		MinVersion:       tls.VersionTLS12,
		MaxVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP256},
		Renegotiation:    tls.RenegotiateNever,
	}
	uconn := &uTLSConn{UConn: utls.UClient(nil, tlsConfigToUTLS(base, "example.test"), utls.HelloChrome_133)}
	if err := prepareUTLSRenegotiationPolicy(uconn, utls.RenegotiateNever); err != nil {
		t.Fatalf("prepareUTLSRenegotiationPolicy: %v", err)
	}

	hello := uconn.HandshakeState.Hello
	if hello == nil {
		t.Fatal("uTLS did not build a ClientHello")
	}
	if reflect.DeepEqual(hello.AlpnProtocols, base.NextProtos) {
		t.Fatalf("ALPN unexpectedly followed tls.Config instead of the preset: %v", hello.AlpnProtocols)
	}
	if reflect.DeepEqual(hello.CipherSuites, base.CipherSuites) {
		t.Fatalf("cipher suites unexpectedly followed tls.Config instead of the preset: %v", hello.CipherSuites)
	}
	if reflect.DeepEqual(hello.SupportedCurves, []utls.CurveID{utls.CurveP256}) {
		t.Fatalf("curves unexpectedly followed tls.Config instead of the preset: %v", hello.SupportedCurves)
	}
	if !containsTLSVersion(hello.SupportedVersions, utls.VersionTLS13) {
		t.Fatalf("preset-supported versions = %v, want TLS 1.3 despite tls.Config MaxVersion TLS 1.2", hello.SupportedVersions)
	}
	if !hasUTLSRenegotiationPolicy(uconn, utls.RenegotiateNever) {
		t.Fatal("Chrome preset overrode tls.Config.Renegotiation")
	}
}

func TestFixedPresetRenegotiationPolicyKeepsExtensionBytes(t *testing.T) {
	newConfig := func() *utls.Config {
		return tlsConfigToUTLS(&tls.Config{
			Rand:          zeroTLSReader{},
			Renegotiation: tls.RenegotiateNever,
		}, "example.test")
	}

	direct := utls.UClient(nil, newConfig(), utls.HelloChrome_133)
	if err := direct.BuildHandshakeStateWithoutSession(); err != nil {
		t.Fatalf("build direct preset: %v", err)
	}

	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_133)
	if err != nil {
		t.Fatalf("UTLSIdToSpec: %v", err)
	}
	setUTLSSpecRenegotiationPolicy(&spec, utls.RenegotiateNever)
	custom := utls.UClient(nil, newConfig(), utls.HelloCustom)
	if err := custom.ApplyPreset(&spec); err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}
	if err := custom.BuildHandshakeStateWithoutSession(); err != nil {
		t.Fatalf("build policy-adjusted preset: %v", err)
	}

	renegotiationInfo := []byte{0xff, 0x01, 0x00, 0x01, 0x00}
	if !bytes.Contains(direct.HandshakeState.Hello.Raw, renegotiationInfo) {
		t.Fatal("direct Chrome preset does not contain the expected renegotiation_info bytes")
	}
	if !bytes.Contains(custom.HandshakeState.Hello.Raw, renegotiationInfo) {
		t.Fatal("restoring the policy changed the encoded renegotiation_info extension")
	}
}

func containsTLSVersion(versions []uint16, want uint16) bool {
	for _, version := range versions {
		if version == want {
			return true
		}
	}
	return false
}

func hasUTLSRenegotiationPolicy(conn *uTLSConn, want utls.RenegotiationSupport) bool {
	for _, extension := range conn.Extensions {
		if renegotiation, ok := extension.(*utls.RenegotiationInfoExtension); ok {
			return renegotiation.Renegotiation == want
		}
	}
	return false
}

type zeroTLSReader struct{}

func (zeroTLSReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
