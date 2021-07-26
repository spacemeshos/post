package persistence

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type numericalSorter []os.FileInfo

// A compile time check to ensure that numericalSorter fully implements the sort.Interface interface.
var _ sort.Interface = (*numericalSorter)(nil)

func (s numericalSorter) Len() int      { return len(s) }
func (s numericalSorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s numericalSorter) Less(i, j int) bool {
	nameA := s[i].Name()
	nameA = strings.TrimSuffix(nameA, filepath.Ext(nameA))

	nameB := s[j].Name()
	nameB = strings.TrimSuffix(nameB, filepath.Ext(nameB))

	// Get the integer values of each filename, placed after the delimiter.
	a, err1 := strconv.ParseInt(nameA[strings.Index(nameA, "_")+1:], 10, 64)
	b, err2 := strconv.ParseInt(nameB[strings.Index(nameB, "_")+1:], 10, 64)

	// If any were not numbers, sort lexicographically.
	if err1 != nil || err2 != nil {
		return nameA < nameB
	}

	return a < b
}
