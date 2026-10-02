// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license described in THIRD_PARTY_NOTICES.md.

// Package godebug makes the settings in the $GODEBUG environment variable
// available to other packages. These settings are often used for compatibility
// tweaks, when we need to change a default behavior but want to let users
// opt back in to the original. For example GODEBUG=http2server=0 disables
// HTTP/2 support in the net/http server.
//
// In typical usage, code should declare a Setting as a global
// and then call Value each time the current setting value is needed:
//
//	var http2server = godebug.New("http2server")
//
//	func ServeConn(c net.Conn) {
//		if http2server.Value() == "0" {
//			disallow HTTP/2
//			...
//		}
//		...
//	}
//
// Each time a non-default setting causes a change in program behavior,
// code should call [Setting.IncNonDefault] to increment a counter that can
// be reported by [runtime/metrics.Read].
// Note that counters used with IncNonDefault must be added to
// various tables in other packages. See the [Setting.IncNonDefault]
// documentation for details.
package godebug

// Note: Be careful about new imports here. Any package
// that internal/godebug imports cannot itself import internal/godebug,
// meaning it cannot introduce a GODEBUG setting of its own.
// We keep imports to the absolute bare minimum.
import (
	"sync"
	"sync/atomic"
	_ "unsafe" // go:linkname

	"github.com/jwwsjlm/req/v3/internal/bisect"
	"github.com/jwwsjlm/req/v3/internal/godebugs"
)

// A Setting is a single setting in the $GODEBUG environment variable.
type Setting struct {
	name string
	once sync.Once
	*setting
}

type setting struct {
	value          atomic.Pointer[value]
	nonDefaultOnce sync.Once
	nonDefault     atomic.Uint64
	info           *godebugs.Info
}

type value struct {
	text   string
	bisect *bisect.Matcher
}

// New returns a new Setting for the $GODEBUG setting with the given name.
//
// GODEBUGs meant for use by end users must be listed in ../godebugs/table.go,
// which is used for generating and checking various documentation.
// If the name is not listed in that table, New will succeed but calling Value
// on the returned Setting will panic.
// To disable that panic for access to an undocumented setting,
// prefix the name with a #, as in godebug.New("#gofsystrace").
// The # is a signal to New but not part of the key used in $GODEBUG.
func New(name string) *Setting {
	return &Setting{name: name}
}

// Name returns the name of the setting.
func (s *Setting) Name() string {
	if s.name != "" && s.name[0] == '#' {
		return s.name[1:]
	}
	return s.name
}

// Undocumented reports whether this is an undocumented setting.
func (s *Setting) Undocumented() bool {
	return s.name != "" && s.name[0] == '#'
}

// String returns a printable form for the setting: name=value.
func (s *Setting) String() string {
	return s.Name() + "=" + s.Value()
}

// IncNonDefault increments the non-default behavior counter
// associated with the given setting.
// This counter is exposed in the runtime/metrics value
// /godebug/non-default-behavior/<name>:events.
//
// Note that Value must be called at least once before IncNonDefault.
func (s *Setting) IncNonDefault() {
	s.nonDefaultOnce.Do(s.register)
	s.nonDefault.Add(1)
}

func (s *Setting) register() {
	if s.info == nil || s.info.Opaque {
		panic("godebug: unexpected IncNonDefault of " + s.name)
	}
}

// cache is a cache of all the GODEBUG settings,
// a locked map[string]*atomic.Pointer[string].
//
// All Settings with the same name share a single
// *atomic.Pointer[string], so that when GODEBUG
// changes only that single atomic string pointer
// needs to be updated.
//
// A name appears in the values map either if it is the
// name of a Setting for which Value has been called
// at least once, or if the name has ever appeared in
// a name=value pair in the $GODEBUG environment variable.
// Once entered into the map, the name is never removed.
var cache sync.Map // name string -> value *atomic.Pointer[string]

var empty value

// Value returns the current value for the GODEBUG setting s.
//
// Value maintains an internal cache that is synchronized
// with changes to the $GODEBUG environment variable,
// making Value efficient to call as frequently as needed.
// Clients should therefore typically not attempt their own
// caching of Value's result.
func (s *Setting) Value() string {
	s.once.Do(func() {
		s.setting = lookup(s.Name())
		if s.info == nil && !s.Undocumented() {
			panic("godebug: Value of name not listed in godebugs.All: " + s.name)
		}
	})
	v := *s.value.Load()
	if v.bisect != nil && !v.bisect.Stack(&stderr) {
		return ""
	}
	return v.text
}

// lookup returns the unique *setting value for the given name.
func lookup(name string) *setting {
	if v, ok := cache.Load(name); ok {
		return v.(*setting)
	}
	s := new(setting)
	s.info = godebugs.Lookup(name)
	s.value.Store(&empty)
	if v, loaded := cache.LoadOrStore(name, s); loaded {
		// Lost race: someone else created it. Use theirs.
		return v.(*setting)
	}

	return s
}

type runtimeStderr struct{}

var stderr runtimeStderr

func (*runtimeStderr) Write(b []byte) (int, error) {
	return len(b), nil
}
