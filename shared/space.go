package shared

import (
	"fmt"
)

// ValidateSpace validates whether the given space amount is valid.
func ValidateSpace(space uint64) error {
	if space > MaxSpace {
		return fmt.Errorf("space (%d) is greater than the supported max (%d)", space, MaxSpace)
	}
	if uint64(space)%LabelGroupSize != 0 {
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
	if space/filesize > MaxNumOfFiles {
		return fmt.Errorf("number of files (%d) is greater than the supported max (%d)", space/filesize, MaxNumOfFiles)
	}

	return nil
}

func NumOfFiles(space uint64, filesize uint64) (int, error) {
	if err := ValidateFileSize(space, filesize); err != nil {
		return 0, err
	}

	return int(space / filesize), nil
}

// NumOfLabelGroups returns the number of label groups of a given space amount.
func NumOfLabelGroups(space uint64) uint64 {
	return uint64(space) / LabelGroupSize
}
