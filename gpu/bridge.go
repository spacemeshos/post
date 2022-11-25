package gpu

// #cgo LDFLAGS: -lgpu-setup
//
// #include <api.h>
// #include <stdlib.h>
import "C"

import (
	"sync"

	"github.com/spacemeshos/post/shared"
)

// mtx is a mutual exclusion lock for serializing calls to gpu-post lib.
// If not applied, concurrent calls are expected to cause a crash.
var mtx sync.Mutex

const (
	ComputeAPIClassUnspecified = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_UNSPECIFIED))
	ComputeAPIClassCPU         = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_CPU))
	ComputeAPIClassCuda        = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_CUDA))
	ComputeAPIClassVulkan      = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_VULKAN))

	StopResultPowFound              = StopResult(C.SPACEMESH_API_POW_SOLUTION_FOUND)
	StopResultOk                    = StopResult(C.SPACEMESH_API_ERROR_NONE)
	StopResultError                 = StopResult(C.SPACEMESH_API_ERROR)
	StopResultErrorTimeout          = StopResult(C.SPACEMESH_API_ERROR_TIMEOUT)
	StopResultErrorAlready          = StopResult(C.SPACEMESH_API_ERROR_ALREADY)
	StopResultErrorCancelled        = StopResult(C.SPACEMESH_API_ERROR_CANCELED)
	StopResultErrorNoCompoteOptions = StopResult(C.SPACEMESH_API_ERROR_NO_COMPOTE_OPTIONS)
	StopResultErrorInvalidParameter = StopResult(C.SPACEMESH_API_ERROR_INVALID_PARAMETER)
	StopResultErrorInvalidProvider  = StopResult(C.SPACEMESH_API_ERROR_INVALID_PROVIDER_ID)
)

type ComputeAPIClass uint

func (c ComputeAPIClass) String() string {
	switch c {
	case ComputeAPIClassUnspecified:
		return "Unspecified"
	case ComputeAPIClassCPU:
		return "CPU"
	case ComputeAPIClassCuda:
		return "Cuda"
	case ComputeAPIClassVulkan:
		return "Vulkan"
	default:
		return "N/A"
	}
}

type StopResult int

func (s StopResult) String() string {
	switch s {
	case StopResultOk:
		return "ok"
	case StopResultError:
		return "general error"
	case StopResultErrorTimeout:
		return "timeout"
	case StopResultErrorAlready:
		return "already stopped"
	case StopResultErrorCancelled:
		return "canceled"
	default:
		panic("unreachable")
	}
}

func cScryptPositions(opt *option) ([]byte, uint64, int, int) {
	mtx.Lock()
	defer mtx.Unlock()

	outputSize := shared.DataSize(uint64(opt.endPosition-opt.startPosition+1), uint(opt.bitsPerLabel))
	cProviderId := C.uint(opt.computeProviderID)

	cCommitment := C.CBytes(opt.commitment)
	defer C.free(cCommitment)

	cStartPosition := C.uint64_t(opt.startPosition)
	cEndPosition := C.uint64_t(opt.endPosition)
	cHashLenBits := C.uint32_t(opt.bitsPerLabel)

	cSalt := C.CBytes(opt.salt)
	defer C.free(cSalt)

	cOptions := C.uint(opt.optionBits())
	cOutputSize := C.size_t(outputSize)

	cOut := (C.calloc(cOutputSize, 1))
	defer C.free(cOut)

	cN := C.uint(opt.n)
	cR := C.uint(opt.r)
	cP := C.uint(opt.p)

	cD := C.CBytes(opt.d)
	defer C.free(cD)

	var cIdxSolution C.uint64_t
	var cHashesComputed C.uint64_t
	var cHashesPerSec C.uint64_t

	retVal := C.scryptPositions(
		cProviderId,
		(*C.uchar)(cCommitment),
		cStartPosition,
		cEndPosition,
		cHashLenBits,
		(*C.uchar)(cSalt),
		cOptions,
		(*C.uchar)(cOut),
		cN,
		cR,
		cP,
		(*C.uchar)(cD),
		&cIdxSolution,
		&cHashesComputed,
		&cHashesPerSec,
	)

	// Output size could be smaller than anticipated if `C.stop` was called while `C.scryptPositions` was blocking.
	outputSize = shared.DataSize(uint64(cHashesComputed), uint(opt.bitsPerLabel))
	output := C.GoBytes(cOut, C.int(outputSize))

	return output, uint64(cIdxSolution), int(cHashesPerSec), int(retVal)
}

func cGetProviders() []ComputeProvider {
	mtx.Lock()
	defer mtx.Unlock()

	numProviders := C.spacemesh_api_get_providers(nil, 0)
	cProviders := make([]C.PostComputeProvider, numProviders)
	providers := make([]ComputeProvider, numProviders)

	_ = C.spacemesh_api_get_providers(&cProviders[0], numProviders)

	for i := 0; i < int(numProviders); i++ {
		providers[i].ID = uint(cProviders[i].id)
		providers[i].Model = C.GoString(&cProviders[i].model[0])
		providers[i].ComputeAPI = ComputeAPIClass(cProviders[i].compute_api)
	}

	return providers
}

func cStopCleared() bool {
	return C.spacemesh_api_stop_inprogress() == 0
}

func cStop(msTimeout uint) StopResult {
	cMsTimeout := C.uint(msTimeout)
	return StopResult(C.stop(cMsTimeout))
}
