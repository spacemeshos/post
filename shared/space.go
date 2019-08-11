package shared

import (
	"fmt"
	"github.com/ricochet2200/go-disk-usage/du"
)

// ValidateSpace validates whether the given space amount is valid.
func ValidateSpace(space uint64) error {
	if space > MaxSpace {
		return fmt.Errorf("space (%d) is greater than the supported max (%d)", space, MaxSpace)
	}
	if space%LabelGroupSize != 0 {
		return fmt.Errorf("space (%d) must be a multiple of %d", space, LabelGroupSize)
	}

	return nil
}

func ValidateFileSize(space uint64, filesize uint64) error {
	if space%filesize != 0 {
		return fmt.Errorf("space (%d) must be a multiple of filesize (%d)", space, filesize)
	}
	if filesize < LabelGroupSize {
		return fmt.Errorf("filesize (%d) must be greater than %d", filesize, LabelGroupSize)
	}
	if space/filesize > uint64(MaxNumFiles) {
		return fmt.Errorf("number of files (%d) is greater than the supported max (%d)", space/filesize, MaxNumFiles)
	}

	return nil
}

func NumFiles(space uint64, filesize uint64) (int, error) {
	if err := ValidateFileSize(space, filesize); err != nil {
		return 0, err
	}

	return int(space / filesize), nil
}

// NumLabelGroups returns the number of label groups of a given space amount.
func NumLabelGroups(space uint64) uint64 {
	return uint64(space) / LabelGroupSize
}

func AvailableSpace(path string) uint64 {
	usage := du.NewDiskUsage(path)
	return usage.Available()
}
