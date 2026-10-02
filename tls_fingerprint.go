package req

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	utls "github.com/refraction-networking/utls"
)

const maxTLSPlaintextLength = 1 << 14

// chrome152Spec adapts surf v1.0.206's Chrome 152 profile on top of uTLS's
// common Chrome fields. Trust-anchor bytes are from enetx/surf (MIT); see
// THIRD_PARTY_NOTICES.md. Each handshake owns its spec and extension data.
// chrome152Spec 复用 uTLS 的公共 Chrome 字段，补上 surf 的 Chrome 152 扩展；
// 每次握手独占数据，避免随机排序和 GREASE 写入污染下一条连接。
func chrome152Spec() *utls.ClientHelloSpec {
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_133)
	if err != nil {
		return nil
	}
	var random [1]byte
	if _, err := rand.Read(random[:]); err != nil {
		return nil
	}
	grease := utls.SignatureScheme(uint16(random[0]&15)*0x1010 + 0x0a0a)
	for _, ext := range spec.Extensions {
		if sig, ok := ext.(*utls.SignatureAlgorithmsExtension); ok {
			sig.SupportedSignatureAlgorithms = append([]utls.SignatureScheme{
				grease, 0x0904, 0x0905, 0x0906, // GREASE and ML-DSA 44/65/87
			}, sig.SupportedSignatureAlgorithms...)
		}
	}
	// Google Chrome 152.0.7977 desktop trust-anchor list. Order is fixed by
	// the browser build; only the surrounding extension order is shuffled.
	anchors := &utls.GenericExtension{Id: 0xca34, Data: []byte{
		0x00, 0xcc,
		0x04, 0xd6, 0x79, 0x09, 0x06,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x07,
		0x04, 0xd6, 0x79, 0x09, 0x0c,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x06,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x13,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x0d,
		0x04, 0xd6, 0x79, 0x09, 0x01,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x0d,
		0x04, 0xd6, 0x79, 0x09, 0x0d,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x0f,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x08,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x12,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x09,
		0x04, 0xd6, 0x79, 0x09, 0x02,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x01,
		0x04, 0xd6, 0x79, 0x09, 0x0e,
		0x04, 0xd6, 0x79, 0x09, 0x09,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x0a,
		0x04, 0xd6, 0x79, 0x09, 0x03,
		0x04, 0xd6, 0x79, 0x09, 0x0f,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x0b,
		0x04, 0xd6, 0x79, 0x09, 0x04,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x14,
		0x04, 0xd6, 0x79, 0x09, 0x0a,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x13,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x12,
		0x04, 0xd6, 0x79, 0x09, 0x07,
		0x04, 0xd6, 0x79, 0x09, 0x08,
		0x08, 0x83, 0x9a, 0x64, 0x8c, 0x9b, 0x2d, 0x01, 0x0c,
		0x04, 0xd6, 0x79, 0x09, 0x05,
		0x05, 0x82, 0xdf, 0x13, 0x02, 0x0e,
		0x04, 0xd6, 0x79, 0x09, 0x0b,
	}}
	last := len(spec.Extensions) - 1
	trailingGREASE := spec.Extensions[last]
	spec.Extensions[last] = anchors
	spec.Extensions = append(spec.Extensions, trailingGREASE, &utls.UtlsPreSharedKeyExtension{})
	spec.Extensions = utls.ShuffleChromeTLSExtensions(spec.Extensions)
	return &spec
}

// SetTLSFingerprintRandomizedALPN uses a randomized uTLS fingerprint that
// always advertises ALPN. It is suitable when HTTP/2 negotiation is allowed.
//
// SetTLSFingerprintRandomizedALPN 使用始终携带 ALPN 的随机 uTLS 指纹，
// 适用于允许协商 HTTP/2 的客户端。
func (c *Client) SetTLSFingerprintRandomizedALPN() *Client {
	return c.SetTLSFingerprint(randomizedTLSFingerprintID(utls.HelloRandomizedALPN, nil))
}

// SetTLSFingerprintRandomizedNoALPN uses a randomized uTLS fingerprint that
// does not advertise ALPN. It is suitable for HTTP/1-only peers.
//
// SetTLSFingerprintRandomizedNoALPN 使用不携带 ALPN 的随机 uTLS 指纹，
// 适用于仅支持 HTTP/1 的对端。
func (c *Client) SetTLSFingerprintRandomizedNoALPN() *Client {
	return c.SetTLSFingerprint(randomizedTLSFingerprintID(utls.HelloRandomizedNoALPN, nil))
}

