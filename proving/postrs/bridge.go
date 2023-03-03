package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "prover.h"
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/spacemeshos/post/config"
)

type Proof struct {
	Nonce    uint32
	Indicies []uint64
}

func GenerateProof(datadir string, challenge []byte, cfg config.Config) (*Proof, error) {
	datadirPtr := C.CString(datadir)
	defer C.free(unsafe.Pointer(datadirPtr))

	challengePtr := C.CBytes(challenge)
	defer C.free(challengePtr)

	config := C.Config{
		n:               20,
		b:               C.uint32_t(cfg.B),
		k1:              C.uint32_t(cfg.K1),
		k2:              C.uint32_t(cfg.K2),
		labels_per_unit: C.uint64_t(cfg.LabelsPerUnit),
	}

	cProof := C.generate_proof(
		datadirPtr,
		(*C.uchar)(challengePtr),
		C.size_t(len(challenge)),
		config,
	)

	if cProof == nil {
		return nil, fmt.Errorf("got nil")
	}
	defer C.free_proof(cProof)

	indicies := unsafe.Slice((*uint64)(unsafe.Pointer(cProof.indicies.ptr)), cProof.indicies.len)
	indicies = append([]uint64{}, indicies...)

	return &Proof{
		Nonce:    uint32(cProof.nonce),
		Indicies: indicies,
	}, nil
}
