package req

import (
	"net/url"
	"testing"
)

var benchmarkParsedURL *url.URL

func BenchmarkParseRequestURLClientQuery(b *testing.B) {
	c := C().
		SetBaseURL("https://api.example.com").
		SetCommonQueryParams(map[string]string{
			"account": "123456",
			"cursor":  "next-page-token",
			"expand":  "profile,settings",
			"filter":  "active",
			"lang":    "zh-CN",
			"limit":   "100",
			"region":  "cn-east-1",
			"sort":    "created_at",
		})
	r := c.R()
	r.RawURL = "/v1/users?source=benchmark"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := parseRequestURL(c, r); err != nil {
			b.Fatal(err)
		}
		benchmarkParsedURL = r.URL
	}
}

func BenchmarkParseRequestURLMergedQuery(b *testing.B) {
	c := C().
		SetBaseURL("https://api.example.com").
		SetCommonQueryParams(map[string]string{
			"account": "123456",
			"cursor":  "next-page-token",
			"expand":  "profile,settings",
			"filter":  "active",
			"lang":    "zh-CN",
			"limit":   "100",
			"region":  "cn-east-1",
			"sort":    "created_at",
		})
	r := c.R().
		SetQueryParam("cursor", "request-cursor").
		SetQueryParam("filter", "verified").
		AddQueryParams("tag", "go", "http", "client")
	r.RawURL = "/v1/users?source=benchmark"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := parseRequestURL(c, r); err != nil {
			b.Fatal(err)
		}
		benchmarkParsedURL = r.URL
	}
}
