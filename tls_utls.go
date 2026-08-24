package req

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	utls "github.com/refraction-networking/utls"
)

// utlsFingerprintConfig describes how future TLS connections build their
// ClientHello. It is immutable after installation so Client.Clone can safely
// share the description and bind a new handshake closure to the cloned client.
//
// utlsFingerprintConfig 描述后续 TLS 连接如何构造 ClientHello。安装后保持不可变，
// 因此 Client.Clone 可以安全共享这份描述，并为克隆后的 client 重新绑定握手闭包。
type utlsFingerprintConfig struct {
	clientHelloID utls.ClientHelloID
	apply         func(*uTLSConn) error
}

// uTLSConn adapts a uTLS UConn to the TLS connection interface used by req.
//
// uTLSConn 将 uTLS UConn 适配为 req 使用的 TLS 连接接口。
type uTLSConn struct {
	*utls.UConn
}

func (conn *uTLSConn) ConnectionState() tls.ConnectionState {
	return tlsConnectionStateFromUTLS(conn.Conn.ConnectionState())
}

func (c *Client) setTLSFingerprint(clientHelloID utls.ClientHelloID, apply func(*uTLSConn) error) *Client {
	c.Transport.setTLSFingerprint(&utlsFingerprintConfig{
		clientHelloID: cloneUTLSClientHelloID(clientHelloID),
		apply:         apply,
	})
	return c
}

func (t *Transport) setTLSFingerprint(profile *utlsFingerprintConfig) {
	t.tlsFingerprint = profile
	t.bindTLSFingerprint()
}

func cloneUTLSClientHelloID(id utls.ClientHelloID) utls.ClientHelloID {
	clone := id
	if id.Seed != nil {
		seed := *id.Seed
		clone.Seed = &seed
	}
	if id.Weights != nil {
		weights := *id.Weights
		clone.Weights = &weights
	}
	return clone
}

// bindTLSFingerprint binds the immutable fingerprint description to t. It is
// called again by Transport.Clone so the handshake reads the clone's TLS
// config rather than the TLS config of the transport that was cloned.
//
// bindTLSFingerprint 将不可变的指纹描述绑定到 t。Transport.Clone 会再次调用它，
// 使握手读取克隆 transport 自己的 TLS 配置，而不是被克隆来源的配置。
func (t *Transport) bindTLSFingerprint() {
	profile := t.tlsFingerprint
	if profile == nil {
		return
	}
	t.TLSHandshakeContext = func(ctx context.Context, endpointHost string, plainConn net.Conn) (net.Conn, *tls.ConnectionState, error) {
		// Clone at the connection boundary. A tls.Config must not be modified
		// after use, and this also keeps per-connection ServerName derivation local.
		// 在连接边界克隆配置。tls.Config 使用后不应再修改，同时可将每条连接的
		// ServerName 推导限制在当前握手内。
		baseConfig := cloneTLSConfig(t.TLSClientConfig)
		uconfig := tlsConfigToUTLS(baseConfig, endpointHost)
		clientHelloID := profile.clientHelloID
		apply := profile.apply
		presetPrepared := clientHelloID == utls.HelloGolang
		if apply == nil && !presetPrepared {
			// UTLSIdToSpec is the same source used internally for fixed parrot
			// presets. Applying that fresh spec as HelloCustom lets us restore the
			// caller's renegotiation policy without building the ClientHello twice.
			// 固定 parrot preset 内部同样来自 UTLSIdToSpec。按 HelloCustom 应用
			// fresh spec，可在不重复构造 ClientHello 的前提下恢复重协商策略。
			if spec, err := utls.UTLSIdToSpec(clientHelloID); err == nil {
				setUTLSSpecRenegotiationPolicy(&spec, utls.RenegotiationSupport(baseConfig.Renegotiation))
				clientHelloID = utls.HelloCustom
				uconfig.PreferSkipResumptionOnNilExtension = true
				apply = func(conn *uTLSConn) error { return conn.ApplyPreset(&spec) }
				presetPrepared = true
			}
		}
		uconn := &uTLSConn{UConn: utls.UClient(
			plainConn,
			uconfig,
			clientHelloID,
		)}
		if apply != nil {
			if err := apply(uconn); err != nil {
				return nil, nil, err
			}
		}
		// A browser preset may carry a renegotiation_info extension whose
		// internal uTLS setting overwrites Config.Renegotiation. Materialize the
		// preset, then restore the caller's policy without changing the encoded
		// initial extension bytes.
		// 浏览器 preset 的 renegotiation_info 扩展可能反向覆盖配置。先实例化
		// preset，再恢复调用方策略；初始扩展的编码字节不会因此改变。
		if !presetPrepared {
			if err := prepareUTLSRenegotiationPolicy(uconn, utls.RenegotiationSupport(baseConfig.Renegotiation)); err != nil {
				return nil, nil, err
			}
		}
		if err := uconn.HandshakeContext(ctx); err != nil {
			return nil, nil, err
		}
		// Marshaling resets uTLS's extension field to its mimicry default. Put
		// the policy back before exposing the connection so a later TLS 1.2
		// renegotiation attempt cannot re-enable it.
		// uTLS 编码时会把扩展字段重置为伪装默认值；返回连接前再次恢复，避免
		// 后续 TLS 1.2 重协商尝试重新开启该策略。
		setUTLSRenegotiationPolicy(uconn, utls.RenegotiationSupport(baseConfig.Renegotiation))
		state := tlsConnectionStateFromUTLS(uconn.Conn.ConnectionState())
		return uconn, &state, nil
	}
}

