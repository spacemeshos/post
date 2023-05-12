package postrs

// #cgo LDFLAGS: -lpost
// #include "prover.h"
import "C"

import (
	"errors"
	"fmt"
)

func OpenCLProviders() ([]Provider, error) {
	return cGetProviders()
}

func CPUProviderID() uint {
	return cCPUProviderID()
}

// ScryptPositionsResult is the result of a ScryptPositions call.
type ScryptPositionsResult struct {
	Output      []byte  // The output of the scrypt computation.
	IdxSolution *uint64 // The index of a solution to the proof of work (if checked for).
}

type option struct {
	providerID *uint

	commitment    []byte
	startPosition uint64
	endPosition   uint64
	n             uint32
	vrfDifficulty []byte
}

func (o *option) validate() error {
	if o.providerID == nil {
		return errors.New("`providerID` is required")
	}

	if o.commitment == nil {
		return errors.New("`commitment` is required")
	}

	if o.startPosition > o.endPosition {
		return fmt.Errorf("invalid `startPosition` and `endPosition`; expected: start <= end, given: %v > %v", o.startPosition, o.endPosition)
	}

	if o.n > 0 && o.n&(o.n-1) != 0 {
		return fmt.Errorf("invalid `n`; expected: power of 2, given: %v", o.n)
	}

	return nil
}

type OptionFunc func(*option) error

// WithProviderID sets the ID of the openCL provider to use.
func WithProviderID(id uint) OptionFunc {
	return func(opts *option) error {
		opts.providerID = new(uint)
		*opts.providerID = id
		return nil
	}
}

func WithCommitment(commitment []byte) OptionFunc {
	return func(opts *option) error {
		if len(commitment) != 32 {
			return fmt.Errorf("invalid `commitment` length; expected: 32, given: %v", len(commitment))
		}

		opts.commitment = commitment
		return nil
	}
}

func WithScryptN(n uint32) OptionFunc {
	return func(opts *option) error {
		opts.n = n
		return nil
	}
}

func WithVRFDifficulty(difficulty []byte) OptionFunc {
	return func(opts *option) error {
		if len(difficulty) != 32 {
			return fmt.Errorf("invalid `difficulty` length; expected: 32, given: %v", len(difficulty))
		}

		opts.vrfDifficulty = difficulty
		return nil
	}
}

type Scrypt struct {
	options *option
	init    *C.Initializer
}

func NewScrypt(opts ...OptionFunc) (*Scrypt, error) {
	options := &option{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	if err := options.validate(); err != nil {
		return nil, err
	}

	init, err := cNewInitializer(options)
	if err != nil {
		return nil, err
	}
	if *options.providerID != cCPUProviderID() {
		gpuMtx.Device(*options.providerID).Lock()
	}

	return &Scrypt{
		options: options,
		init:    init,
	}, nil
}

func (s *Scrypt) Close() {
	cFreeInitializer(s.init)
	if *s.options.providerID != cCPUProviderID() {
		gpuMtx.Device(*s.options.providerID).Unlock()
	}
}

type PositionsFunc func(*option) error

func WithStartAndEndPosition(start, end uint64) PositionsFunc {
	return func(opts *option) error {
		opts.startPosition = start
		opts.endPosition = end
		return nil
	}
}

// ScryptPositions computes the scrypt output for the given options.
func (s *Scrypt) Positions(start, end uint64) (ScryptPositionsResult, error) {
	s.options.startPosition = start
	s.options.endPosition = end

	if err := s.options.validate(); err != nil {
		return ScryptPositionsResult{}, err
	}

	output, idxSolution, err := cScryptPositions(s.init, s.options)
	return ScryptPositionsResult{
		Output:      output,
		IdxSolution: idxSolution,
	}, err
}
