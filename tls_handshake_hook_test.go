package req

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"net/http/httptrace"
)

func TestCustomTLSHandshakeRejectsIncompleteResult(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	replacement, replacementPeer := net.Pipe()
	defer replacementPeer.Close()

	transport := T()
	transport.TLSHandshakeContext = func(context.Context, string, net.Conn) (net.Conn, *tls.ConnectionState, error) {
		return replacement, nil, nil
	}
	pconn := &persistConn{t: transport, conn: clientConn}

	if err := transport.customTlsHandshake(context.Background(), nil, "example.test", pconn); err == nil || !strings.Contains(err.Error(), "incomplete result") {
		t.Fatal("incomplete TLS handshake hook result was accepted")
	}
	replacementPeer.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := replacementPeer.Read(make([]byte, 1)); err == nil {
		t.Fatal("replacement connection from an incomplete hook result was not closed")
	}
}

func TestCustomTLSHandshakeIndependentSuccessClosesPlainConn(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	replacement, replacementPeer := net.Pipe()
	defer replacementPeer.Close()

	transport := T()
	transport.TLSHandshakeContext = func(_ context.Context, _ string, conn net.Conn) (net.Conn, *tls.ConnectionState, error) {
		if err := conn.Close(); err != nil {
			return nil, nil, err
		}
		return replacement, &tls.ConnectionState{HandshakeComplete: true}, nil
	}
	pconn := &persistConn{t: transport, conn: clientConn}
	if err := transport.customTlsHandshake(context.Background(), nil, "example.test", pconn); err != nil {
		t.Fatalf("custom TLS handshake: %v", err)
	}
	defer pconn.conn.Close()

	serverConn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := serverConn.Read(make([]byte, 1)); err == nil {
		t.Fatal("hook returned an independent connection without closing plainConn")
	}
}

func TestCustomTLSHandshakeTimeoutCancelsHook(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	transport := T()
	transport.TLSHandshakeTimeout = 20 * time.Millisecond
	hookDone := make(chan struct{})
	transport.TLSHandshakeContext = func(_ context.Context, _ string, conn net.Conn) (net.Conn, *tls.ConnectionState, error) {
		defer close(hookDone)
		var b [1]byte
		_, err := conn.Read(b[:])
		return nil, nil, err
	}
	pconn := &persistConn{t: transport, conn: clientConn}

	var starts atomic.Int32
	var dones atomic.Int32
	trace := &httptrace.ClientTrace{
		TLSHandshakeStart: func() { starts.Add(1) },
		TLSHandshakeDone:  func(tls.ConnectionState, error) { dones.Add(1) },
	}
	err := transport.customTlsHandshake(context.Background(), trace, "example.test", pconn)
	if err != (tlsHandshakeTimeoutError{}) {
		t.Fatalf("custom TLS handshake error = %v, want timeout", err)
	}
	select {
	case <-hookDone:
	case <-time.After(time.Second):
		t.Fatal("closing the plain connection did not release the TLS hook")
	}
	if starts.Load() != 1 || dones.Load() != 1 {
		t.Fatalf("TLS trace callbacks = start:%d done:%d, want 1 each", starts.Load(), dones.Load())
	}
}

func TestCustomTLSHandshakeTimeoutDoesNotWaitForUncooperativeHook(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	replacement, replacementPeer := net.Pipe()
	defer replacementPeer.Close()

	releaseHook := make(chan struct{})
	hookReturned := make(chan struct{})
	transport := T()
	transport.TLSHandshakeTimeout = 20 * time.Millisecond
	transport.TLSHandshakeContext = func(context.Context, string, net.Conn) (net.Conn, *tls.ConnectionState, error) {
		<-releaseHook
		defer close(hookReturned)
		return replacement, &tls.ConnectionState{HandshakeComplete: true}, nil
	}
	pconn := &persistConn{t: transport, conn: clientConn}

	started := time.Now()
	err := transport.customTlsHandshake(context.Background(), nil, "example.test", pconn)
	if err != (tlsHandshakeTimeoutError{}) {
		t.Fatalf("custom TLS handshake error = %v, want timeout", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("custom TLS timeout waited for an uncooperative hook: %v", elapsed)
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
