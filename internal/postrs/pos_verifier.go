package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "post.h"
import "C"

import (
	"errors"
	"unsafe"
)

type VerifyPosOptions struct {
	fromFile *uint32
	toFile   *uint32
	fraction float64
}

var ErrInvalidPos = errors.New("invalid POS")

type VerifyPosOptionsFunc func(*VerifyPosOptions) error

func FromFile(fromFile uint32) VerifyPosOptionsFunc {
	return func(o *VerifyPosOptions) error {
		o.fromFile = &fromFile
		return nil
	}
}

func ToFile(toFile uint32) VerifyPosOptionsFunc {
	return func(o *VerifyPosOptions) error {
		o.toFile = &toFile
		return nil
	}
}

func WithFraction(fraction float64) VerifyPosOptionsFunc {
	return func(o *VerifyPosOptions) error {
		o.fraction = fraction
		return nil
	}
}

func VerifyPos(dataDir string, scryptParams ScryptParams, o ...VerifyPosOptionsFunc) error {
	opts := &VerifyPosOptions{
		fraction: 5.0,
	}

	for _, opt := range o {
		if err := opt(opts); err != nil {
			return err
		}
	}

	dataDirPtr := C.CString(dataDir)
	defer C.free(unsafe.Pointer(dataDirPtr))

	result := C.verify_pos(dataDirPtr, (*C.uint32_t)(opts.fromFile), (*C.uint32_t)(opts.toFile), C.double(opts.fraction), scryptParams)
	switch result {
	case C.Ok:
		return nil
	case C.Invalid:
		return ErrInvalidPos
	case C.InvalidArgument:
		return ErrInvalidArgument
	default:
		return ErrUnknown
	}
}
