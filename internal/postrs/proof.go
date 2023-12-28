package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "post.h"
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"go.uber.org/zap"

	"github.com/spacemeshos/post/shared"
)

type ScryptParams = C.ScryptParams

type Config struct {
	MinNumUnits   uint32
	MaxNumUnits   uint32
	LabelsPerUnit uint64

	K1 uint32 // K1 specifies the difficulty for a label to be a candidate for a proof.
	K2 uint32 // K2 is the number of labels below the required difficulty required for a proof.

	PowDifficulty [32]byte
}

func NewScryptParams(n, r, p uint) ScryptParams {
	return ScryptParams{
		n: C.size_t(n),
		r: C.size_t(r),
		p: C.size_t(p),
	}
}

// ErrVerifierClosed is returned when calling a method on an already closed Scrypt instance.
var ErrVerifierClosed = errors.New("verifier has been closed")

func GenerateProof(dataDir string, challenge []byte, logger *zap.Logger, nonces, threads uint, K1, K2 uint32, powDifficulty [32]byte, powFlags PowFlags) (*shared.Proof, error) {
	if logger != nil {
		setLogCallback(logger)
	}

	dataDirPtr := C.CString(dataDir)
	defer C.free(unsafe.Pointer(dataDirPtr))

	challengePtr := C.CBytes(challenge)
	defer C.free(challengePtr)

	config := C.ProofConfig{
		k1: C.uint32_t(K1),
		k2: C.uint32_t(K2),
	}
	for i, b := range powDifficulty {
		config.pow_difficulty[i] = C.uchar(b)
	}

	cProof := C.generate_proof(
		dataDirPtr,
		(*C.uchar)(challengePtr),
		config,
		C.size_t(nonces),
		C.size_t(threads),
		powFlags,
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
		Pow:     uint64(cProof.pow),
	}, nil
}

type PowFlags = C.RandomXFlag

// Get the recommended PoW flags.
//
// Does not include:
// * FLAG_LARGE_PAGES
// * FLAG_FULL_MEM
// * FLAG_SECURE
//
// The above flags need to be set manually, if required.
func GetRecommendedPowFlags() PowFlags {
	return C.recommended_pow_flags()
}

const (
	// Use the full dataset. AKA "Fast mode".
	PowFastMode = C.RandomXFlag_FLAG_FULL_MEM
	// Allocate memory in large pages.
	PowLargePages = C.RandomXFlag_FLAG_LARGE_PAGES
	// Use JIT compilation support.
	PowJIT = C.RandomXFlag_FLAG_JIT
	// When combined with FLAG_JIT, the JIT pages are never writable and executable at the same time.
	PowSecure = C.RandomXFlag_FLAG_SECURE
	// Use hardware accelerated AES.
	PowHardAES = C.RandomXFlag_FLAG_HARD_AES
	// Optimize Argon2 for CPUs with the SSSE3 instruction set.
	PowArgon2SSSE3 = C.RandomXFlag_FLAG_ARGON2_SSSE3
	// Optimize Argon2 for CPUs with the SSSE3 instruction set.
	PowArgon2AVX2 = C.RandomXFlag_FLAG_ARGON2_AVX2
	// Optimize Argon2 for CPUs without the AVX2 or SSSE3 instruction sets.
	PowArgon2 = C.RandomXFlag_FLAG_ARGON2
)

type Verifier struct {
	mu    sync.RWMutex
	inner *C.Verifier
	id    []byte
}

// Create a new verifier.
// The verifier must be closed after use with Close().
func NewVerifier(id []byte, powFlags PowFlags) (*Verifier, error) {
	verifier := Verifier{
		id: id,
	}
	result := C.new_verifier(powFlags, &verifier.inner)
	if result != C.VerifyResult_Ok {
		return nil, fmt.Errorf("failed to create verifier")
	}
	return &verifier, nil
}

func (v *Verifier) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.inner == nil {
		return nil
	}

	C.free_verifier(v.inner)
	v.inner = nil
	return nil
}

