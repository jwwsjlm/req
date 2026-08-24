package altsvc

// Jar is a container of AltSvc.
type Jar interface {
	// SetAltSvc stores an Alt-Svc entry for an address.
	// SetAltSvc 为地址保存 Alt-Svc 条目。
	SetAltSvc(addr string, as *AltSvc)
	// GetAltSvc returns the Alt-Svc entry for an address.
	// GetAltSvc 返回地址对应的 Alt-Svc 条目。
	GetAltSvc(addr string) *AltSvc
}
