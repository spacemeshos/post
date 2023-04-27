package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "prover.h"
import "C"

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
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

var ErrFetchProviders = errors.New("failed to fetch providers")

func cGetProviders() ([]ComputeProvider, error) {
	cNumProviders := C.get_providers_count()
	if cNumProviders == 0 {
		return nil, nil
	}

	cProviders := make([]C.Provider, cNumProviders)
	providers := make([]ComputeProvider, cNumProviders)
	retVal := C.get_providers(&cProviders[0], cNumProviders)
	if retVal != C.InitializeOk {
		return nil, ErrFetchProviders
	}

	for i := uint(0); i < uint(cNumProviders); i++ {
		// TODO(mafa): imo the id should come from the postrs library and not be assigned be me (for consistency)
		providers[i].ID = i
		providers[i].Model = C.GoString((*C.char)(&cProviders[i].name[0]))

		// TODO(mafa): the gpu-post code had an additional field in the provider struct
		// that indicated what class of device it was.

		// I only need it to know which provider is the CPU provider, so this could
		// also be a boolean `isCPU`, a separate function like `get_cpu_provider_id()` or
		// any other way that makes sense.
		providers[i].ComputeAPI = ComputeAPIClassUnspecified
	}

	return providers, nil
}

func Initialize() error {
	commitment := make([]byte, 32)
	cCommitment := C.CBytes(commitment)
	defer C.free(cCommitment)

	difficulty := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).Bytes()
	cDifficulty := C.CBytes(difficulty)
	defer C.free(cDifficulty)

	init := C.new_initializer(0, 8192, (*C.uchar)(cCommitment), (*C.uchar)(cDifficulty))
	defer C.free_initializer(init)

	cOutputSize := C.size_t(16 * 2)
	cOut := (C.calloc(cOutputSize, 1))
	defer C.free(cOut)

	var cIdxSolution C.uint64_t

	// TODO(mafa): does this calculate 1 or 2 labels? - in gpu-post it's 2, here it appears to be 1
	retVal := C.initialize(init, 1, 2, (*C.uchar)(cOut), &cIdxSolution)
	if retVal != C.InitializeOk {
		return fmt.Errorf("failed to initialize: %d", retVal)
	}

	output := C.GoBytes(cOut, C.int(cOutputSize))

	// TODO(mafa): in gpu-post calculating 16 byte labels 1 and 2 with commitment 0x0 gives
	// 0x82032392c5605bfbfed09343fa06f086073b1e043c5746ee6e48af178ebeac91
	// here it is
	// 0x82032392c5605bfbfed09343fa06f08600000000000000000000000000000000
	// (only one label?)
	fmt.Printf("output: %x\n", output)
	return nil
}

func GenerateProof(dataDir string, challenge []byte, cfg config.Config, nonces uint, threads uint, powScrypt config.ScryptParams) (*shared.Proof, error) {
	dataDirPtr := C.CString(dataDir)
	defer C.free(unsafe.Pointer(dataDirPtr))

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
		dataDirPtr,
		(*C.uchar)(challengePtr),
		config,
		C.size_t(nonces),
		C.size_t(threads),
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

func VerifyProof(proof *shared.Proof, metadata *shared.ProofMetadata, cfg config.Config, powScrypt, labelScrypt config.ScryptParams) error {
	if proof == nil {
		return errors.New("proof cannot be nil")
	}
	if metadata == nil {
		return errors.New("metadata cannot be nil")
	}
	if len(metadata.NodeId) != 32 {
		return errors.New("node id length must be 32")
	}
	if len(metadata.CommitmentAtxId) != 32 {
		return errors.New("commitment atx id length must be 32")
	}
	if len(metadata.Challenge) != 32 {
		return errors.New("challenge length must be 32")
	}
	if len(proof.Indices) == 0 {
		return errors.New("proof indices are empty")
	}

	config := C.Config{
		k1:                C.uint32_t(cfg.K1),
		k2:                C.uint32_t(cfg.K2),
		k3:                C.uint32_t(cfg.K3),
		k2_pow_difficulty: C.uint64_t(cfg.K2PowDifficulty),
		k3_pow_difficulty: C.uint64_t(cfg.K3PowDifficulty),
		pow_scrypt:        translateScryptParams(powScrypt),
		scrypt:            translateScryptParams(labelScrypt),
	}

	indicesSliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&proof.Indices))
	cProof := C.Proof{
		nonce:  C.uint32_t(proof.Nonce),
		k2_pow: C.uint64_t(proof.K2Pow),
		k3_pow: C.uint64_t(proof.K3Pow),
		indices: C.ArrayU8{
			ptr: (*C.uchar)(unsafe.Pointer(indicesSliceHdr.Data)),
			len: C.size_t(indicesSliceHdr.Len),
			cap: C.size_t(indicesSliceHdr.Cap),
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
