package header

import (
	"fmt"
	"math/rand"
	"net/textproto"
	"reflect"
	"sort"
	"testing"
)

type legacySorter struct {
	order map[string]int
	kvs   []KeyValues
}

func (s *legacySorter) Len() int      { return len(s.kvs) }
func (s *legacySorter) Swap(i, j int) { s.kvs[i], s.kvs[j] = s.kvs[j], s.kvs[i] }
func (s *legacySorter) Less(i, j int) bool {
	if index, ok := s.order[textproto.CanonicalMIMEHeaderKey(s.kvs[i].Key)]; ok {
		i = index
	}
	if index, ok := s.order[textproto.CanonicalMIMEHeaderKey(s.kvs[j].Key)]; ok {
		j = index
	}
	return i < j
}

func legacySort(kvs []KeyValues, orderedKeys []string) {
	order := make(map[string]int)
	for i, key := range orderedKeys {
		order[textproto.CanonicalMIMEHeaderKey(key)] = i
	}
	sort.Sort(&legacySorter{order: order, kvs: kvs})
}

func TestSortKeyValuesPreservesOrderingBehavior(t *testing.T) {
	tests := []struct {
		name  string
		kvs   []KeyValues
		order []string
	}{
		{
			name: "mixed canonical and noncanonical keys",
			kvs: []KeyValues{
				{Key: "x-request-id"},
				{Key: "Accept"},
				{Key: "host"},
				{Key: "user-agent"},
				{Key: "X-Trace"},
				{Key: "accept-language"},
			},
			order: []string{"Host", "User-Agent", "Accept", "Accept-Language"},
		},
		{
			name: "duplicates and unordered keys",
			kvs: []KeyValues{
				{Key: "Cookie", Values: []string{"a=1"}},
				{Key: "X-Extra"},
				{Key: "cookie", Values: []string{"b=2"}},
				{Key: "Accept"},
				{Key: "X-Last"},
			},
			order: []string{"Accept", "Cookie"},
		},
		{
			name: "duplicate ordered keys use last rank",
			kvs: []KeyValues{
				{Key: "Accept"},
				{Key: "Cookie"},
				{Key: "User-Agent"},
			},
			order: []string{"accept", "cookie", "ACCEPT", "user-agent"},
		},
		{
			name:  "empty order",
			kvs:   []KeyValues{{Key: "B"}, {Key: "A"}},
			order: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := append([]KeyValues(nil), tt.kvs...)
			want := append([]KeyValues(nil), tt.kvs...)
			SortKeyValues(got, tt.order)
			legacySort(want, tt.order)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("SortKeyValues() = %#v, want legacy ordering %#v", got, want)
			}
		})
	}
}

func TestSortKeyValuesMatchesLegacyAcrossHeaderCases(t *testing.T) {
	headerNames := []string{
		"Accept", "Accept-Encoding", "Accept-Language", "Authorization",
		"Cache-Control", "Content-Length", "Content-Type", "Cookie", "Host",
		"Origin", "Priority", "Referer", "Sec-Fetch-Dest", "Sec-Fetch-Mode",
		"Sec-Fetch-Site", "User-Agent", "X-API-Key", "X-Request-ID",
	}
	rng := rand.New(rand.NewSource(1))

	for iteration := 0; iteration < 200; iteration++ {
		permutation := rng.Perm(len(headerNames))
		kvs := make([]KeyValues, len(headerNames))
		for i, index := range permutation {
			kvs[i] = KeyValues{Key: randomHeaderCase(rng, headerNames[index])}
		}

		orderPermutation := rng.Perm(len(headerNames))
		orderLength := 1 + rng.Intn(len(headerNames))
		order := make([]string, orderLength)
		for i, index := range orderPermutation[:orderLength] {
			order[i] = randomHeaderCase(rng, headerNames[index])
		}

		got := append([]KeyValues(nil), kvs...)
		want := append([]KeyValues(nil), kvs...)
		SortKeyValues(got, order)
		legacySort(want, order)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: SortKeyValues() = %#v, want legacy ordering %#v", iteration, got, want)
		}
	}
}

func TestSortKeyValuesMapPathMatchesLegacy(t *testing.T) {
	const count = 48
	kvs := make([]KeyValues, count)
	order := make([]string, count)
	for i := 0; i < count; i++ {
		kvs[i] = KeyValues{Key: randomHeaderCase(rand.New(rand.NewSource(int64(i+1))), fmt.Sprintf("X-Header-%03d", i))}
		order[i] = randomHeaderCase(rand.New(rand.NewSource(int64(count-i))), fmt.Sprintf("X-Header-%03d", count-i-1))
	}

	got := append([]KeyValues(nil), kvs...)
	want := append([]KeyValues(nil), kvs...)
	SortKeyValues(got, order)
	legacySort(want, order)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SortKeyValues() = %#v, want legacy ordering %#v", got, want)
	}
}

func randomHeaderCase(rng *rand.Rand, value string) string {
	b := []byte(value)
	for i, c := range b {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			if rng.Intn(2) == 0 {
				b[i] = c | 0x20
			} else {
				b[i] = c &^ 0x20
			}
		}
	}
	return string(b)
}
