package header

import "strings"

const (
	DefaultUserAgent     = "req/v3 (https://github.com/jwwsjlm/req)"
	UserAgent            = "User-Agent"
	Location             = "Location"
	ContentType          = "Content-Type"
	PlainTextContentType = "text/plain; charset=utf-8"
	JsonContentType      = "application/json; charset=utf-8"
	XmlContentType       = "text/xml; charset=utf-8"
	FormContentType      = "application/x-www-form-urlencoded"
	WwwAuthenticate      = "WWW-Authenticate"
	Authorization        = "Authorization"
	HeaderOderKey        = "__header_order__"
	PseudoHeaderOderKey  = "__pseudo_header_order__"
)

var reqWriteExcludeHeader = map[string]bool{
	// Host is :authority, already sent.
	// Content-Length is automatic.
	"host":           true,
	"content-length": true,
	// Per 8.1.2.2 Connection-Specific Header
	// Fields, don't send connection-specific
	// fields. We have already checked if any
	// are error-worthy so just ignore the rest.
	"connection":        true,
	"proxy-connection":  true,
	"transfer-encoding": true,
	"upgrade":           true,
	"keep-alive":        true,
	// Ignore header order keys which is only used internally.
	HeaderOderKey:       true,
	PseudoHeaderOderKey: true,
}

// IsExcluded 判断字段是否由协议单独处理或禁止写入普通 Header 列表。
func IsExcluded(key string) bool {
	return reqWriteExcludeHeader[strings.ToLower(key)]
}
