package tls

import (
	"context"
	"crypto/tls"
	"net"
)

// Conn is the recommended interface for the connection
// returned by the DailTLS function (Client.SetDialTLS,
// Transport.DialTLSContext), so that the TLS handshake negotiation
// can automatically decide whether to use HTTP2 or HTTP1 (ALPN).
// If this interface is not implemented, HTTP1 will be used by default.
type Conn interface {
	net.Conn
	// ConnectionState returns basic TLS details about the connection.
	// ConnectionState 返回连接的基本 TLS 状态信息。
	ConnectionState() tls.ConnectionState
	// Handshake runs the client or server handshake
	// protocol if it has not yet been run.
	//
	// Most uses of this package need not call Handshake explicitly: the
	// first Read or Write will call it automatically.
	//
	// For control over canceling or setting a timeout on a handshake, use
	// HandshakeContext or the Dialer's DialContext method instead.
	//
	// Handshake 在尚未握手时执行客户端或服务端 TLS 握手；通常首次读写会自动触发。
	Handshake() error

	// HandshakeContext runs the client or server handshake
	// protocol if it has not yet been run.
	//
	// The provided Context must be non-nil. If the context is canceled before
	// the handshake is complete, the handshake is interrupted and an error is returned.
	// Once the handshake has completed, cancellation of the context will not affect the
	// connection.
	//
	// Most uses of this package need not call HandshakeContext explicitly: the
	// first Read or Write will call it automatically.
	//
	// HandshakeContext 使用非 nil context 执行 TLS 握手，并可在握手完成前取消。
	HandshakeContext(ctx context.Context) error
}
