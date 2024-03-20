package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "post.h"
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"go.uber.org/zap"
)

type VerifyPosOptions struct {
	fromFile *uint32
	toFile   *uint32
	fraction float64
	logger   *zap.Logger
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

func VerifyPosWithLogger(logger *zap.Logger) VerifyPosOptionsFunc {
	return func(o *VerifyPosOptions) error {
		o.logger = logger
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

	if opts.logger != nil {
		setLogCallback(opts.logger)
	}

	dataDirPtr := C.CString(dataDir)
	defer C.free(unsafe.Pointer(dataDirPtr))

	result := C.verify_pos(
		dataDirPtr,
		(*C.uint32_t)(opts.fromFile),
		(*C.uint32_t)(opts.toFile),
		C.double(opts.fraction), scryptParams,
	)
	switch result.tag {
	case C.VerifyResult_Ok:
		return nil
	case C.VerifyResult_Invalid:
		result := castBytes[C.VerifyPosResult_Invalid_Body](result.anon0[:])
		return fmt.Errorf("%w: file %d contains invalid label at offset %d", ErrInvalidPos, result.file, result.offset)
	case C.VerifyResult_InvalidArgument:
		return ErrInvalidArgument
	default:
		return ErrUnknown
	}
}

// cast bytes to a pointer of type T
func castBytes[T any](body []byte) *T {
	var v T
	if len(body) < int(unsafe.Sizeof(v)) {
		panic("invalid size")
	}
	return (*T)(unsafe.Pointer(&body[0]))
}
