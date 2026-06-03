package req

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"
)

// DNSOverTLSProvider describes a DNS-over-TLS upstream.
type DNSOverTLSProvider struct {
	// ServerName is used for the DoT TLS SNI and certificate verification.
	ServerName string
	// Addresses are IP:port endpoints, usually ending in :853.
	Addresses []string
}

var (
	DNSOverTLSCloudflare = DNSOverTLSProvider{
		ServerName: "1dot1dot1dot1.cloudflare-dns.com",
		Addresses:  []string{"1.1.1.1:853", "1.0.0.1:853"},
	}
	DNSOverTLSGoogle = DNSOverTLSProvider{
		ServerName: "dns.google",
		Addresses:  []string{"8.8.8.8:853", "8.8.4.4:853"},
	}
	DNSOverTLSQuad9 = DNSOverTLSProvider{
		ServerName: "dns.quad9.net",
		Addresses:  []string{"9.9.9.9:853", "149.112.112.112:853"},
	}
	DNSOverTLSAdGuard = DNSOverTLSProvider{
		ServerName: "dns.adguard-dns.com",
		Addresses:  []string{"94.140.14.14:853", "94.140.15.15:853"},
	}
	DNSOverTLSAli = DNSOverTLSProvider{
		ServerName: "dns.alidns.com",
		Addresses:  []string{"223.5.5.5:853", "223.6.6.6:853"},
	}
)

// SetDNSResolver sets the resolver used by default dialing and HTTP/3.
func (c *Client) SetDNSResolver(resolver *net.Resolver) *Client {
	c.Transport.SetDNSResolver(resolver)
	return c
}

// SetDNSOverTLS configures DNS-over-TLS with the given provider.
func (c *Client) SetDNSOverTLS(provider DNSOverTLSProvider) *Client {
	c.Transport.SetDNSOverTLS(provider)
	return c
}

// SetDNSOverTLSCloudflare configures Cloudflare DNS-over-TLS.
func (c *Client) SetDNSOverTLSCloudflare() *Client {
	return c.SetDNSOverTLS(DNSOverTLSCloudflare)
}

// SetDNSOverTLSGoogle configures Google DNS-over-TLS.
func (c *Client) SetDNSOverTLSGoogle() *Client {
	return c.SetDNSOverTLS(DNSOverTLSGoogle)
}

// SetDNSOverTLSQuad9 configures Quad9 DNS-over-TLS.
func (c *Client) SetDNSOverTLSQuad9() *Client {
	return c.SetDNSOverTLS(DNSOverTLSQuad9)
}

// SetDNSOverTLSAdGuard configures AdGuard DNS-over-TLS.
func (c *Client) SetDNSOverTLSAdGuard() *Client {
	return c.SetDNSOverTLS(DNSOverTLSAdGuard)
}

// SetDNSOverTLSAli configures AliDNS DNS-over-TLS.
func (c *Client) SetDNSOverTLSAli() *Client {
	return c.SetDNSOverTLS(DNSOverTLSAli)
}

// SetDNSResolver sets the resolver used by default dialing and HTTP/3.
func (t *Transport) SetDNSResolver(resolver *net.Resolver) *Transport {
	t.Resolver = resolver
	if t.t3 != nil {
		t.t3.Resolver = resolver
	}
	return t
}

// SetDNSOverTLS configures DNS-over-TLS with the given provider.
func (t *Transport) SetDNSOverTLS(provider DNSOverTLSProvider) *Transport {
	return t.SetDNSResolver(NewDNSOverTLSResolver(provider))
}

// SetDNSOverTLSCloudflare configures Cloudflare DNS-over-TLS.
func (t *Transport) SetDNSOverTLSCloudflare() *Transport {
	return t.SetDNSOverTLS(DNSOverTLSCloudflare)
}

// SetDNSOverTLSGoogle configures Google DNS-over-TLS.
func (t *Transport) SetDNSOverTLSGoogle() *Transport {
	return t.SetDNSOverTLS(DNSOverTLSGoogle)
}

// SetDNSOverTLSQuad9 configures Quad9 DNS-over-TLS.
func (t *Transport) SetDNSOverTLSQuad9() *Transport {
	return t.SetDNSOverTLS(DNSOverTLSQuad9)
}

// SetDNSOverTLSAdGuard configures AdGuard DNS-over-TLS.
func (t *Transport) SetDNSOverTLSAdGuard() *Transport {
	return t.SetDNSOverTLS(DNSOverTLSAdGuard)
}

// SetDNSOverTLSAli configures AliDNS DNS-over-TLS.
func (t *Transport) SetDNSOverTLSAli() *Transport {
	return t.SetDNSOverTLS(DNSOverTLSAli)
}

// NewDNSOverTLSResolver creates a net.Resolver that resolves names through DoT.
func NewDNSOverTLSResolver(provider DNSOverTLSProvider) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial:     dnsOverTLSDialer(provider),
	}
}

func dnsOverTLSDialer(provider DNSOverTLSProvider) func(context.Context, string, string) (net.Conn, error) {
	addresses := cloneSlice(provider.Addresses)
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		if provider.ServerName == "" {
			return nil, errors.New("req: DNS-over-TLS provider ServerName is empty")
		}
		if len(addresses) == 0 {
			return nil, errors.New("req: DNS-over-TLS provider Addresses is empty")
		}

		var lastErr error
		var d net.Dialer
		for _, address := range addresses {
			conn, err := d.DialContext(ctx, "tcp", address)
			if err != nil {
				lastErr = err
				continue
			}
			if tc, ok := conn.(*net.TCPConn); ok {
				_ = tc.SetKeepAlive(true)
				_ = tc.SetKeepAlivePeriod(3 * time.Minute)
			}
			return tls.Client(conn, &tls.Config{
				ServerName:         provider.ServerName,
				ClientSessionCache: tls.NewLRUClientSessionCache(32),
			}), nil
		}
		return nil, lastErr
	}
}