// SetTLSFingerprintRandomizedALPNWithSeed uses a reproducible randomized
// fingerprint that always advertises ALPN. The seed is copied before use, so
// modifying it after this call cannot change future handshakes. A nil seed
// requests a new uTLS-generated seed for each connection.
//
// SetTLSFingerprintRandomizedALPNWithSeed 使用可复现且始终携带 ALPN 的
// 随机指纹。方法会先复制 seed，调用后继续修改原值不会影响后续握手；
// 传入 nil 时，每条连接仍由 uTLS 生成新的 seed。
func (c *Client) SetTLSFingerprintRandomizedALPNWithSeed(seed *utls.PRNGSeed) *Client {
	return c.SetTLSFingerprint(randomizedTLSFingerprintID(utls.HelloRandomizedALPN, seed))
}

// SetTLSFingerprintRandomizedNoALPNWithSeed uses a reproducible randomized
// fingerprint that never advertises ALPN. The seed is copied before use, so
// modifying it after this call cannot change future handshakes. A nil seed
// requests a new uTLS-generated seed for each connection.
//
// SetTLSFingerprintRandomizedNoALPNWithSeed 使用可复现且不携带 ALPN 的
// 随机指纹。方法会先复制 seed，调用后继续修改原值不会影响后续握手；
// 传入 nil 时，每条连接仍由 uTLS 生成新的 seed。
func (c *Client) SetTLSFingerprintRandomizedNoALPNWithSeed(seed *utls.PRNGSeed) *Client {
	return c.SetTLSFingerprint(randomizedTLSFingerprintID(utls.HelloRandomizedNoALPN, seed))
}

// SetTLSFingerprintRandomizedALPN delegates to the default client's method of
// the same name.
//
// SetTLSFingerprintRandomizedALPN 将配置应用到包级默认 Client。
func SetTLSFingerprintRandomizedALPN() *Client {
	return defaultClient.SetTLSFingerprintRandomizedALPN()
}

// SetTLSFingerprintRandomizedNoALPN delegates to the default client's method
// of the same name.
//
// SetTLSFingerprintRandomizedNoALPN 将配置应用到包级默认 Client。
func SetTLSFingerprintRandomizedNoALPN() *Client {
	return defaultClient.SetTLSFingerprintRandomizedNoALPN()
}

// SetTLSFingerprintRandomizedALPNWithSeed delegates to the default client's
// method of the same name.
//
// SetTLSFingerprintRandomizedALPNWithSeed 将带 seed 的配置应用到包级默认
// Client。
func SetTLSFingerprintRandomizedALPNWithSeed(seed *utls.PRNGSeed) *Client {
	return defaultClient.SetTLSFingerprintRandomizedALPNWithSeed(seed)
}

// SetTLSFingerprintRandomizedNoALPNWithSeed delegates to the default client's
// method of the same name.
//
// SetTLSFingerprintRandomizedNoALPNWithSeed 将带 seed 的配置应用到包级默认
// Client。
func SetTLSFingerprintRandomizedNoALPNWithSeed(seed *utls.PRNGSeed) *Client {
	return defaultClient.SetTLSFingerprintRandomizedNoALPNWithSeed(seed)
}

// ParseTLSClientHello validates and strictly parses one complete plaintext TLS
// ClientHello record. Unknown extensions and captured PSK extensions are
// rejected. This keeps the resulting spec inside uTLS sessionController's
// invariants before it reaches ApplyPreset. The returned factory reparses a
// private copy of raw on every call, so each handshake receives a fresh spec
// and later caller mutations cannot affect it.
//
// ParseTLSClientHello 校验并严格解析一条完整的明文 TLS ClientHello record。
// 未知扩展和捕获到的 PSK 扩展会被拒绝，确保生成的 spec 在传给 ApplyPreset
// 前满足 uTLS sessionController 的不变量。返回的 factory 每次都会从 raw 的
// 私有副本重新解析，确保每次握手获得全新的 spec，且调用方之后修改原始字节
// 不会产生影响。
func ParseTLSClientHello(raw []byte) (func() *utls.ClientHelloSpec, error) {
	rawCopy := append([]byte(nil), raw...)
	if err := validateTLSClientHelloRecord(rawCopy); err != nil {
		return nil, err
	}
	if _, err := parseTLSClientHelloStrict(rawCopy); err != nil {
		return nil, fmt.Errorf("req: parse TLS ClientHello: %w", err)
	}

	return func() *utls.ClientHelloSpec {
		// uTLS intentionally keeps slices for parts of a parsed ClientHello. Give
		// every parse its own backing bytes so a returned spec cannot mutate the
		// factory's template or race with another handshake.
		//
		// uTLS 会保留部分已解析 ClientHello 的切片。每次解析都使用独立的底层
		// 字节，避免返回的 spec 污染 factory 模板或与另一条握手发生数据竞争。
		rawForParse := append([]byte(nil), rawCopy...)
		spec, err := parseTLSClientHelloStrict(rawForParse)
		if err != nil {
			// rawCopy is private and was parsed successfully above. Reaching this
			// branch would indicate an internal invariant violation.
			// rawCopy 是已成功解析的私有副本；到达此分支说明内部不变量被破坏。
			panic("req: validated TLS ClientHello became invalid: " + err.Error())
		}
		return spec
	}, nil
}

