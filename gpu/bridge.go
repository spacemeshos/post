package gpu

// #cgo LDFLAGS: -Wl,-rpath,${SRCDIR} -L${SRCDIR} -lgpu-setup
// #include "./api.h"
// #include <stdlib.h>
//
import "C"
import (
	"unsafe"
)

type cUchar = C.uchar
type cUint = C.uint

type APIType uint32

const (
	CPU       APIType = C.SPACEMESH_API_CPU
	GPUCuda           = C.SPACEMESH_API_CUDA
	GPUOpenCL         = C.SPACEMESH_API_OPENCL
	GPUVulkan         = C.SPACEMESH_API_VULKAN
	GPUAll            = C.SPACEMESH_API_GPU
	ALL               = C.SPACEMESH_API_ALL
)

func cScryptPositions(id, salt []byte, startPosition, endPosition uint64, options uint32, hashLenBits uint8, outputSize uint64, n, r, p uint32) ([]byte, int) {
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

func cStats() int {
	return int(C.stats())
}

func cGPUCount(apiType APIType, onlyAvailable bool) int {
	cAPIType := C.int(apiType)
	var cOnlyAvailable C.int
	if onlyAvailable {
		cOnlyAvailable = 1
	}

	return int(C.spacemesh_api_get_gpu_count(cAPIType, cOnlyAvailable))
}

func cFree(p unsafe.Pointer) {
	C.free(p)
}
