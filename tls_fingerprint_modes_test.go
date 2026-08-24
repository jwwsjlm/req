package req

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	utls "github.com/refraction-networking/utls"
)

func TestTLSFingerprintRandomizedModes(t *testing.T) {
	tests := []struct {
		name     string
		base     utls.ClientHelloID
		wantALPN bool
		apply    func(*Client) *Client
	}{
		{
			name:     "alpn",
			base:     utls.HelloRandomizedALPN,
			wantALPN: true,
			apply:    (*Client).SetTLSFingerprintRandomizedALPN,
		},
		{
			name:     "no_alpn",
			base:     utls.HelloRandomizedNoALPN,
			wantALPN: false,
			apply:    (*Client).SetTLSFingerprintRandomizedNoALPN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := C()
			if got := tt.apply(client); got != client {
				t.Fatal("fingerprint setter did not preserve the client chain")
			}
			if client.Transport.TLSHandshakeContext == nil {
				t.Fatal("fingerprint setter did not install a TLS handshake")
			}

			id := randomizedTLSFingerprintID(tt.base, nil)
			spec, err := utls.UTLSIdToSpec(id)
			if err != nil {
				t.Fatalf("UTLSIdToSpec returned an error: %v", err)
			}
			if got := clientHelloSpecHasALPN(&spec); got != tt.wantALPN {
				t.Fatalf("ALPN presence = %v, want %v", got, tt.wantALPN)
			}
		})
	}
}

func TestTLSFingerprintRandomizedSeedIsCopied(t *testing.T) {
	var seed utls.PRNGSeed
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	want := seed

	id := randomizedTLSFingerprintID(utls.HelloRandomizedALPN, &seed)
	if id.Seed == nil {
		t.Fatal("copied seed is nil")
	}
	if id.Seed == &seed {
		t.Fatal("fingerprint retained the caller's seed pointer")
	}

	seed[0] ^= 0xff
	if *id.Seed != want {
		t.Fatal("modifying the caller's seed changed the configured fingerprint")
	}

	client := C()
	if got := client.SetTLSFingerprintRandomizedALPNWithSeed(&want); got != client {
		t.Fatal("ALPN seeded setter did not preserve the client chain")
	}
	if got := client.SetTLSFingerprintRandomizedNoALPNWithSeed(&want); got != client {
		t.Fatal("NoALPN seeded setter did not preserve the client chain")
	}
}

func TestSetTLSFingerprintCopiesRandomizationState(t *testing.T) {
	var seed utls.PRNGSeed
	seed[0] = 7
	weights := utls.DefaultWeights
	id := utls.HelloRandomizedALPN
	id.Seed = &seed
	id.Weights = &weights

	client := C().SetTLSFingerprint(id)
	seed[0] = 99
	weights.Extensions_Append_ALPN = 0

	configured := client.Transport.tlsFingerprint.clientHelloID
	if configured.Seed == id.Seed || configured.Weights == id.Weights {
		t.Fatal("SetTLSFingerprint retained caller-owned randomization pointers")
	}
	if configured.Seed[0] != 7 {
		t.Fatal("modifying the caller seed changed the configured fingerprint")
	}
	if configured.Weights.Extensions_Append_ALPN == 0 {
		t.Fatal("modifying the caller weights changed the configured fingerprint")
	}
}

func TestParseTLSClientHelloReturnsFreshSpecs(t *testing.T) {
	raw := testRawClientHello(false)
	factory, err := ParseTLSClientHello(raw)
	if err != nil {
		t.Fatalf("ParseTLSClientHello returned an error: %v", err)
	}

	first := factory()
	second := factory()
	if first == second {
		t.Fatal("factory reused a ClientHelloSpec pointer")
	}
	if len(first.CipherSuites) == 0 || len(second.CipherSuites) == 0 {
		t.Fatal("parsed ClientHello has no cipher suites")
	}

	wantCipherSuite := second.CipherSuites[0]
	first.CipherSuites[0] ^= 0xffff
	if second.CipherSuites[0] != wantCipherSuite {
		t.Fatal("factory reused mutable cipher suite storage")
	}
	wantCompressionMethod := second.CompressionMethods[0]
	first.CompressionMethods[0] ^= 0xff
	third := factory()
	if len(third.CompressionMethods) == 0 || third.CompressionMethods[0] != wantCompressionMethod {
		t.Fatal("factory reused parsed ClientHello byte storage")
	}

	for i := range raw {
		raw[i] = 0
	}
	fourth := factory()
	if len(fourth.CipherSuites) == 0 || fourth.CipherSuites[0] != wantCipherSuite {
		t.Fatal("modifying the caller's raw bytes changed the parsed fingerprint")
	}
}

