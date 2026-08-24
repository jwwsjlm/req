// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license described in THIRD_PARTY_NOTICES.md.

//go:build (js && wasm) || wasip1

package req

import (
	"context"
	"net"
)

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return nil
}