func prepareUTLSRenegotiationPolicy(conn *uTLSConn, policy utls.RenegotiationSupport) error {
	if len(conn.Extensions) == 0 {
		if err := conn.BuildHandshakeStateWithoutSession(); err != nil {
			return err
		}
	}
	setUTLSRenegotiationPolicy(conn, policy)
	return nil
}

func setUTLSSpecRenegotiationPolicy(spec *utls.ClientHelloSpec, policy utls.RenegotiationSupport) {
	for _, extension := range spec.Extensions {
		if renegotiation, ok := extension.(*utls.RenegotiationInfoExtension); ok {
			renegotiation.Renegotiation = policy
		}
	}
}

func setUTLSRenegotiationPolicy(conn *uTLSConn, policy utls.RenegotiationSupport) bool {
	found := false
	for _, extension := range conn.Extensions {
		if renegotiation, ok := extension.(*utls.RenegotiationInfoExtension); ok {
			renegotiation.Renegotiation = policy
			found = true
		}
	}
	return found
}

// tlsConfigToUTLS converts the client-side fields of crypto/tls.Config into a
// uTLS config without silently dropping certificate verification or mTLS
// settings. Server-only callbacks with incompatible state types are omitted.
//
// tlsConfigToUTLS 将 crypto/tls.Config 的客户端字段转换为 uTLS 配置，避免静默
// 丢失证书校验或 mTLS 设置。状态类型不兼容且仅服务端使用的回调不会被复制。
func tlsConfigToUTLS(config *tls.Config, endpointHost string) *utls.Config {
	if config == nil {
		config = &tls.Config{}
	}
	uconfig := &utls.Config{
		Rand:                               config.Rand,
		Time:                               config.Time,
		Certificates:                       tlsCertificatesToUTLS(config.Certificates),
		RootCAs:                            config.RootCAs,
		NextProtos:                         config.NextProtos,
		ServerName:                         tlsServerName(config.ServerName, endpointHost),
		ClientAuth:                         utls.ClientAuthType(config.ClientAuth),
		ClientCAs:                          config.ClientCAs,
		InsecureSkipVerify:                 config.InsecureSkipVerify,
		CipherSuites:                       config.CipherSuites,
		PreferServerCipherSuites:           config.PreferServerCipherSuites,
		SessionTicketsDisabled:             config.SessionTicketsDisabled,
		SessionTicketKey:                   config.SessionTicketKey,
		MinVersion:                         config.MinVersion,
		MaxVersion:                         config.MaxVersion,
		CurvePreferences:                   tlsCurvesToUTLS(config.CurvePreferences),
		DynamicRecordSizingDisabled:        config.DynamicRecordSizingDisabled,
		Renegotiation:                      utls.RenegotiationSupport(config.Renegotiation),
		KeyLogWriter:                       config.KeyLogWriter,
		EncryptedClientHelloConfigList:     config.EncryptedClientHelloConfigList,
		EncryptedClientHelloKeys:           tlsECHKeysToUTLS(config.EncryptedClientHelloKeys),
		PreferSkipResumptionOnNilExtension: config.ClientSessionCache != nil,
	}
	if config.GetClientCertificate != nil {
		uconfig.GetClientCertificate = tlsGetClientCertificateToUTLS(config.GetClientCertificate)
	}
	if config.VerifyPeerCertificate != nil {
		uconfig.VerifyPeerCertificate = config.VerifyPeerCertificate
	}
	if config.VerifyConnection != nil {
		uconfig.VerifyConnection = func(state utls.ConnectionState) error {
			return callTLSConnectionStateCallback("VerifyConnection", config.VerifyConnection, tlsConnectionStateFromUTLS(state))
		}
	}
	if config.EncryptedClientHelloRejectionVerify != nil {
		uconfig.EncryptedClientHelloRejectionVerify = func(state utls.ConnectionState) error {
			return callTLSConnectionStateCallback("EncryptedClientHelloRejectionVerify", config.EncryptedClientHelloRejectionVerify, tlsConnectionStateFromUTLS(state))
		}
	}
	if config.ClientSessionCache != nil {
		uconfig.ClientSessionCache = &utlsClientSessionCache{cache: config.ClientSessionCache}
	}
	return uconfig
}

