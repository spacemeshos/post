package initialization

import (
	"github.com/spacemeshos/post/persistence"
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
	numBytesWritten, _, err := d.NumBytesWritten()
	if err != nil {
		return 0, err
	}

	return shared.NumLabels(numBytesWritten, d.bitsPerLabel), nil
}

func (d *DiskState) NumBytesWritten() (uint64, int, error) {
	return persistence.NumBytesWritten(d.datadir, shared.IsInitFile)
}

func (d *DiskState) NumFiles() (int, error) {
	_, numFiles, err := d.NumBytesWritten()
	return numFiles, err
}
