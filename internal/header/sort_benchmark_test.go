package header

import (
	"fmt"
	"net/textproto"
	"testing"
)

var benchmarkFirstHeader string

func BenchmarkSortKeyValues(b *testing.B) {
	input := []KeyValues{
		{Key: "accept", Values: []string{"*/*"}},
		{Key: "accept-encoding", Values: []string{"gzip, deflate, br"}},
		{Key: "accept-language", Values: []string{"zh-CN,zh;q=0.9"}},
		{Key: "cache-control", Values: []string{"no-cache"}},
		{Key: "content-type", Values: []string{"application/json"}},
		{Key: "cookie", Values: []string{"session=benchmark"}},
		{Key: "host", Values: []string{"example.com"}},
		{Key: "origin", Values: []string{"https://example.com"}},
		{Key: "pragma", Values: []string{"no-cache"}},
		{Key: "priority", Values: []string{"u=1, i"}},
		{Key: "referer", Values: []string{"https://example.com/"}},
		{Key: "sec-ch-ua", Values: []string{"benchmark"}},
		{Key: "sec-ch-ua-mobile", Values: []string{"?0"}},
		{Key: "sec-ch-ua-platform", Values: []string{"Windows"}},
		{Key: "sec-fetch-dest", Values: []string{"empty"}},
		{Key: "sec-fetch-mode", Values: []string{"cors"}},
		{Key: "sec-fetch-site", Values: []string{"same-origin"}},
		{Key: "te", Values: []string{"trailers"}},
		{Key: "upgrade-insecure-requests", Values: []string{"1"}},
		{Key: "user-agent", Values: []string{"benchmark"}},
		{Key: "x-api-key", Values: []string{"secret"}},
		{Key: "x-request-id", Values: []string{"request-id"}},
	}
	order := []string{
		"host", "connection", "content-length", "pragma", "cache-control",
		"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "upgrade-insecure-requests",
		"user-agent", "accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest",
		"referer", "accept-encoding", "accept-language", "cookie",
	}
	b.Run("Lowercase", func(b *testing.B) {
		benchmarkSortKeyValues(b, input, order)
	})
	b.Run("Canonical", func(b *testing.B) {
		canonical := make([]KeyValues, len(input))
		for i, kv := range input {
			canonical[i] = kv
			canonical[i].Key = textproto.CanonicalMIMEHeaderKey(kv.Key)
		}
		benchmarkSortKeyValues(b, canonical, order)
	})
}

func benchmarkSortKeyValues(b *testing.B, input []KeyValues, order []string) {
	work := make([]KeyValues, len(input))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(work, input)
		SortKeyValues(work, order)
		benchmarkFirstHeader = work[0].Key
	}
}

func BenchmarkSortKeyValuesLarge(b *testing.B) {
	const count = 128
	input := make([]KeyValues, count)
	order := make([]string, count)
	for i := 0; i < count; i++ {
		input[i] = KeyValues{Key: fmt.Sprintf("x-header-%03d", count-i-1)}
		order[i] = fmt.Sprintf("x-header-%03d", i)
	}
	benchmarkSortKeyValues(b, input, order)
}
