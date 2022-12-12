package persistence

import (
	"os"
)

func NumBytesWritten(dir string, predicate func(os.FileInfo) bool) (uint64, int, error) {
	allFiles, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	includedFiles := make([]os.FileInfo, 0)
	for _, file := range allFiles {
		info, err := file.Info()
		if err != nil {
			continue
		}

		if predicate(info) {
			includedFiles = append(includedFiles, info)
		}
	}

	if len(includedFiles) == 0 {
		return 0, 0, nil
	}

	var numBytesWritten uint64
	for _, file := range includedFiles {
		numBytesWritten += uint64(file.Size())
	}

	return numBytesWritten, len(includedFiles), nil
}
