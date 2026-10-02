// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license described in THIRD_PARTY_NOTICES.md.

package http2

import (
	"crypto/tls"
	"os"
	"strings"

	"golang.org/x/net/http/httpguts"
)

var (
	VerboseLogs    bool
	logFrameWrites bool
	logFrameReads  bool
)

func init() {
	e := os.Getenv("GODEBUG")
	if strings.Contains(e, "http2debug=1") {
		VerboseLogs = true
	}
	if strings.Contains(e, "http2debug=2") {
		VerboseLogs = true
		logFrameWrites = true
		logFrameReads = true
	}
}

const (
	// ClientPreface is the string that must be sent by new
	// connections from clients.
	ClientPreface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

	// NextProtoTLS is the NPN/ALPN protocol negotiated during
	// HTTP/2's TLS setup.
	NextProtoTLS = "h2"

	// https://httpwg.org/specs/rfc7540.html#SettingValues
	initialHeaderTableSize = 4096

	initialWindowSize = 65535 // 6.9.2 Initial Flow Control Window Size
)

var clientPreface = []byte(ClientPreface)

// validWireHeaderFieldName reports whether v is a valid header field
// name (key). See httpguts.ValidHeaderName for the base rules.
//
// Further, http2 says:
//
//	"Just as in HTTP/1.x, header field names are strings of ASCII
//	characters that are compared in a case-insensitive
//	fashion. However, header field names MUST be converted to
//	lowercase prior to their encoding in HTTP/2. "
func validWireHeaderFieldName(v string) bool {
	if len(v) == 0 {
		return false
	}
	for _, r := range v {
		if !httpguts.IsTokenRune(r) {
			return false
		}
		if 'A' <= r && r <= 'Z' {
			return false
		}
	}
	return true
}

// mustUint31 保留负数检查；int32 本身已保证值不会超过 31 位正数上限。
func mustUint31(v int32) uint32 {
	if v < 0 {
		panic("out of range")
	}
	return uint32(v)
}

type httpError struct {
	_       incomparable
	msg     string
	timeout bool
}

func (e *httpError) Error() string { return e.msg }

func (e *httpError) Timeout() bool { return e.timeout }

func (e *httpError) Temporary() bool { return true }

var errH2Timeout error = &httpError{msg: "http2: timeout awaiting response headers", timeout: true}

type connectionStater interface {
	ConnectionState() tls.ConnectionState
}

// validPseudoPath reports whether v is a valid :path pseudo-header
// value. It must be either:
//
//	*) a non-empty string starting with '/'
//	*) the string '*', for OPTIONS requests.
//
// For now this is only used a quick check for deciding when to clean
// up Opaque URLs before sending requests from the Transport.
// See golang.org/issue/16847
//
// We used to enforce that the path also didn't start with "//", but
// Google's GFE accepts such paths and Chrome sends them, so ignore
// that part of the spec. See golang.org/issue/19103.
func validPseudoPath(v string) bool {
	return (len(v) > 0 && v[0] == '/') || v == "*"
}

// incomparable is a zero-width, non-comparable type. Adding it to a struct
// makes that struct also non-comparable, and generally doesn't add
// any size (as long as it's first).
type incomparable [0]func()
