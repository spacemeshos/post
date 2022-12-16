package initialization

import (
	"os"

	"github.com/spacemeshos/post/shared"
)

type DiskState struct {
	datadir      string
	bitsPerLabel uint
}

func NewDiskState(datadir string, bitsPerLabel uint) *DiskState {
	return &DiskState{datadir, bitsPerLabel}
}

func (d *DiskState) NumLabelsWritten() (uint64, error) {
	numBytesWritten, err := d.NumBytesWritten()
	if err != nil {
		return 0, err
	}

	return shared.NumLabels(numBytesWritten, d.bitsPerLabel), nil
}

func (d *DiskState) NumBytesWritten() (uint64, error) {
	files, err := GetFiles(d.datadir, shared.IsInitFile)
	if err != nil {
		return 0, err
	}

	var numBytesWritten uint64
	for _, file := range files {
		numBytesWritten += uint64(file.Size())
	}

	return numBytesWritten, nil
}

func (d *DiskState) NumFilesWritten() (int, error) {
	files, err := GetFiles(d.datadir, shared.IsInitFile)
	if err != nil {
		return 0, err
	}

	return len(files), err
}

func GetFiles(dir string, predicate func(os.FileInfo) bool) ([]os.FileInfo, error) {
	allFiles, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
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

	return includedFiles, nil
}