func randomizedTLSFingerprintID(base utls.ClientHelloID, seed *utls.PRNGSeed) utls.ClientHelloID {
	id := base
	id.Seed = nil
	if seed != nil {
		seedCopy := *seed
		id.Seed = &seedCopy
	}
	return id
}

func parseTLSClientHelloStrict(raw []byte) (*utls.ClientHelloSpec, error) {
	fingerprinter := utls.Fingerprinter{
		AllowBluntMimicry: false,
		RealPSKResumption: false,
	}
	spec, err := fingerprinter.RawClientHello(raw)
	if err != nil {
		return nil, err
	}
	if err := validateTLSClientHelloSpec(spec); err != nil {
		return nil, err
	}
	return spec, nil
}

// validateTLSClientHelloSpec rejects session-related extension layouts that
// uTLS 1.8.2 treats as internal invariants and checks with uAssert during
// ApplyPreset. Captured PSK data is intentionally unsupported: even fake PSK
// parsing retains identities and binders that are unsafe to replay blindly.
//
// validateTLSClientHelloSpec 拒绝会让 uTLS 1.8.2 在 ApplyPreset 中以 uAssert
// 检查的会话扩展布局。捕获的 PSK 数据被有意禁用：即使解析为 fake PSK，
// 其中的 identity 和 binder 仍会被保留，不能被盲目重放。
func validateTLSClientHelloSpec(spec *utls.ClientHelloSpec) error {
	if spec == nil {
		return fmt.Errorf("req: parsed TLS ClientHello spec is nil")
	}

	var sessionTicketCount, pskCount int
	pskIndex := -1
	for i, extension := range spec.Extensions {
		switch extension.(type) {
		case utls.ISessionTicketExtension:
			sessionTicketCount++
		case utls.PreSharedKeyExtension:
			pskCount++
			pskIndex = i
		}
	}

	if sessionTicketCount > 1 {
		return fmt.Errorf("req: TLS ClientHello contains %d session ticket extensions; only one is supported", sessionTicketCount)
	}
	if pskCount > 1 {
		return fmt.Errorf("req: TLS ClientHello contains %d pre-shared key extensions; only one is supported", pskCount)
	}
	if pskIndex >= 0 && pskIndex != len(spec.Extensions)-1 {
		return fmt.Errorf("req: TLS ClientHello pre-shared key extension must be the last extension")
	}
	if pskCount != 0 {
		return fmt.Errorf("req: TLS ClientHello pre-shared key extension is not supported")
	}
	return nil
}

func validateTLSClientHelloRecord(raw []byte) error {
	const (
		tlsRecordHeaderLength    = 5
		tlsHandshakeHeaderLength = 4
		tlsHandshakeRecordType   = 22
		tlsClientHelloType       = 1
	)

	if len(raw) < tlsRecordHeaderLength+tlsHandshakeHeaderLength {
		return fmt.Errorf("req: TLS ClientHello record is too short: %d bytes", len(raw))
	}
	if raw[0] != tlsHandshakeRecordType {
		return fmt.Errorf("req: TLS record type %d is not a handshake", raw[0])
	}

	recordLength := int(binary.BigEndian.Uint16(raw[3:5]))
	if recordLength > maxTLSPlaintextLength {
		return fmt.Errorf("req: TLS ClientHello record length %d exceeds %d bytes", recordLength, maxTLSPlaintextLength)
	}
	if recordLength != len(raw)-tlsRecordHeaderLength {
		return fmt.Errorf("req: TLS record length is %d, got %d payload bytes", recordLength, len(raw)-tlsRecordHeaderLength)
	}
	if recordLength < tlsHandshakeHeaderLength {
		return fmt.Errorf("req: TLS handshake record is too short: %d bytes", recordLength)
	}
	if raw[tlsRecordHeaderLength] != tlsClientHelloType {
		return fmt.Errorf("req: TLS handshake type %d is not ClientHello", raw[tlsRecordHeaderLength])
	}

	handshakeLength := int(raw[6])<<16 | int(raw[7])<<8 | int(raw[8])
	if handshakeLength != recordLength-tlsHandshakeHeaderLength {
		return fmt.Errorf("req: TLS ClientHello length is %d, got %d body bytes", handshakeLength, recordLength-tlsHandshakeHeaderLength)
	}
	return nil
}
