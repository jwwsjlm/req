package examples

import (
	"encoding/binary"
	"testing"

	"github.com/jwwsjlm/req/v3"
	utls "github.com/refraction-networking/utls"
)

func TestSeededRandomizedTLSFingerprintConfiguration(t *testing.T) {
	seed, err := utls.NewPRNGSeed()
	if err != nil {
		t.Fatal(err)
	}

	client := req.C().SetTLSFingerprintRandomizedALPNWithSeed(seed)
	if client.GetTransport().TLSHandshakeContext == nil {
		t.Fatal("uTLS handshake was not configured")
	}
}

func TestCapturedClientHelloConfiguration(t *testing.T) {
	factory, err := req.ParseTLSClientHello(minimalClientHelloRecord())
	if err != nil {
		t.Fatal(err)
	}

	client := req.C().SetTLSFingerprintSpecFactory(factory)
	if client.GetTransport().TLSHandshakeContext == nil {
		t.Fatal("captured ClientHello was not configured")
	}
}

// minimalClientHelloRecord returns a complete plaintext TLS record suitable
// for demonstrating strict parsing. Real applications normally load bytes
// captured in an authorized test environment.
// minimalClientHelloRecord 返回可用于严格解析演示的完整明文 TLS record；
// 实际应用通常读取授权测试环境中捕获的字节。
func minimalClientHelloRecord() []byte {
	body := make([]byte, 0, 48)
	body = append(body, 0x03, 0x03)
	body = append(body, make([]byte, 32)...)
	body = append(body, 0x00)
	body = append(body, 0x00, 0x02, 0x13, 0x01)
	body = append(body, 0x01, 0x00)

	handshake := []byte{0x01, byte(len(body) >> 16), byte(len(body) >> 8), byte(len(body))}
	handshake = append(handshake, body...)
	record := []byte{0x16, 0x03, 0x01, 0x00, 0x00}
	binary.BigEndian.PutUint16(record[3:5], uint16(len(handshake)))
	return append(record, handshake...)
}
