package req

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"time"

	"github.com/jwwsjlm/req/v3/internal/http3"
	"github.com/quic-go/quic-go"
)

const defaultHTTP3AltSvcFailureCooldown = 30 * time.Second

const (
	// HTTP3SettingQpackMaxTableCapacity is SETTINGS_QPACK_MAX_TABLE_CAPACITY.
	HTTP3SettingQpackMaxTableCapacity uint64 = 0x01
	// HTTP3SettingMaxFieldSectionSize is SETTINGS_MAX_FIELD_SECTION_SIZE.
	HTTP3SettingMaxFieldSectionSize uint64 = 0x06
	// HTTP3SettingEnableConnectProtocol is SETTINGS_ENABLE_CONNECT_PROTOCOL.
	HTTP3SettingEnableConnectProtocol uint64 = 0x08
	// HTTP3SettingQpackBlockedStreams is SETTINGS_QPACK_BLOCKED_STREAMS.
	HTTP3SettingQpackBlockedStreams uint64 = 0x07
	// HTTP3SettingH3Datagram is SETTINGS_H3_DATAGRAM.
	HTTP3SettingH3Datagram uint64 = 0x33
	// HTTP3SettingH3DatagramDraft is the draft H3_DATAGRAM setting used by some browser profiles.
	HTTP3SettingH3DatagramDraft uint64 = 0xffd277
	// HTTP3SettingEnableWebTransport is SETTINGS_ENABLE_WEBTRANSPORT.
	HTTP3SettingEnableWebTransport uint64 = 0x2b603742
)

// SetHTTP3AdditionalSettings sets additional HTTP/3 SETTINGS values.
// Values are applied to future HTTP/3 transports and any currently enabled
// HTTP/3 transport.
func (t *Transport) SetHTTP3AdditionalSettings(settings map[uint64]uint64) *Transport {
	t.http3AdditionalSettings = cloneHTTP3Settings(settings)
	if t.t3 != nil {
		t.t3.AdditionalSettings = cloneHTTP3Settings(t.http3AdditionalSettings)
	}
	return t
}

// SetHTTP3AdditionalSetting sets one additional HTTP/3 SETTINGS value.
func (t *Transport) SetHTTP3AdditionalSetting(id, value uint64) *Transport {
	if t.http3AdditionalSettings == nil {
		t.http3AdditionalSettings = make(map[uint64]uint64)
	}
	t.http3AdditionalSettings[id] = value
	if t.t3 != nil {
		t.t3.AdditionalSettings = cloneHTTP3Settings(t.http3AdditionalSettings)
	}
	return t
}

// SetHTTP3Grease adds a randomized HTTP/3 GREASE setting.
func (t *Transport) SetHTTP3Grease() *Transport {
	id, value := http3GreaseSetting()
	return t.SetHTTP3AdditionalSetting(id, value)
}

// EnableHTTP3Datagrams enables HTTP/3 datagram support on the HTTP/3 and QUIC layers.
func (t *Transport) EnableHTTP3Datagrams() *Transport {
	t.http3EnableDatagrams = true
	if t.http3QUICConfig != nil && !t.http3QUICConfig.EnableDatagrams {
		t.http3QUICConfig = t.http3QUICConfig.Clone()
		t.http3QUICConfig.EnableDatagrams = true
	}
	if t.t3 != nil {
		t.t3.EnableDatagrams = true
		if t.t3.QUICConfig != nil && !t.t3.QUICConfig.EnableDatagrams {
			t.t3.QUICConfig = t.t3.QUICConfig.Clone()
			t.t3.QUICConfig.EnableDatagrams = true
		}
	}
	return t
}

// DisableHTTP3Datagrams disables HTTP/3 datagram support.
func (t *Transport) DisableHTTP3Datagrams() *Transport {
	t.http3EnableDatagrams = false
	if t.http3QUICConfig != nil && t.http3QUICConfig.EnableDatagrams {
		t.http3QUICConfig = t.http3QUICConfig.Clone()
		t.http3QUICConfig.EnableDatagrams = false
	}
	if t.t3 != nil {
		t.t3.EnableDatagrams = false
		if t.t3.QUICConfig != nil && t.t3.QUICConfig.EnableDatagrams {
			t.t3.QUICConfig = t.t3.QUICConfig.Clone()
			t.t3.QUICConfig.EnableDatagrams = false
		}
	}
	return t
}

// EnableHTTP3ExtendedConnect enables HTTP/3 Extended CONNECT (RFC 9220).
func (t *Transport) EnableHTTP3ExtendedConnect() *Transport {
	t.http3EnableExtendedConnect = true
	if t.t3 != nil {
		t.t3.EnableExtendedConnect = true
	}
	return t
}

// DisableHTTP3ExtendedConnect disables HTTP/3 Extended CONNECT (RFC 9220).
func (t *Transport) DisableHTTP3ExtendedConnect() *Transport {
	t.http3EnableExtendedConnect = false
	if t.t3 != nil {
		t.t3.EnableExtendedConnect = false
	}
	return t
}

// SetHTTP3MaxResponseHeaderBytes sets the HTTP/3 SETTINGS_MAX_FIELD_SECTION_SIZE
// value and response header read limit.
func (t *Transport) SetHTTP3MaxResponseHeaderBytes(max int) *Transport {
	t.http3MaxResponseHeaderBytes = max
	if t.t3 != nil {
		t.t3.MaxResponseHeaderBytes = max
	}
	return t
}