// callTLSConnectionStateCallback converts callback panics into handshake
// errors. In particular, a ConnectionState reconstructed from uTLS cannot
// carry crypto/tls's private keying-material exporter, so calling
// ExportKeyingMaterial would otherwise crash the process.
//
// callTLSConnectionStateCallback 将回调 panic 转换为握手错误。尤其是从 uTLS
// 重建的 ConnectionState 无法携带 crypto/tls 私有的 keying-material exporter，
// 因此直接调用 ExportKeyingMaterial 原本会导致进程崩溃。
func callTLSConnectionStateCallback(name string, callback func(tls.ConnectionState) error, state tls.ConnectionState) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("req: tls.Config.%s panicked with converted uTLS ConnectionState: %v", name, recovered)
		}
	}()
	return callback(state)
}

// tlsServerName preserves an explicit ServerName. Otherwise it accepts either
// a host, a bracketed IPv6 literal, or a host:port endpoint without truncating
// an unbracketed IPv6 address at its final colon.
//
// tlsServerName 优先保留显式 ServerName；否则兼容主机名、带方括号的 IPv6，
// 以及 host:port，并且不会在最后一个冒号处错误截断无方括号 IPv6 地址。
func tlsServerName(configured, endpointHost string) string {
	if configured != "" {
		return configured
	}
	if host, _, err := net.SplitHostPort(endpointHost); err == nil {
		return host
	}
	return strings.TrimSuffix(strings.TrimPrefix(endpointHost, "["), "]")
}

func tlsCertificatesToUTLS(certificates []tls.Certificate) []utls.Certificate {
	if len(certificates) == 0 {
		return nil
	}
	converted := make([]utls.Certificate, len(certificates))
	for i := range certificates {
		converted[i] = tlsCertificateToUTLS(&certificates[i])
	}
	return converted
}

func tlsCertificateToUTLS(certificate *tls.Certificate) utls.Certificate {
	if certificate == nil {
		return utls.Certificate{}
	}
	return utls.Certificate{
		Certificate:                  certificate.Certificate,
		PrivateKey:                   certificate.PrivateKey,
		SupportedSignatureAlgorithms: tlsSignatureSchemesToUTLS(certificate.SupportedSignatureAlgorithms),
		OCSPStaple:                   certificate.OCSPStaple,
		SignedCertificateTimestamps:  certificate.SignedCertificateTimestamps,
		Leaf:                         certificate.Leaf,
	}
}

func tlsGetClientCertificateToUTLS(get func(*tls.CertificateRequestInfo) (*tls.Certificate, error)) func(*utls.CertificateRequestInfo) (*utls.Certificate, error) {
	return func(info *utls.CertificateRequestInfo) (*utls.Certificate, error) {
		var request *tls.CertificateRequestInfo
		if info != nil {
			request = &tls.CertificateRequestInfo{
				AcceptableCAs:    info.AcceptableCAs,
				SignatureSchemes: utlsSignatureSchemesToTLS(info.SignatureSchemes),
				Version:          info.Version,
			}
		}
		// crypto/tls does not expose a constructor for preserving the private
		// CertificateRequestInfo context. Public selection fields are preserved.
		// crypto/tls 未提供保留 CertificateRequestInfo 私有 context 的构造器；
		// 这里保留证书选择所需的全部公开字段。
		certificate, err := get(request)
		if err != nil || certificate == nil {
			return nil, err
		}
		converted := tlsCertificateToUTLS(certificate)
		return &converted, nil
	}
}

func tlsSignatureSchemesToUTLS(values []tls.SignatureScheme) []utls.SignatureScheme {
	if len(values) == 0 {
		return nil
	}
	converted := make([]utls.SignatureScheme, len(values))
	for i, value := range values {
		converted[i] = utls.SignatureScheme(value)
	}
	return converted
}

