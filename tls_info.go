package req

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
)

// TLSInfo contains TLS and leaf certificate details from a response.
type TLSInfo struct {
	CommonName               string
	DNSNames                 []string
	Emails                   []string
	IssuerCommonName         string
	IssuerOrganizations      []string
	Organizations            []string
	ServerName               string
	FingerprintSHA256        string
	FingerprintSHA256OpenSSL string
	Version                  string
}

// TLSInfo returns TLS and certificate details for HTTPS responses.
func (r *Response) TLSInfo() *TLSInfo {
	if r == nil || r.Response == nil || r.Response.TLS == nil {
		return nil
	}
	return newTLSInfo(r.Response.TLS)
}

// TLSGrabber is an alias of TLSInfo, named after surf's similar helper.
func (r *Response) TLSGrabber() *TLSInfo {
	return r.TLSInfo()
}

func newTLSInfo(cs *tls.ConnectionState) *TLSInfo {
	if cs == nil || len(cs.PeerCertificates) == 0 {
		return nil
	}
	cert := cs.PeerCertificates[0]
	sum := sha256.Sum256(cert.Raw)
	return &TLSInfo{
		CommonName:               cert.Subject.CommonName,
		DNSNames:                 cloneSlice(cert.DNSNames),
		Emails:                   cloneSlice(cert.EmailAddresses),
		IssuerCommonName:         cert.Issuer.CommonName,
		IssuerOrganizations:      cloneSlice(cert.Issuer.Organization),
		Organizations:            cloneSlice(cert.Subject.Organization),
		ServerName:               cs.ServerName,
		FingerprintSHA256:        hex.EncodeToString(sum[:]),
		FingerprintSHA256OpenSSL: opensslFingerprint(sum[:]),
		Version:                  tlsVersionName(cs.Version),
	}
}

func tlsVersionName(version uint16) string {
	switch version {
	case 0x0300:
		return "SSL30"
	case tls.VersionTLS10:
		return "TLS10"
	case tls.VersionTLS11:
		return "TLS11"
	case tls.VersionTLS12:
		return "TLS12"
	case tls.VersionTLS13:
		return "TLS13"
	default:
		return ""
	}
}

func opensslFingerprint(fp []byte) string {
	var buf bytes.Buffer
	for i, b := range fp {
		if i > 0 {
			buf.WriteByte(':')
		}
		_, _ = fmt.Fprintf(&buf, "%02X", b)
	}
	return buf.String()
}
