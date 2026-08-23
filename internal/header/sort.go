package header

import (
	"net/textproto"
	"sort"

	"github.com/jwwsjlm/req/v3/internal/ascii"
	"golang.org/x/net/http/httpguts"
)

// KeyValues holds a header field name and its values.
// KeyValues 保存一个 Header 字段名及其值。
type KeyValues struct {
	Key    string
	Values []string
}

// sorter keeps precomputed ranks aligned with kvs while sorting.
// sorter 在排序期间保持预计算顺序与 kvs 元素同步。
type sorter struct {
	ranks []int
	kvs   []KeyValues
}

const linearHeaderOrderSearchLimit = 1024

func (s *sorter) Len() int { return len(s.kvs) }
func (s *sorter) Swap(i, j int) {
	s.kvs[i], s.kvs[j] = s.kvs[j], s.kvs[i]
	s.ranks[i], s.ranks[j] = s.ranks[j], s.ranks[i]
}
func (s *sorter) Less(i, j int) bool {
	if rank := s.ranks[i]; rank >= 0 {
		i = rank
	}
	if rank := s.ranks[j]; rank >= 0 {
		j = rank
	}
	return i < j
}

// SortKeyValues reorders kvs in place according to orderedKeys while
// preserving the package's legacy header-name matching behavior.
//
// SortKeyValues 按 orderedKeys 原地调整 kvs，并保持旧实现的字段名匹配语义。
func SortKeyValues(kvs []KeyValues, orderedKeys []string) {
	if len(kvs) < 2 || len(orderedKeys) == 0 {
		return
	}

	var ranks []int
	if len(kvs) <= linearHeaderOrderSearchLimit/len(orderedKeys) {
		ranks = headerOrderRanksLinear(kvs, orderedKeys)
	} else {
		ranks = headerOrderRanksMap(kvs, orderedKeys)
	}
	sort.Sort(&sorter{ranks: ranks, kvs: kvs})
}

func headerOrderRanksLinear(kvs []KeyValues, orderedKeys []string) []int {
	ranks := make([]int, len(kvs))
	for i := range kvs {
		ranks[i] = -1
		for rank, key := range orderedKeys {
			if canonicalHeaderKeyEqual(kvs[i].Key, key) {
				ranks[i] = rank
			}
		}
	}
	return ranks
}

// canonicalHeaderKeyEqual is allocation-free but intentionally has the same
// equality semantics as applying textproto.CanonicalMIMEHeaderKey to both
// values. Invalid field names are only equal when their original bytes match;
// valid names use ASCII-only case folding.
//
// canonicalHeaderKeyEqual 不分配内存，但与两侧调用 CanonicalMIMEHeaderKey
// 后比较的语义一致：非法字段名只能原字节相等，合法字段名仅按 ASCII 忽略大小写。
func canonicalHeaderKeyEqual(a, b string) bool {
	if a == b {
		return true
	}
	return ascii.EqualFold(a, b) && httpguts.ValidHeaderFieldName(a)
}

func headerOrderRanksMap(kvs []KeyValues, orderedKeys []string) []int {
	// Canonicalize once before sorting. The legacy implementation called the
	// same standard-library function from Less for every comparison. Do not size
	// the map directly from len(orderedKeys): repeated caller-provided keys must
	// not trigger a potentially huge eager allocation. The bounded hint still
	// covers normal browser-sized order lists without changing map contents.
	//
	// 排序前只规范化一次。旧实现会在 Less 的每次比较中调用同一个标准库函数。
	// 不直接按 orderedKeys 长度预分配，避免大量重复键触发不必要的大块内存申请；
	// 有上限的提示仍覆盖常见浏览器 Header 顺序列表，且不改变 map 内容。
	capacityHint := min(len(orderedKeys), len(kvs), linearHeaderOrderSearchLimit)
	order := make(map[string]int, capacityHint)
	for rank, key := range orderedKeys {
		order[textproto.CanonicalMIMEHeaderKey(key)] = rank
	}
	ranks := make([]int, len(kvs))
	for i := range kvs {
		rank, ok := order[textproto.CanonicalMIMEHeaderKey(kvs[i].Key)]
		if !ok {
			rank = -1
		}
		ranks[i] = rank
	}
	return ranks
}
