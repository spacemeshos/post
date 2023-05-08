package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "prover.h"
import "C"

import (
	"errors"
	"math"
	"unsafe"
)

// gpuMtx is an instance of deviceMutex that can be used to prevent concurrent calls
// to the same GPU (by ProviderID) from multiple goroutines.
var gpuMtx deviceMutex

// DeviceClass is an enum for the type of device (CPU or GPU).
type DeviceClass int

const (
	ClassCPU = DeviceClass((C.DeviceClass)(C.CPU))
	ClassGPU = DeviceClass((C.DeviceClass)(C.GPU))
)

// Provider is a struct that contains information about an OpenCL provider.
// libpostrs returns a list of these structs when calling cGetProviders().
// Each Provider is an OpenCL platform + Device combination.
type Provider struct {
	ID         uint
	Model      string
	DeviceType DeviceClass
}

func (c DeviceClass) String() string {
	switch c {
	case ClassCPU:
		return "CPU"
	case ClassGPU:
		return "GPU"
	default:
		return "Unknown"
	}
}

var (
	ErrInvalidProviderID = errors.New("invalid provider ID")

	ErrInvalidLabelsRange = errors.New("invalid labels range")
	ErrOpenCL             = errors.New("OpenCL error")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrFetchProviders     = errors.New("failed to fetch providers")
	ErrUnknown            = errors.New("unknown error")
)

const (
	// LabelLength is the length of the label in bytes.
	LabelLength = 16
)

// InitResultToError converts the return value of the C.initialize() function to a Go error.
func InitResultToError(retVal uint32) error {
	switch retVal {
	case C.InitializeOk:
		return nil
	case C.InitializeInvalidLabelsRange:
		return ErrInvalidLabelsRange
	case C.InitializeOclError:
		return ErrOpenCL
	case C.InitializeInvalidArgument:
		return ErrInvalidArgument
	case C.InitializeFailedToGetProviders:
		return ErrFetchProviders
	default:
		return ErrUnknown
	}
}

// cScryptPositions calls the C functions from libpostrs that create the labels
// and VRF proofs. It returns them as well as well as a possible error.
func cScryptPositions(opt *option) ([]byte, *uint64, error) {
	// TODO(mafa): disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace)

	if *opt.providerID != cCPUProviderID() {
		gpuMtx.Device(*opt.providerID).Lock()
		defer gpuMtx.Device(*opt.providerID).Unlock()
	}

	cProviderId := C.uint32_t(*opt.providerID)
	cN := C.uintptr_t(opt.n)
	cCommitment := C.CBytes(opt.commitment)
	defer C.free(cCommitment)
	cDifficulty := C.CBytes(opt.vrfDifficulty)
	defer C.free(cDifficulty)
	init := C.new_initializer(cProviderId, cN, (*C.uchar)(cCommitment), (*C.uchar)(cDifficulty))
	if init == nil {
		return nil, nil, ErrInvalidProviderID
	}
	defer C.free_initializer(init)

	outputSize := LabelLength * (opt.endPosition - opt.startPosition + 1)
	cStartPosition := C.uint64_t(opt.startPosition)
	cEndPosition := C.uint64_t(opt.endPosition)
	cOutputSize := C.size_t(outputSize)
	cOut := (C.calloc(cOutputSize, 1))
	defer C.free(cOut)

	var cIdxSolution C.uint64_t
	retVal := C.initialize(init, cStartPosition, cEndPosition, (*C.uint8_t)(cOut), &cIdxSolution)
	if err := InitResultToError(retVal); err != nil {
		return nil, nil, err
	}

	var vrfNonce *uint64
	if cIdxSolution != math.MaxUint64 { // TODO(mafa): we should find a better way to indicate no solution (e.g. InitializeOk = no solution, InitializeOkPow = solution)
		vrfNonce = new(uint64)
		*vrfNonce = uint64(cIdxSolution)
	}

	output := C.GoBytes(cOut, C.int(cOutputSize))
	return output, vrfNonce, nil
}

func cCPUProviderID() uint {
	return C.CPU_PROVIDER_ID
}

func cGetProviders() ([]Provider, error) {
	// TODO(mafa): disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace)

	cNumProviders := C.get_providers_count()
	if cNumProviders == 0 {
		return nil, ErrFetchProviders
	}

	cProviders := make([]C.Provider, cNumProviders)
	providers := make([]Provider, cNumProviders)
	retVal := C.get_providers(&cProviders[0], cNumProviders)
	if err := InitResultToError(retVal); err != nil {
		return nil, err
	}

	for i := uint(0); i < uint(cNumProviders); i++ {
		providers[i].ID = (uint)(cProviders[i].id)
		// TODO(mafa): `name` could be char instead of `uint8_t` then this cast isn't needed to work around staticcheck
		providers[i].Model = C.GoString((*C.char)(unsafe.Pointer((&cProviders[i].name[0]))))
		providers[i].DeviceType = DeviceClass(cProviders[i].class_)
	}

	return providers, nil
}
