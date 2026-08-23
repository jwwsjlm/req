package header

import (
	"sort"
	"strings"
)

type KeyValues struct {
	Key    string
	Values []string
}

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
	s := &sorter{
		ranks: ranks,
		kvs:   kvs,
	}
	sort.Sort(s)
}

func headerOrderRanksLinear(kvs []KeyValues, orderedKeys []string) []int {
	ranks := make([]int, len(kvs))
	for i := range kvs {
		ranks[i] = -1
		for rank, key := range orderedKeys {
			// HTTP field names are ASCII and case-insensitive. EqualFold avoids
			// allocating a normalized copy for typical browser-sized order lists.
			if strings.EqualFold(kvs[i].Key, key) {
				ranks[i] = rank
			}
		}
	}
	return ranks
}

func headerOrderRanksMap(kvs []KeyValues, orderedKeys []string) []int {
	order := make(map[string]int, len(orderedKeys))
	for rank, key := range orderedKeys {
		order[strings.ToLower(key)] = rank
	}
	ranks := make([]int, len(kvs))
	for i := range kvs {
		rank, ok := order[strings.ToLower(kvs[i].Key)]
		if !ok {
			rank = -1
		}
		ranks[i] = rank
	}
	return ranks
}
