package initialization

import (
	"errors"
	"fmt"
)

var (
	ErrAlreadyInitializing          = errors.New("already initializing")
	ErrCannotResetWhileInitializing = errors.New("cannot reset while initializing")
	ErrStateMetadataFileMissing     = errors.New("metadata file is missing")
)

type ErrReferenceLabelMismatch struct {
	Index      uint64
	Commitment []byte

	Expected []byte
	Actual   []byte
}

func (e ErrReferenceLabelMismatch) Error() string {
	return fmt.Sprintf("reference label mismatch at %d with commitment %x: expected %x, actual %x", e.Index, e.Commitment, e.Expected, e.Actual)
}
