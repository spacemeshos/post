package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "post.h"
import "C"

import (
	"errors"
)

// gpuMtx is an instance of deviceMutex that can be used to prevent concurrent calls
// to the same GPU (by ProviderID) from multiple goroutines.
var gpuMtx deviceMutex

// DeviceClass is an enum for the type of device (CPU or GPU).
type DeviceClass int

const (
	ClassUnspecified = 0
	ClassCPU         = DeviceClass((C.DeviceClass)(C.CPU))
	ClassGPU         = DeviceClass((C.DeviceClass)(C.GPU))
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
		return "Unspecified"
	}
}

var (
	ErrInvalidProviderID    = errors.New("invalid provider ID")
	ErrInvalidLabelsRange   = errors.New("invalid labels range")
	ErrInitializationFailed = errors.New("initialization failed")
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrFetchProviders       = errors.New("failed to fetch providers")
	ErrUnknown              = errors.New("unknown error")
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
	case C.InitializeOkNonceNotFound:
		return nil
	case C.InitializeInvalidLabelsRange:
		return ErrInvalidLabelsRange
	case C.InitializeError:
		return ErrInitializationFailed
	case C.InitializeInvalidArgument:
		return ErrInvalidArgument
	case C.InitializeFailedToGetProviders:
		return ErrFetchProviders
	default:
		return ErrUnknown
	}
}

// cNewInitializer calls the C function from libpostrs that creates the
// initializer.
func cNewInitializer(opt *option) (*C.Initializer, error) {
	if opt.logger != nil {
		SetLogCallback(opt.logger)
	}

	cProviderId := C.uint32_t(*opt.providerID)
	cN := C.uintptr_t(opt.n)
	cCommitment := C.CBytes(opt.commitment)
	defer C.free(cCommitment)
	cDifficulty := C.CBytes(opt.vrfDifficulty)
	defer C.free(cDifficulty)
	init := C.new_initializer(cProviderId, cN, (*C.uchar)(cCommitment), (*C.uchar)(cDifficulty))
	if init == nil {
		return nil, ErrInvalidProviderID
	}
	return init, nil
}

// cFreeInitializer calls the C function from libpostrs that frees the memory
// allocated for the initializer.
func cFreeInitializer(init *C.Initializer) {
	C.free_initializer(init)
}

// cScryptPositions calls the C functions from libpostrs that create the labels
// and VRF proofs.
func cScryptPositions(init *C.Initializer, opt *option, start, end uint64) ([]byte, *uint64, error) {
	outputSize := LabelLength * (end - start + 1)
	cStartPosition := C.uint64_t(start)
	cEndPosition := C.uint64_t(end)
	cOutputSize := C.size_t(outputSize)
	cOut := (C.malloc(cOutputSize))
	defer C.free(cOut)

	var cIdxSolution C.uint64_t
	retVal := C.initialize(init, cStartPosition, cEndPosition, (*C.uint8_t)(cOut), &cIdxSolution)
	if err := InitResultToError(retVal); err != nil {
		return nil, nil, err
	}

	output := C.GoBytes(cOut, C.int(cOutputSize))

	if retVal == C.InitializeOkNonceNotFound {
		return output, nil, nil
	}

	vrfNonce := new(uint64)
	*vrfNonce = uint64(cIdxSolution)
	return output, vrfNonce, nil
}

// cCPUProviderID returns the ID for the (non OpenCL) CPU provider.
func cCPUProviderID() uint {
	return C.CPU_PROVIDER_ID
}

// cGetProviders calls the C function from libpostrs that fetches the list of
// OpenCL providers.
func cGetProviders() ([]Provider, error) {
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
		providers[i].Model = C.GoString(&cProviders[i].name[0])
		providers[i].DeviceType = DeviceClass(cProviders[i].class_)
	}

	return providers, nil
}
