package gpu

// #include "./api.h"
// #include <stdlib.h>
//
import "C"
import (
	"github.com/spacemeshos/post/shared"
	"sync"
	"unsafe"
)

// mtx is a mutual exclusion lock for serializing calls to gpu-post lib.
// If not applied, concurrent calls are expected to cause a crash.
var mtx sync.Mutex

type (
	cChar  = C.char
	cUchar = C.uchar
)

const (
	ComputeAPIClassUnspecified = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_UNSPECIFIED))
	ComputeAPIClassCPU         = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_CPU))
	ComputeAPIClassCuda        = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_CUDA))
	ComputeAPIClassVulkan      = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_VULKAN))

	StopResultOk             = StopResult(C.SPACEMESH_API_ERROR_NONE)
	StopResultError          = StopResult(C.SPACEMESH_API_ERROR)
	StopResultErrorTimeout   = StopResult(C.SPACEMESH_API_ERROR_TIMEOUT)
	StopResultErrorAlready   = StopResult(C.SPACEMESH_API_ERROR_ALREADY)
	StopResultErrorCancelled = StopResult(C.SPACEMESH_API_ERROR_CANCELED)
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
		return "cancelled"
	default:
		panic("unreachable")
	}
}

func cScryptPositions(providerId uint, id, salt []byte, startPosition, endPosition uint64, labelSize uint32, options uint32, n, r, p uint32) ([]byte, uint64, int, int) {
	mtx.Lock()
	defer mtx.Unlock()

	outputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(labelSize))

	cProviderId := C.uint(providerId)
	cId := (*C.uchar)(GoBytes(id).CBytesClone().data)
	cStartPosition := C.uint64_t(startPosition)
	cEndPosition := C.uint64_t(endPosition)
	cHashLenBits := C.uint32_t(labelSize)
	cSalt := (*C.uchar)(GoBytes(salt).CBytesClone().data)
	cOptions := C.uint(options)
	cOutputSize := C.size_t(outputSize)
	cOut := (*C.uchar)(C.calloc(cOutputSize, 1))
	cN := C.uint(n)
	cR := C.uint(r)
	cP := C.uint(p)
	cD := (*C.uchar)(C.calloc(32, 1))

	var cIdxSolution C.uint64_t
	var cHashesComputed C.uint64_t
	var cHashesPerSec C.uint64_t

	defer func() {
		cFree(unsafe.Pointer(cId))
		cFree(unsafe.Pointer(cSalt))
		cFree(unsafe.Pointer(cOut))
		cFree(unsafe.Pointer(cD))

	}()

	retVal := C.scryptPositions(
		cProviderId,
		cId,
		cStartPosition,
		cEndPosition,
		cHashLenBits,
		cSalt,
		cOptions,
		cOut,
		cN,
		cR,
		cP,
		cD,
		&cIdxSolution,
		&cHashesComputed,
		&cHashesPerSec,
	)

	// Output size could be smaller than anticipated if `C.stop` was called while `C.scryptPositions` was blocking.
	outputSize = shared.DataSize(uint64(cHashesComputed), uint(labelSize))
	output := cBytesCloneToGoBytes(cOut, int(outputSize))

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
		providers[i].Model = cStringArrayToGoString(cProviders[i].model)
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

func cFree(p unsafe.Pointer) {
	C.free(p)
}
