// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license described in THIRD_PARTY_NOTICES.md.
package http2

import "time"

// A timer is a time.Timer, as an interface which can be replaced in tests.
type timer = interface {
	C() <-chan time.Time
	Reset(d time.Duration) bool
	Stop() bool
}

// timeTimer adapts a time.Timer to the timer interface.
type timeTimer struct {
	*time.Timer
}

func (t timeTimer) C() <-chan time.Time { return t.Timer.C }
