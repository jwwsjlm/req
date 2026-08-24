package http2

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"net/http/httptrace"

	transportopts "github.com/jwwsjlm/req/v3/internal/transport"
)

type tlsHookTestConn struct {
	net.Conn
}

func (c *tlsHookTestConn) ConnectionState() tls.ConnectionState {
	return tls.ConnectionState{HandshakeComplete: true}
}

func (c *tlsHookTestConn) Handshake() error {
	return nil
}

func (c *tlsHookTestConn) HandshakeContext(context.Context) error {
	return nil
}

func TestCustomTLSHandshakeRejectsConnectionWithoutTLSStateSupport(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	transport := &Transport{Options: &transportopts.Options{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return clientConn, nil
		},
		TLSHandshakeContext: func(_ context.Context, _ string, conn net.Conn) (net.Conn, *tls.ConnectionState, error) {
			return conn, &tls.ConnectionState{HandshakeComplete: true}, nil
		},
	}}

	_, err := transport.dialTLSWithContext(context.Background(), "tcp", "example.test:443", &tls.Config{})
	if err == nil {
		t.Fatal("custom TLS hook connection without ConnectionState support was accepted")
	}
	if !strings.Contains(err.Error(), "TLS state support") {
		t.Fatalf("unexpected custom TLS hook error: %v", err)
	}
}

func TestCustomTLSHandshakeIndependentSuccessClosesPlainConn(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	replacement, replacementPeer := net.Pipe()
	defer replacementPeer.Close()

	transport := &Transport{Options: &transportopts.Options{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return clientConn, nil
		},
		TLSHandshakeContext: func(_ context.Context, _ string, conn net.Conn) (net.Conn, *tls.ConnectionState, error) {
			if err := conn.Close(); err != nil {
				return nil, nil, err
			}
			return &tlsHookTestConn{Conn: replacement}, &tls.ConnectionState{HandshakeComplete: true}, nil
		},
	}}

	conn, err := transport.dialTLSWithContext(context.Background(), "tcp", "example.test:443", &tls.Config{})
	if err != nil {
		t.Fatalf("custom TLS handshake: %v", err)
	}
	defer conn.Close()

	serverConn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := serverConn.Read(make([]byte, 1)); err == nil {
		t.Fatal("hook returned an independent connection without closing plainConn")
	}
}

func TestCustomTLSHandshakeTimeoutDoesNotWaitForUncooperativeHook(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	replacement, replacementPeer := net.Pipe()
	defer replacementPeer.Close()

	releaseHook := make(chan struct{})
	hookReturned := make(chan struct{})
	transport := &Transport{Options: &transportopts.Options{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return clientConn, nil
		},
		TLSHandshakeTimeout: 20 * time.Millisecond,
		TLSHandshakeContext: func(context.Context, string, net.Conn) (net.Conn, *tls.ConnectionState, error) {
			<-releaseHook
			defer close(hookReturned)
			return &tlsHookTestConn{Conn: replacement}, &tls.ConnectionState{HandshakeComplete: true}, nil
		},
	}}

	var starts atomic.Int32
	var dones atomic.Int32
	trace := &httptrace.ClientTrace{
		TLSHandshakeStart: func() { starts.Add(1) },
		TLSHandshakeDone:  func(tls.ConnectionState, error) { dones.Add(1) },
	}
	ctx := httptrace.WithClientTrace(context.Background(), trace)
	started := time.Now()
	_, err := transport.dialTLSWithContext(ctx, "tcp", "example.test:443", &tls.Config{})
	if err != (tlsHandshakeTimeoutError{}) {
		t.Fatalf("custom TLS handshake error = %v, want timeout", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("custom TLS timeout waited for an uncooperative hook: %v", elapsed)
	}
	if starts.Load() != 1 || dones.Load() != 1 {
		t.Fatalf("TLS trace callbacks = start:%d done:%d, want 1 each", starts.Load(), dones.Load())
	}

	close(releaseHook)
	select {
	case <-hookReturned:
	case <-time.After(time.Second):
		t.Fatal("TLS hook did not return after release")
	}
	replacementPeer.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := replacementPeer.Read(make([]byte, 1)); err == nil {
		t.Fatal("late replacement connection was not closed")
	}
}
