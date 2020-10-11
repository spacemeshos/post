package persistence

import (
	"os"
	"sort"
	"strconv"
	"strings"
)

type numericalSorter []os.FileInfo

// A compile time check to ensure that numericalSorter fully implements sort.Interface.
var _ sort.Interface = (*numericalSorter)(nil)

func (s numericalSorter) Len() int      { return len(s) }
func (s numericalSorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s numericalSorter) Less(i, j int) bool {
	pathA := s[i].Name()
	pathB := s[j].Name()

	// Get the integer values of each filename, placed after the delimiter.
	a, err1 := strconv.ParseInt(pathA[strings.Index(pathA, "-")+1:], 10, 64)
	b, err2 := strconv.ParseInt(pathB[strings.Index(pathB, "-")+1:], 10, 64)

	// If any were not numbers, sort lexicographically.
	if err1 != nil || err2 != nil {
		return pathA < pathB
	}

	return a < b
}