func TestParseTLSClientHelloFactoryIsSafeForConcurrentSpecs(t *testing.T) {
	factory, err := ParseTLSClientHello(testRawClientHello(false))
	if err != nil {
		t.Fatalf("ParseTLSClientHello returned an error: %v", err)
	}

	const workers = 32
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			spec := factory()
			if len(spec.CompressionMethods) == 0 {
				errs <- fmt.Errorf("spec %d has no compression methods", i)
				return
			}
			<-start
			spec.CompressionMethods[0] ^= byte(i + 1)
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestParseTLSClientHelloIsStrict(t *testing.T) {
	_, err := ParseTLSClientHello(testRawClientHello(true))
	if err == nil {
		t.Fatal("unknown extension was accepted")
	}
	if !strings.Contains(err.Error(), "unsupported extension") {
		t.Fatalf("unexpected strict parsing error: %v", err)
	}
}

func TestParseTLSClientHelloRejectsUnsafeSessionExtensions(t *testing.T) {
	psk := tlsClientHelloExtension{id: 41, data: []byte{0, 0, 0, 0}}
	sessionTicket := tlsClientHelloExtension{id: 35}

	tests := []struct {
		name       string
		extensions []tlsClientHelloExtension
		want       string
	}{
		{
			name:       "PSK is last but unsupported",
			extensions: []tlsClientHelloExtension{psk},
			want:       "pre-shared key extension is not supported",
		},
		{
			name:       "PSK is not last",
			extensions: []tlsClientHelloExtension{psk, sessionTicket},
			want:       "pre-shared key extension must be the last extension",
		},
		{
			name:       "duplicate PSK",
			extensions: []tlsClientHelloExtension{psk, psk},
			want:       "2 pre-shared key extensions",
		},
		{
			name:       "duplicate session ticket",
			extensions: []tlsClientHelloExtension{sessionTicket, sessionTicket},
			want:       "2 session ticket extensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTLSClientHello(testRawClientHelloWithExtensions(tt.extensions))
			if err == nil {
				t.Fatal("unsafe session extension layout was accepted")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ParseTLSClientHello error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestParseTLSClientHelloAcceptedSessionLayoutAppliesWithoutPanic(t *testing.T) {
	factory, err := ParseTLSClientHello(testRawClientHelloWithExtensions([]tlsClientHelloExtension{{id: 35}}))
	if err != nil {
		t.Fatalf("ParseTLSClientHello returned an error: %v", err)
	}

	client, peer := net.Pipe()
	defer client.Close()
	defer peer.Close()

	conn := utls.UClient(client, &utls.Config{
		ServerName:             "example.com",
		SessionTicketsDisabled: true,
	}, utls.HelloCustom)
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("ApplyPreset panicked for an accepted ClientHello: %v", recovered)
		}
	}()
	if err := conn.ApplyPreset(factory()); err != nil {
		t.Fatalf("ApplyPreset returned an error for an accepted ClientHello: %v", err)
	}
}

func TestParseTLSClientHelloValidatesRecordFraming(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "empty"},
		{name: "wrong record type", raw: func() []byte {
			raw := testRawClientHello(false)
			raw[0] = 23
			return raw
		}()},
		{name: "wrong record length", raw: func() []byte {
			raw := testRawClientHello(false)
			raw[4]++
			return raw
		}()},
		{name: "wrong handshake type", raw: func() []byte {
			raw := testRawClientHello(false)
			raw[5] = 2
			return raw
		}()},
		{name: "wrong handshake length", raw: func() []byte {
			raw := testRawClientHello(false)
			raw[8]++
			return raw
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseTLSClientHello(tt.raw); err == nil {
				t.Fatal("invalid TLS record was accepted")
			}
		})
	}
}

func clientHelloSpecHasALPN(spec *utls.ClientHelloSpec) bool {
	for _, extension := range spec.Extensions {
		if _, ok := extension.(*utls.ALPNExtension); ok {
			return true
		}
	}
	return false
}

type tlsClientHelloExtension struct {
	id   uint16
	data []byte
}

func testRawClientHello(withUnknownExtension bool) []byte {
	var extensions []tlsClientHelloExtension
	if withUnknownExtension {
		extensions = append(extensions, tlsClientHelloExtension{id: 0x1234})
	}
	return testRawClientHelloWithExtensions(extensions)
}

func testRawClientHelloWithExtensions(extensions []tlsClientHelloExtension) []byte {
	body := make([]byte, 0, 64)
	body = append(body, 0x03, 0x03)             // legacy_version
	body = append(body, make([]byte, 32)...)    // random
	body = append(body, 0x00)                   // session ID length
	body = append(body, 0x00, 0x02, 0x13, 0x01) // cipher suites
	body = append(body, 0x01, 0x00)             // compression methods
	if len(extensions) > 0 {
		extensionBytes := make([]byte, 0)
		for _, extension := range extensions {
			var header [4]byte
			binary.BigEndian.PutUint16(header[:2], extension.id)
			binary.BigEndian.PutUint16(header[2:], uint16(len(extension.data)))
			extensionBytes = append(extensionBytes, header[:]...)
			extensionBytes = append(extensionBytes, extension.data...)
		}
		var extensionLength [2]byte
		binary.BigEndian.PutUint16(extensionLength[:], uint16(len(extensionBytes)))
		body = append(body, extensionLength[:]...)
		body = append(body, extensionBytes...)
	}

	handshake := make([]byte, 4, 4+len(body))
	handshake[0] = 1 // ClientHello
	handshake[1] = byte(len(body) >> 16)
	handshake[2] = byte(len(body) >> 8)
	handshake[3] = byte(len(body))
	handshake = append(handshake, body...)

	record := make([]byte, 5, 5+len(handshake))
	record[0] = 22 // handshake
	record[1] = 0x03
	record[2] = 0x01
	binary.BigEndian.PutUint16(record[3:5], uint16(len(handshake)))
	return append(record, handshake...)
}
