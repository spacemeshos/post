package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "post.h"
import "C"

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"unsafe"

	"go.uber.org/zap"

	"github.com/spacemeshos/post/shared"
)

type ScryptParams = C.ScryptParams

type HexEncoded []byte

func (h HexEncoded) String() string {
	return hex.EncodeToString(h)
}

// Translate scrypt parameters expressed as N,R,P to Nfactor, Rfactor and Pfactor
// that are understood by scrypt-jane.
// Relation:
// N = 1 << (nfactor + 1)
// r = 1 << rfactor
// p = 1 << pfactor
func TranslateScryptParams(n, r, p uint) ScryptParams {
	return ScryptParams{
		nfactor: C.uint8_t(math.Log2(float64(n))) - 1,
		rfactor: C.uint8_t(math.Log2(float64(r))),
		pfactor: C.uint8_t(math.Log2(float64(p))),
	}
}

type postOptions struct {
	powCreatorId []byte
}

type PostOptionFunc func(*postOptions) error

func WithPowCreator(id []byte) PostOptionFunc {
	return func(opts *postOptions) error {
		opts.powCreatorId = id
		return nil
	}
}

func GenerateProof(dataDir string, challenge []byte, logger *zap.Logger, nonces, threads uint, K1, K2 uint32, powDifficulty [32]byte, powFlags PowFlags, options ...PostOptionFunc) (*shared.Proof, error) {
	opts := postOptions{}
	for _, o := range options {
		if err := o(&opts); err != nil {
			return nil, err
		}
	}
	if logger != nil {
		setLogCallback(logger)
	}

	dataDirPtr := C.CString(dataDir)
	defer C.free(unsafe.Pointer(dataDirPtr))

	challengePtr := C.CBytes(challenge)
	defer C.free(challengePtr)

	var powCreatorId unsafe.Pointer
	if opts.powCreatorId != nil {
		logger.Debug("Proving with PoW creator ID", zap.Stringer("id", HexEncoded(opts.powCreatorId)))
		powCreatorId = C.CBytes(opts.powCreatorId)
		defer C.free(powCreatorId)
	}

	config := C.Config{
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
		(*C.uchar)(powCreatorId),
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
	inner     *C.Verifier
	closeOnce sync.Once
}

// Create a new verifier.
// The verifier must be closed after use with Close().
func NewVerifier(powFlags PowFlags) (*Verifier, error) {
	verifier := Verifier{}
	result := C.new_verifier(powFlags, &verifier.inner)
	if result != C.Ok {
		return nil, fmt.Errorf("failed to create verifier")
	}
	return &verifier, nil
}

func (v *Verifier) Close() error {
	v.closeOnce.Do(func() { C.free_verifier(v.inner) })
	return nil
}

func (v *Verifier) VerifyProof(
	proof *shared.Proof,
	metadata *shared.ProofMetadata,
	logger *zap.Logger,
	k1, k2, k3 uint32,
	powDifficulty [32]byte,
	scryptParams ScryptParams,
	options ...PostOptionFunc,
) error {
	opts := postOptions{}
	for _, o := range options {
		if err := o(&opts); err != nil {
			return err
		}
	}
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

	config := C.Config{
		k1:     C.uint32_t(k1),
		k2:     C.uint32_t(k2),
		k3:     C.uint32_t(k3),
		scrypt: scryptParams,
	}
	for i, b := range powDifficulty {
		config.pow_difficulty[i] = C.uchar(b)
	}

	indicesSliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&proof.Indices))
	cProof := C.Proof{
		nonce: C.uint32_t(proof.Nonce),
		pow:   C.uint64_t(proof.Pow),
		indices: C.ArrayU8{
			ptr: (*C.uchar)(unsafe.Pointer(indicesSliceHdr.Data)),
			len: C.size_t(indicesSliceHdr.Len),
			cap: C.size_t(indicesSliceHdr.Cap),
		},
	}

	if opts.powCreatorId != nil {
		logger.Debug("verifying POST with PoW creator ID", zap.Stringer("id", HexEncoded(opts.powCreatorId)))
		minerIdSliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&opts.powCreatorId))
		cProof.pow_creator = C.ArrayU8{
			ptr: (*C.uchar)(unsafe.Pointer(minerIdSliceHdr.Data)),
			len: C.size_t(minerIdSliceHdr.Len),
			cap: C.size_t(minerIdSliceHdr.Cap),
		}
	}

	cMetadata := C.ProofMetadata{
		node_id:           *(*[32]C.uchar)(unsafe.Pointer(&metadata.NodeId[0])),
		commitment_atx_id: *(*[32]C.uchar)(unsafe.Pointer(&metadata.CommitmentAtxId[0])),
		challenge:         *(*[32]C.uchar)(unsafe.Pointer(&metadata.Challenge[0])),
		num_units:         C.uint32_t(metadata.NumUnits),
		labels_per_unit:   C.uint64_t(metadata.LabelsPerUnit),
	}
	result := C.verify_proof(
		v.inner,
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
