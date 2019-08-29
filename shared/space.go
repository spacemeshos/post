package shared

import (
	"fmt"
	"github.com/ricochet2200/go-disk-usage/du"
)

// ValidateSpace indicates whether a given space amount is valid.
func ValidateSpace(space uint64) error {
	if space > MaxSpace {
		return fmt.Errorf("space (%d) is greater than the supported max (%d)", space, MaxSpace)
	}
	if !IsPowerOfTwo(space) {
		return fmt.Errorf("space (%d) must be a power of 2", space)
	}
	if space%LabelGroupSize != 0 {
		return fmt.Errorf("space (%d) must be a multiple of %d", space, LabelGroupSize)
	}

	return nil
}

// ValidateSpace indicates whether a given space and numFiles are valid, assuming the space param validity.
func ValidateNumFiles(space uint64, numFiles uint64) error {
	if !IsPowerOfTwo(numFiles) {
		return fmt.Errorf("number of files (%d) must be a power of 2", numFiles)
	}

	if numFiles > uint64(MaxNumFiles) {
		return fmt.Errorf("number of files (%d) is greater than the supported max (%d)", numFiles, MaxNumFiles)
	}

	fileSize := space / numFiles
	if fileSize < LabelGroupSize {
		return fmt.Errorf("file size (%d) must be greater than %d", fileSize, LabelGroupSize)
	}

	return nil
}

// NumLabelGroups returns the number of label groups of a given space param, assuming its validity.
func NumLabelGroups(space uint64) uint64 {
	return uint64(space) / LabelGroupSize
}

func AvailableSpace(path string) uint64 {
	usage := du.NewDiskUsage(path)
	return usage.Available()
}
