package gpu

// #cgo LDFLAGS: -Wl,-rpath,${SRCDIR} -L${SRCDIR} -lgpu-setup
// #include "./api.h"
// #include <stdlib.h>
//
import "C"
import (
	"unsafe"
)

type (
	cChar  = C.char
	cUchar = C.uchar
)

const (
	ComputeAPIClassUnspecified = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_UNSPECIFIED))
	ComputeAPIClassCPU         = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_CPU))
	ComputeAPIClassCuda        = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_CUDA))
	ComputeAPIClassVulkan      = ComputeAPIClass((C.ComputeApiClass)(C.COMPUTE_API_CLASS_VULKAN))

	StopResultOk           = StopResult(C.SPACEMESH_API_ERROR_NONE)
	StopResultError        = StopResult(C.SPACEMESH_API_ERROR)
	StopResultErrorTimeout = StopResult(C.SPACEMESH_API_ERROR_TIMEOUT)
	StopResultErrorAlready = StopResult(C.SPACEMESH_API_ERROR_TIMEOUT)
)

type StopResult int

type ComputeProvider struct {
	Id         uint
	Model      string
	ComputeAPI ComputeAPIClass
}

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

func cScryptPositions(providerId uint, id, salt []byte, startPosition, endPosition uint64, hashLenBits uint8, options uint32, outputSize uint64, n, r, p uint32) ([]byte, int) {
	cProviderId := C.uint(providerId)
	cId := (*C.uchar)(GoBytes(id).CBytesClone().data)
	cStartPosition := C.uint64_t(startPosition)
	cEndPosition := C.uint64_t(endPosition)
	cHashLenBits := C.uchar(hashLenBits)
	cSalt := (*C.uchar)(GoBytes(salt).CBytesClone().data)
	cOptions := C.uint(options)
	cOutputSize := C.size_t(outputSize)
	cOut := (*C.uchar)(C.malloc(cOutputSize))
	cN := C.uint(n)
	cR := C.uint(r)
	cP := C.uint(p)

	defer func() {
		cFree(unsafe.Pointer(cId))
		cFree(unsafe.Pointer(cSalt))
		cFree(unsafe.Pointer(cOut))
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
	)
	return cBytesCloneToGoBytes(cOut, int(outputSize)), int(retVal)
}

func cGetProviders() []ComputeProvider {
	numProviders := C.spacemesh_api_get_providers(nil, 0)
	cProviders := make([]C.PostComputeProvider, numProviders)
	providers := make([]ComputeProvider, numProviders)

	_ = C.spacemesh_api_get_providers(&cProviders[0], numProviders)

	for i := 0; i < int(numProviders); i++ {
		providers[i].Id = uint(cProviders[i].id)
		providers[i].Model = cStringArrayToGoString(cProviders[i].model)
		providers[i].ComputeAPI = ComputeAPIClass(cProviders[i].compute_api)
	}

	return providers
}

func cStop(msTimeout uint) StopResult {
	cMsTimeout := C.uint(msTimeout)
	return StopResult(C.stop(cMsTimeout))
}

func cFree(p unsafe.Pointer) {
	C.free(p)
}
