package shared

import "sort"

type Set map[uint64]bool

func (s Set) AsSortedSlice() []uint64 {
	var ret []uint64
	for key, value := range s {
		if value {
			ret = append(ret, key)
		}
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

func SetOf(members ...uint64) Set {
	ret := make(Set)
	for _, member := range members {
		ret[member] = true
	}
	return ret
}
