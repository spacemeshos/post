package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "prover.h"
import "C"

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"unsafe"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/shared"
)

// gpuMtx is a mutual exclusion lock for calls to gpu functions. It is required
// to prevent concurrent calls to the same GPU from multiple goroutines.
var gpuMtx deviceMutex

type deviceMutex struct {
	mtx    sync.Mutex
	device map[uint]*sync.Mutex
}

func (g *deviceMutex) Device(deviceId uint) *sync.Mutex {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if g.device == nil {
		g.device = make(map[uint]*sync.Mutex)
	}

	if _, ok := g.device[deviceId]; !ok {
		g.device[deviceId] = new(sync.Mutex)
	}

	return g.device[deviceId]
}

type DeviceClass int

const (
	ClassCPU = DeviceClass((C.DeviceClass)(C.CPU))
	ClassGPU = DeviceClass((C.DeviceClass)(C.GPU))
)

type ComputeProvider struct {
	ID         uint
	Model      string
	DeviceType DeviceClass
}

func (c DeviceClass) String() string {
	switch c {
	case ClassCPU:
		return "CPU"
	case ClassGPU:
		return "GPU"
	default:
		return "Unknown"
	}
}

var (
	ErrInvalidProviderID = errors.New("invalid provider ID")

	ErrInvalidLabelsRange = errors.New("invalid labels range")
	ErrOclError           = errors.New("OpenCL error")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrFetchProviders     = errors.New("failed to fetch providers")
)

const (
	// LabelLength is the length of the label in bytes.
	LabelLength = 16
)

func InitResultToError(retVal uint32) error {
	switch retVal {
	case C.InitializeOk:
		return nil
	case C.InitializeInvalidLabelsRange:
		return ErrInvalidLabelsRange
	case C.InitializeOclError:
		return ErrOclError
	case C.InitializeInvalidArgument:
		return ErrInvalidArgument
	case C.InitializeFailedToGetProviders:
		return ErrFetchProviders
	default:
		return fmt.Errorf("unknown error")
	}
}

func cScryptPositions(opt *option) ([]byte, *uint64, error) {
	// disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace) // TODO(mafa): make this configurable

	if *opt.providerID != cCPUProviderID() {
		gpuMtx.Device(*opt.providerID).Lock()
		defer gpuMtx.Device(*opt.providerID).Unlock()
	}

	cProviderId := C.uint32_t(*opt.providerID)
	cN := C.uintptr_t(opt.n)
	cCommitment := C.CBytes(opt.commitment)
	defer C.free(cCommitment)
	cDifficulty := C.CBytes(opt.vrfDifficulty)
	defer C.free(cDifficulty)
	init := C.new_initializer(cProviderId, cN, (*C.uchar)(cCommitment), (*C.uchar)(cDifficulty))
	if init == nil {
		return nil, nil, ErrInvalidProviderID
	}

	defer C.free_initializer(init)

	outputSize := LabelLength * (opt.endPosition - opt.startPosition + 1)
	cStartPosition := C.uint64_t(opt.startPosition)
	cEndPosition := C.uint64_t(opt.endPosition)
	cOutputSize := C.size_t(outputSize)
	cOut := (C.calloc(cOutputSize, 1))
	defer C.free(cOut)

	var cIdxSolution C.uint64_t
	retVal := C.initialize(init, cStartPosition, cEndPosition, (*C.uint8_t)(cOut), &cIdxSolution)
	if err := InitResultToError(retVal); err != nil {
		return nil, nil, err
	}

	var vrfNonce *uint64
	if cIdxSolution != math.MaxUint64 { // TODO(mafa): we should find a better way to indicate no solution (e.g. InitializeOk = no solution, InitializeOkPow = solution)
		vrfNonce = new(uint64)
		*vrfNonce = uint64(cIdxSolution)
	}

	output := C.GoBytes(cOut, C.int(cOutputSize))
	return output, vrfNonce, nil
}

func cCPUProviderID() uint {
	return C.CPU_PROVIDER_ID
}

func cGetProviders() ([]ComputeProvider, error) {
	// disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace) // TODO(mafa): make this configurable

	cNumProviders := C.get_providers_count()
	if cNumProviders == 0 {
		return nil, ErrFetchProviders
	}

	cProviders := make([]C.Provider, cNumProviders)
	providers := make([]ComputeProvider, cNumProviders)
	retVal := C.get_providers(&cProviders[0], cNumProviders)
	if err := InitResultToError(retVal); err != nil {
		return nil, err
	}

	for i := uint(0); i < uint(cNumProviders); i++ {
		providers[i].ID = (uint)(cProviders[i].id)
		// TODO(mafa): `name` should be char instead of `uint8_t` then this cast isn't needed to work around staticcheck
		providers[i].Model = C.GoString((*C.char)(unsafe.Pointer((&cProviders[i].name[0]))))
		providers[i].DeviceType = DeviceClass(cProviders[i].class_)
	}

	return providers, nil
}

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

func GenerateProof(dataDir string, challenge []byte, cfg config.Config, nonces uint, threads uint, powScrypt config.ScryptParams) (*shared.Proof, error) {
	// disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace) // TODO(mafa): make this configurable

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
	// disabled for now (calling it more than once crashes the program)
	// C.configure_logging(C.Trace) // TODO(mafa): make this configurable

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
		1, // TODO(mafa): remove this argument after post-rs merge
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
