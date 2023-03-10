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

type proof struct {
	Nonce    uint32
	Indicies []uint64
	K2Pow    uint64
	K3Pow    uint64
}

func GenerateProof(datadir string, challenge []byte, cfg config.Config) (*proof, error) {
	datadirPtr := C.CString(datadir)
	defer C.free(unsafe.Pointer(datadirPtr))

	challengePtr := C.CBytes(challenge)
	defer C.free(challengePtr)

	config := C.Config{
		n:                 C.uint32_t(cfg.N),
		b:                 C.uint32_t(cfg.B),
		k1:                C.uint32_t(cfg.K1),
		k2:                C.uint32_t(cfg.K2),
		k2_pow_difficulty: C.uint64_t(cfg.K2PowDifficulty),
		k3_pow_difficulty: C.uint64_t(cfg.K3PowDifficulty),
		labels_per_unit:   C.uint64_t(cfg.LabelsPerUnit),
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

	indicies := make([]uint64, cProof.indicies.len)
	copy(indicies, unsafe.Slice((*uint64)(unsafe.Pointer(cProof.indicies.ptr)), cProof.indicies.len))

	return &proof{
		Nonce:    uint32(cProof.nonce),
		Indicies: indicies,
		K2Pow:    uint64(cProof.k2_pow),
		K3Pow:    uint64(cProof.k3_pow),
	}, nil
}