func utlsSignatureSchemesToTLS(values []utls.SignatureScheme) []tls.SignatureScheme {
	if len(values) == 0 {
		return nil
	}
	converted := make([]tls.SignatureScheme, len(values))
	for i, value := range values {
		converted[i] = tls.SignatureScheme(value)
	}
	return converted
}

func tlsCurvesToUTLS(values []tls.CurveID) []utls.CurveID {
	if len(values) == 0 {
		return nil
	}
	converted := make([]utls.CurveID, len(values))
	for i, value := range values {
		converted[i] = utls.CurveID(value)
	}
	return converted
}

func tlsECHKeysToUTLS(keys []tls.EncryptedClientHelloKey) []utls.EncryptedClientHelloKey {
	if len(keys) == 0 {
		return nil
	}
	converted := make([]utls.EncryptedClientHelloKey, len(keys))
	for i, key := range keys {
		converted[i] = utls.EncryptedClientHelloKey{
			Config:      key.Config,
			PrivateKey:  key.PrivateKey,
			SendAsRetry: key.SendAsRetry,
		}
	}
	return converted
}

// tlsConnectionStateFromUTLS centralizes the public state shared by uTLS and
// crypto/tls. uTLS v1.8.2 doesn't expose CurveID or HelloRetryRequest, so those
// newer crypto/tls fields retain their zero values.
//
// tlsConnectionStateFromUTLS 集中转换 uTLS 与 crypto/tls 共有的公开状态。
// uTLS v1.8.2 未公开 CurveID 和 HelloRetryRequest，因此这些新字段保持零值。
func tlsConnectionStateFromUTLS(state utls.ConnectionState) tls.ConnectionState {
	return tls.ConnectionState{
		Version:                     state.Version,
		HandshakeComplete:           state.HandshakeComplete,
		DidResume:                   state.DidResume,
		CipherSuite:                 state.CipherSuite,
		NegotiatedProtocol:          state.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  state.NegotiatedProtocolIsMutual,
		ServerName:                  state.ServerName,
		PeerCertificates:            state.PeerCertificates,
		VerifiedChains:              state.VerifiedChains,
		SignedCertificateTimestamps: state.SignedCertificateTimestamps,
		OCSPResponse:                state.OCSPResponse,
		TLSUnique:                   state.TLSUnique,
		ECHAccepted:                 state.ECHAccepted,
	}
}

// utlsClientSessionCache serializes session state through the public APIs of
// both TLS implementations. A state is reused only when the receiving parser
// accepts it; incompatible version-specific encodings become a safe cache miss.
//
// utlsClientSessionCache 通过两套 TLS 实现的公开 API 序列化会话状态。只有目标
// 解析器接受的状态才会复用；版本特有且不兼容的编码会安全降级为缓存未命中。
type utlsClientSessionCache struct {
	cache tls.ClientSessionCache
}

func (c *utlsClientSessionCache) Get(sessionKey string) (*utls.ClientSessionState, bool) {
	state, ok := c.cache.Get(sessionKey)
	if !ok || state == nil {
		return nil, ok
	}
	ticket, session, err := state.ResumptionState()
	if err != nil || session == nil {
		return nil, false
	}
	encoded, err := session.Bytes()
	if err != nil {
		return nil, false
	}
	usession, err := utls.ParseSessionState(encoded)
	if err != nil {
		return nil, false
	}
	converted, err := utls.NewResumptionState(append([]byte(nil), ticket...), usession)
	if err != nil {
		return nil, false
	}
	return converted, true
}

func (c *utlsClientSessionCache) Put(sessionKey string, state *utls.ClientSessionState) {
	if state == nil {
		c.cache.Put(sessionKey, nil)
		return
	}
	ticket, session, err := state.ResumptionState()
	if err != nil || session == nil {
		c.cache.Put(sessionKey, nil)
		return
	}
	encoded, err := session.Bytes()
	if err != nil {
		c.cache.Put(sessionKey, nil)
		return
	}
	tsession, err := tls.ParseSessionState(encoded)
	if err != nil {
		c.cache.Put(sessionKey, nil)
		return
	}
	converted, err := tls.NewResumptionState(append([]byte(nil), ticket...), tsession)
	if err != nil {
		c.cache.Put(sessionKey, nil)
		return
	}
	c.cache.Put(sessionKey, converted)
}
