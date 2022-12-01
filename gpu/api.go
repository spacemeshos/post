package gpu

import (
	"errors"
	"fmt"
	"time"

	"github.com/spacemeshos/post/config"
)

type ComputeProvider struct {
	ID         uint
	Model      string
	ComputeAPI ComputeAPIClass
}

var (
	providers     []ComputeProvider
	cpuProviderID uint
)

const CPUProviderName = "CPU"

func init() {
	providers = cGetProviders()
	for _, p := range providers {
		if p.ComputeAPI == ComputeAPIClassCPU {
			cpuProviderID = p.ID
			return
		}
	}
	panic("no CPU provider available")
}

// Providers returns a list of available compute providers.
func Providers() []ComputeProvider {
	return providers
}

// CPUProviderID returns the ID of the CPU provider.
func CPUProviderID() uint {
	return cpuProviderID
}

// Benchmark returns the hashes per second the selected compute provider achieves on the current machine.
func Benchmark(p ComputeProvider) (int, error) {
	endPosition := uint64(1 << 17)
	if p.Model == CPUProviderName {
		endPosition = uint64(1 << 14)
	}

	res, err := ScryptPositions(
		WithComputeProviderID(p.ID),
		WithCommitment(make([]byte, 32)),
		WithSalt(make([]byte, 32)),
		WithStartAndEndPosition(1, endPosition),
		WithBitsPerLabel(8),
	)
	if err != nil {
		return 0, err
	}

	return res.HashesPerSec, nil
}

// ScryptPositionsResult is the result of a ScryptPositions call.
type ScryptPositionsResult struct {
	Output       []byte  // The output of the scrypt computation.
	IdxSolution  *uint64 // The index of a solution to the proof of work (if checked for).
	HashesPerSec int     // The number of hashes computed per second.
	Stopped      bool    // Whether the computation was stopped.
}

type option struct {
	computeProviderID uint

	commitment []byte
	salt       []byte

	startPosition uint64
	endPosition   uint64

	bitsPerLabel uint32

	computeLeaves bool
	computePow    bool

	n, r, p uint32
	d       []byte
}

func (o *option) optionBits() uint32 {
	var bits uint32
	if o.computeLeaves {
		bits |= (1 << 0)
	}
	if o.computePow {
		bits |= (1 << 1)
	}
	return bits
}

func (o *option) validate() error {
	if o.computeLeaves && (o.bitsPerLabel < config.MinBitsPerLabel || o.bitsPerLabel > config.MaxBitsPerLabel) {
		return fmt.Errorf("invalid `bitsPerLabel`; expected: %d-%d, given: %v", config.MinBitsPerLabel, config.MaxBitsPerLabel, o.bitsPerLabel)
	}

	return nil
}

type OptionFunc func(*option) error

// WithComputeProviderID instructs scrypt to use the specified compute provider.
func WithComputeProviderID(id uint) OptionFunc {
	return func(opts *option) error {
		opts.computeProviderID = id
		return nil
	}
}

// WithCommitment instructs scrypt to use the specified commitment (seed) to calculate the output.
func WithCommitment(commitment []byte) OptionFunc {
	return func(opts *option) error {
		if len(commitment) != 32 {
			return fmt.Errorf("invalid `commitment` length; expected: 32, given: %v", len(commitment))
		}

		opts.commitment = commitment
		return nil
	}
}

// WithSalt instructs scrypt to use the specified salt to calculate the output.
func WithSalt(salt []byte) OptionFunc {
	return func(opts *option) error {
		if len(salt) != 32 {
			return fmt.Errorf("invalid `salt` length; expected: 32, given: %v", len(salt))
		}

		opts.salt = salt
		return nil
	}
}

// WithStartAndEndPosition instructs scrypt to compute the scrypt output for the specified range of positions.
func WithStartAndEndPosition(start, end uint64) OptionFunc {
	return func(opts *option) error {
		opts.startPosition = start
		opts.endPosition = end
		return nil
	}
}

// WithBitsPerLabel instructs scrypt to use the specified number of bits per label.
func WithBitsPerLabel(bitsPerLabel uint32) OptionFunc {
	return func(opts *option) error {
		opts.bitsPerLabel = bitsPerLabel
		return nil
	}
}

// WithComputeLeafs instructs scrypt to compute leafs or not.
// By default computing leafs is enabled.
func WithComputeLeaves(enabled bool) OptionFunc {
	return func(opts *option) error {
		opts.computeLeaves = enabled
		return nil
	}
}

// WithComputePow instructs scrypt to compute a proof of work or not.
// If difficulty is nil, no PoW will be computed. Otherwise it specifies the difficulty
// of the PoW to be computed (higher values are more difficult).
// By default computing proof of work is disabled.
func WithComputePow(difficulty []byte) OptionFunc {
	return func(opts *option) error {
		if difficulty == nil {
			opts.computePow = false
			return nil
		}

		if len(difficulty) != 32 {
			return fmt.Errorf("invalid `difficulty` length; expected: 32, given: %v", len(difficulty))
		}

		opts.computePow = true
		opts.d = difficulty
		return nil
	}
}

// ScryptPositions computes the scrypt output for the given options.
func ScryptPositions(opts ...OptionFunc) (*ScryptPositionsResult, error) {
	options := &option{
		n:             512,
		r:             1,
		p:             1,
		computeLeaves: true,
		d:             make([]byte, 32),
	}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	if err := options.validate(); err != nil {
		return nil, err
	}

	// Wait for the stop flag clearance for avoiding a race condition which can
	// occur if ScryptPositions is called immediately after a prior Stop call.
	var i int
	for {
		i++
		cleared := cStopCleared()
		if cleared {
			break
		}
		if i == 20 {
			return nil, errors.New("stop flag clearance timeout")
		}
		time.Sleep(100 * time.Millisecond)
	}

	output, idxSolution, hashesPerSec, retVal := cScryptPositions(options)

	switch StopResult(retVal) {
	case StopResultPowFound:
		return &ScryptPositionsResult{output, &idxSolution, hashesPerSec, false}, nil
	case StopResultOk:
		return &ScryptPositionsResult{output, nil, hashesPerSec, false}, nil
	case StopResultError:
		return nil, fmt.Errorf("gpu-post error")
	case StopResultErrorTimeout:
		return nil, fmt.Errorf("gpu-post error: timeout")
	case StopResultErrorAlready:
		return nil, fmt.Errorf("gpu-post error: already stopped")
	case StopResultErrorCancelled:
		return &ScryptPositionsResult{output, nil, hashesPerSec, true}, nil
	case StopResultErrorNoCompoteOptions:
		return nil, fmt.Errorf("gpu-post error: no compute options")
	case StopResultErrorInvalidParameter:
		return nil, fmt.Errorf("gpu-post error: invalid param")
	case StopResultErrorInvalidProvider:
		return nil, fmt.Errorf("gpu-post error: invalid provider")
	default:
		panic(fmt.Sprintf("unreachable reVal %d", retVal))
	}
}

func Stop() StopResult {
	return cStop(20000)
}
