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
	numBytesWritten, err := d.NumBytesWritten()
	if err != nil {
		return 0, err
	}

	return shared.NumLabels(numBytesWritten, d.bitsPerLabel), nil
}

func (d *DiskState) NumBytesWritten() (uint64, error) {
	return persistence.NumBytesWritten(d.datadir, shared.IsInitFile)
}
