package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "prover.h"
import "C"

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"unsafe"
)

// gpuMtx is a mutual exclusion lock for calls to gpu functions. It is required
// to prevent concurrent calls to the same GPU from multiple goroutines.
var gpuMtx deviceMutex

type deviceMutex struct {
	mtx    sync.Mutex
	device map[uint]*sync.Mutex
}

func (g *deviceMutex) Device(deviceId uint) *sync.Mutex {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if g.device == nil {
		g.device = make(map[uint]*sync.Mutex)
	}

	if _, ok := g.device[deviceId]; !ok {
		g.device[deviceId] = new(sync.Mutex)
	}

	return g.device[deviceId]
}

type DeviceClass int

const (
	ClassCPU = DeviceClass((C.DeviceClass)(C.CPU))
	ClassGPU = DeviceClass((C.DeviceClass)(C.GPU))
)

type ComputeProvider struct {
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
	ErrOclError           = errors.New("OpenCL error")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrFetchProviders     = errors.New("failed to fetch providers")
)

const (
	// LabelLength is the length of the label in bytes.
	LabelLength = 16
)

func InitResultToError(retVal uint32) error {
	switch retVal {
	case C.InitializeOk:
		return nil
	case C.InitializeInvalidLabelsRange:
		return ErrInvalidLabelsRange
	case C.InitializeOclError:
		return ErrOclError
	case C.InitializeInvalidArgument:
		return ErrInvalidArgument
	case C.InitializeFailedToGetProviders:
		return ErrFetchProviders
	default:
		return fmt.Errorf("unknown error")
	}
}

func cScryptPositions(opt *option) ([]byte, *uint64, error) {
	// disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace) // TODO(mafa): make this configurable

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

func cGetProviders() ([]ComputeProvider, error) {
	// disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace) // TODO(mafa): make this configurable

	cNumProviders := C.get_providers_count()
	if cNumProviders == 0 {
		return nil, ErrFetchProviders
	}

	cProviders := make([]C.Provider, cNumProviders)
	providers := make([]ComputeProvider, cNumProviders)
	retVal := C.get_providers(&cProviders[0], cNumProviders)
	if err := InitResultToError(retVal); err != nil {
		return nil, err
	}

	for i := uint(0); i < uint(cNumProviders); i++ {
		providers[i].ID = (uint)(cProviders[i].id)
		// TODO(mafa): `name` should be char instead of `uint8_t` then this cast isn't needed to work around staticcheck
		providers[i].Model = C.GoString((*C.char)(unsafe.Pointer((&cProviders[i].name[0]))))
		providers[i].DeviceType = DeviceClass(cProviders[i].class_)
	}

	return providers, nil
}
