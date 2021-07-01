package persistence

import (
	"io/ioutil"
	"os"
)

func NumBytesWritten(dir string, predicate func(os.FileInfo) bool) (uint64, error) {
	allFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	includedFiles := make([]os.FileInfo, 0)
	for _, file := range allFiles {
		if predicate(file) {
			includedFiles = append(includedFiles, file)
		}
	}

	if len(includedFiles) == 0 {
		return 0, nil
	}

	var numBytesWritten uint64
	for _, file := range includedFiles {
		numBytesWritten += uint64(file.Size())
	}

	return numBytesWritten, nil
}
