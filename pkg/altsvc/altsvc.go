package altsvc

import (
	"sync"
	"time"
)

// AltSvcJar is default implementation of Jar, which stores
// AltSvc in memory.
type AltSvcJar struct {
	entries map[string]*AltSvc
	mu      sync.Mutex
}

// NewAltSvcJar creates an in-memory Alt-Svc jar that implements Jar.
// NewAltSvcJar 创建一个实现 Jar 接口的内存 Alt-Svc 存储。
func NewAltSvcJar() *AltSvcJar {
	return &AltSvcJar{
		entries: make(map[string]*AltSvc),
	}
}

// GetAltSvc returns the unexpired Alt-Svc entry for addr, or nil when absent or expired.
// GetAltSvc 返回 addr 对应且未过期的 Alt-Svc 条目；不存在或已过期时返回 nil。
func (j *AltSvcJar) GetAltSvc(addr string) *AltSvc {
	if addr == "" {
		return nil
	}
	as, ok := j.entries[addr]
	if !ok {
		return nil
	}
	now := time.Now()
	j.mu.Lock()
	defer j.mu.Unlock()
	if as.Expire.Before(now) { // expired
		delete(j.entries, addr)
		return nil
	}
	return as
}

// SetAltSvc stores the Alt-Svc entry for addr; an empty address is ignored.
// SetAltSvc 保存 addr 对应的 Alt-Svc 条目；空地址会被忽略。
func (j *AltSvcJar) SetAltSvc(addr string, as *AltSvc) {
	if addr == "" {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.entries[addr] = as
}

// AltSvc is the parsed alt-svc.
type AltSvc struct {
	// Protocol is the alt-svc proto, e.g. h3.
	Protocol string
	// Host is the alt-svc's host, could be empty if
	// it's the same host as the raw request.
	Host string
	// Port is the alt-svc's port.
	Port string
	// Expire is the time that the alt-svc should expire.
	Expire time.Time
}