type verifyOptions struct {
	mode C.Mode
}

type VerifyOptionFunc func(*verifyOptions)

// Verify all indices in the proof.
func VerifyAll() VerifyOptionFunc {
	return func(o *verifyOptions) {
		o.mode.tag = C.Mode_All
	}
}

// Verify only the selected index.
// The `ord` is the ordinal number of the index in the proof to verify.
func VerifyOne(ord int) VerifyOptionFunc {
	return func(o *verifyOptions) {
		// `o.mode` is a C tagged union
		o.mode.tag = C.Mode_One
		o.mode.anon0 = (*(*[8]byte)(unsafe.Pointer(&C.Mode_One_Body{
			index: C.size_t(ord),
		})))
	}
}

// Verify a subset of randomly selected K3 indices.
func VerifySubset(k3 int) VerifyOptionFunc {
	return func(o *verifyOptions) {
		// `o.mode` is a C tagged union
		o.mode.tag = C.Mode_Subset
		o.mode.anon0 = (*(*[8]byte)(unsafe.Pointer(&C.Mode_Subset_Body{
			k3: C.size_t(k3),
		})))
	}
}

func (v *Verifier) VerifyProof(proof *shared.Proof, metadata *shared.ProofMetadata, logger *zap.Logger, cfg Config, scryptParams ScryptParams, opts ...VerifyOptionFunc) error {
	if logger != nil {
		setLogCallback(logger)
	}

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

	config := C.ProofConfig{
		k1: C.uint32_t(cfg.K1),
		k2: C.uint32_t(cfg.K2),
	}
	for i, b := range cfg.PowDifficulty {
		config.pow_difficulty[i] = C.uchar(b)
	}
	initConfig := C.InitConfig{
		labels_per_unit: C.uint64_t(cfg.LabelsPerUnit),
		min_num_units:   C.uint32_t(cfg.MinNumUnits),
		max_num_units:   C.uint32_t(cfg.MaxNumUnits),
		scrypt:          scryptParams,
	}

	cProof := C.Proof{
		nonce: C.uint32_t(proof.Nonce),
		pow:   C.uint64_t(proof.Pow),
		indices: C.ArrayU8{
			ptr: (*C.uchar)(unsafe.SliceData(proof.Indices)),
			len: C.size_t(len(proof.Indices)),
			cap: C.size_t(cap(proof.Indices)),
		},
	}

	cMetadata := C.ProofMetadata{
		node_id:           *(*[32]C.uchar)(unsafe.Pointer(&metadata.NodeId[0])),
		commitment_atx_id: *(*[32]C.uchar)(unsafe.Pointer(&metadata.CommitmentAtxId[0])),
		challenge:         *(*[32]C.uchar)(unsafe.Pointer(&metadata.Challenge[0])),
		num_units:         C.uint32_t(metadata.NumUnits),
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.inner == nil {
		return ErrVerifierClosed
	}

	id := C.ArrayU8{
		ptr: (*C.uchar)(unsafe.SliceData(v.id)),
		len: C.size_t(len(v.id)),
		cap: C.size_t(cap(v.id)),
	}

	options := verifyOptions{}
	VerifyAll()(&options)
	for _, opt := range opts {
		opt(&options)
	}

	result := C.verify_proof(
		v.inner,
		cProof,
		&cMetadata,
		config,
		initConfig,
		id,
		options.mode,
	)

	switch result.tag {
	case C.VerifyResult_Ok:
		return nil
	case C.VerifyResult_InvalidIndex:
		result := castBytes[C.VerifyResult_InvalidIndex_Body](result.anon0[:])
		return &ErrInvalidIndex{Index: int(result.index_id)}
	case C.VerifyResult_InvalidArgument:
		return fmt.Errorf("invalid argument")
	default:
		return fmt.Errorf("unknown error")
	}
}

type ErrInvalidIndex struct {
	Index int
}

func (e ErrInvalidIndex) Error() string {
	return fmt.Sprintf("invalid index: %d", e.Index)
}