// SetHTTP3QUICConfig sets the QUIC config used by HTTP/3.
func (t *Transport) SetHTTP3QUICConfig(cfg *quic.Config) *Transport {
	if cfg == nil {
		t.http3QUICConfig = nil
		if t.t3 != nil {
			t.t3.QUICConfig = nil
		}
		return t
	}
	t.http3QUICConfig = cfg.Clone()
	if t.http3EnableDatagrams {
		t.http3QUICConfig.EnableDatagrams = true
	}
	if t.t3 != nil {
		t.t3.QUICConfig = t.http3QUICConfig.Clone()
	}
	return t
}

// SetHTTP3QUICPerformanceProfile applies a balanced QUIC config for HTTP/3
// connection reuse, address validation tokens, keepalive, and receive windows.
func (t *Transport) SetHTTP3QUICPerformanceProfile() *Transport {
	cfg := t.http3QUICProfileConfig()
	cfg.HandshakeIdleTimeout = 5 * time.Second
	cfg.MaxIdleTimeout = 45 * time.Second
	cfg.KeepAlivePeriod = 15 * time.Second
	cfg.InitialStreamReceiveWindow = 512 * 1024
	cfg.MaxStreamReceiveWindow = 8 * 1024 * 1024
	cfg.InitialConnectionReceiveWindow = 1024 * 1024
	cfg.MaxConnectionReceiveWindow = 24 * 1024 * 1024
	cfg.MaxIncomingStreams = -1
	cfg.MaxIncomingUniStreams = 100
	cfg.InitialPacketSize = 1200
	if cfg.TokenStore == nil {
		cfg.TokenStore = quic.NewLRUTokenStore(256, 4)
	}
	return t.SetHTTP3QUICConfig(cfg)
}

// SetHTTP3QUICChromeProfile applies Chrome-like QUIC performance defaults for HTTP/3.
func (t *Transport) SetHTTP3QUICChromeProfile() *Transport {
	return t.SetHTTP3QUICPerformanceProfile()
}

// SetHTTP3TLSClientConfig sets a TLS config used only by HTTP/3.
// If nil, HTTP/3 inherits the transport's TLSClientConfig.
func (t *Transport) SetHTTP3TLSClientConfig(cfg *tls.Config) *Transport {
	if cfg == nil {
		t.http3TLSClientConfig = nil
		t.syncHTTP3TLSClientConfig()
		return t
	}
	t.http3TLSClientConfig = cfg.Clone()
	t.syncHTTP3TLSClientConfig()
	return t
}

// SetHTTP3TLSChromeProfile applies Chrome-like HTTP/3 TLS constraints using
// Go's crypto/tls. uTLS ClientHello customization is only used for HTTP/1.1
// and HTTP/2.
func (t *Transport) SetHTTP3TLSChromeProfile() *Transport {
	cfg := t.http3TLSProfileConfig()
	cfg.CurvePreferences = []tls.CurveID{
		tls.X25519MLKEM768,
		tls.X25519,
		tls.CurveP256,
		tls.CurveP384,
	}
	return t.SetHTTP3TLSClientConfig(cfg)
}

// SetHTTP3TLSFirefoxProfile applies Firefox-like HTTP/3 TLS constraints using
// Go's crypto/tls. uTLS ClientHello customization is only used for HTTP/1.1
// and HTTP/2.
func (t *Transport) SetHTTP3TLSFirefoxProfile() *Transport {
	cfg := t.http3TLSProfileConfig()
	cfg.CurvePreferences = []tls.CurveID{
		tls.X25519,
		tls.CurveP256,
		tls.CurveP384,
	}
	return t.SetHTTP3TLSClientConfig(cfg)
}

// EnableHTTP3FallbackOnError allows forced HTTP/3 requests to retry with
// HTTP/2 or HTTP/1.1 when the HTTP/3 attempt fails before the request becomes
// unreplayable. It also applies to HTTP/3 selected through Alt-Svc.
func (t *Transport) EnableHTTP3FallbackOnError() *Transport {
	t.http3FallbackOnFailure = true
	return t
}

// DisableHTTP3FallbackOnError disables fallback for forced HTTP/3 requests.
func (t *Transport) DisableHTTP3FallbackOnError() *Transport {
	t.http3FallbackOnFailure = false
	return t
}

// SetHTTP3AltSvcFailureCooldown sets how long a failed Alt-Svc HTTP/3 endpoint
// is skipped after fallback. Zero keeps the default; a negative duration disables
// the cooldown.
func (t *Transport) SetHTTP3AltSvcFailureCooldown(cooldown time.Duration) *Transport {
	t.http3AltSvcFailureCooldown = cooldown
	return t
}

func (t *Transport) http3QUICProfileConfig() *quic.Config {
	if t.http3QUICConfig != nil {
		return t.http3QUICConfig.Clone()
	}
	return &quic.Config{}
}

func (t *Transport) http3TLSProfileConfig() *tls.Config {
	cfg := &tls.Config{}
	if current := t.activeHTTP3TLSClientConfig(); current != nil {
		cfg = current.Clone()
	}
	cfg.MinVersion = tls.VersionTLS13
	cfg.MaxVersion = tls.VersionTLS13
	cfg.NextProtos = []string{http3.NextProtoH3}
	if cfg.ClientSessionCache == nil {
		cfg.ClientSessionCache = tls.NewLRUClientSessionCache(64)
	}
	return cfg
}

func cloneHTTP3Settings(settings map[uint64]uint64) map[uint64]uint64 {
	if len(settings) == 0 {
		return nil
	}
	clone := make(map[uint64]uint64, len(settings))
	for id, value := range settings {
		clone[id] = value
	}
	return clone
}

func http3GreaseSetting() (uint64, uint64) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0x21, 0
	}
	const maxVarInt = uint64(1<<62) - 1
	maxn := (maxVarInt - 0x21) / 0x1f
	n := binary.LittleEndian.Uint64(b[:]) % maxn
	return 0x1f*n + 0x21, binary.BigEndian.Uint64(b[:]) & maxVarInt
}
