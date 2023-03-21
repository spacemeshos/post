package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "prover.h"
import "C"

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/shared"
)

// Translate scrypt parameters expressed as N,R,P to Nfactor, Rfactor and Pfactor
// that are understood by scrypt-jane.
// Relation:
// N = 1 << (nfactor + 1)
// r = 1 << rfactor
// p = 1 << pfactor
func translateScryptParams(params config.ScryptParams) C.ScryptParams {
	return C.ScryptParams{
		nfactor: C.uint8_t(math.Log2(float64(params.N))) - 1,
		rfactor: C.uint8_t(math.Log2(float64(params.R))),
		pfactor: C.uint8_t(math.Log2(float64(params.P))),
	}
}

func GenerateProof(datadir string, challenge []byte, cfg config.Config, nonces uint, powScrypt config.ScryptParams) (*shared.Proof, error) {
	datadirPtr := C.CString(datadir)
	defer C.free(unsafe.Pointer(datadirPtr))

	challengePtr := C.CBytes(challenge)
	defer C.free(challengePtr)

	config := C.Config{
		k1:                C.uint32_t(cfg.K1),
		k2:                C.uint32_t(cfg.K2),
		k2_pow_difficulty: C.uint64_t(cfg.K2PowDifficulty),
		k3_pow_difficulty: C.uint64_t(cfg.K3PowDifficulty),
		pow_scrypt:        translateScryptParams(powScrypt),
	}

	cProof := C.generate_proof(
		datadirPtr,
		(*C.uchar)(challengePtr),
		C.size_t(len(challenge)),
		config,
		C.size_t(nonces),
	)

	if cProof == nil {
		return nil, fmt.Errorf("got nil")
	}
	defer C.free_proof(cProof)

	indices := make([]uint8, cProof.indices.len)
	copy(indices, unsafe.Slice((*uint8)(unsafe.Pointer(cProof.indices.ptr)), cProof.indices.len))

	return &shared.Proof{
		Nonce:   uint32(cProof.nonce),
		Indices: indices,
		K2Pow:   uint64(cProof.k2_pow),
		K3Pow:   uint64(cProof.k3_pow),
	}, nil
}

func VerifyProof(proof *shared.Proof, metadata *shared.ProofMetadata, cfg config.Config, theads uint, powScrypt, labelScrypt config.ScryptParams) error {
	config := C.Config{
		k1:                C.uint32_t(cfg.K1),
		k2:                C.uint32_t(cfg.K2),
		k3:                C.uint32_t(cfg.K3),
		k2_pow_difficulty: C.uint64_t(cfg.K2PowDifficulty),
		k3_pow_difficulty: C.uint64_t(cfg.K3PowDifficulty),
		pow_scrypt:        translateScryptParams(powScrypt),
		scrypt:            translateScryptParams(labelScrypt),
	}

	cProof := C.Proof{
		nonce:  C.uint32_t(proof.Nonce),
		k2_pow: C.uint64_t(proof.K2Pow),
		k3_pow: C.uint64_t(proof.K3Pow),
		indices: C.ArrayU8{
			ptr: (*C.uchar)(unsafe.Pointer(&proof.Indices[0])),
			len: C.size_t(len(proof.Indices)),
			cap: C.size_t(cap(proof.Indices)),
		},
	}

	cMetadata := C.ProofMetadata{
		node_id:           *(*[32]C.uchar)(unsafe.Pointer(&metadata.NodeId[0])),
		commitment_atx_id: *(*[32]C.uchar)(unsafe.Pointer(&metadata.CommitmentAtxId[0])),
		challenge:         *(*[32]C.uchar)(unsafe.Pointer(&metadata.Challenge[0])),
		num_units:         C.uint32_t(metadata.NumUnits),
		labels_per_unit:   C.uint64_t(metadata.LabelsPerUnit),
	}
	result := C.verify_proof(
		cProof,
		&cMetadata,
		config,
		1,
	)

	switch result {
	case C.Ok:
		return nil
	case C.Invalid:
		return fmt.Errorf("invalid proof")
	case C.InvalidArgument:
		return fmt.Errorf("invalid argument")
	default:
		return fmt.Errorf("unknown error")
	}
}
