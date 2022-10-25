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
	cpuProviderID int
)

func init() {
	providers = cGetProviders()
	cpuProviderID = int(filterCPUProvider(providers).ID)
}

func Providers() []ComputeProvider {
	return providers
}

func CPUProviderID() int {
	return cpuProviderID
}

func filterCPUProvider(providers []ComputeProvider) ComputeProvider {
	for _, p := range providers {
		if p.Model == "CPU" {
			return p
		}
	}
	panic("unreachable")
}

func Benchmark(p ComputeProvider) (int, error) {
	endPosition := uint64(1 << 17)
	if p.Model == "CPU" {
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

type ScryptPositionsResult struct {
	Output       []byte
	IdxSolution  uint64
	HashesPerSec int
	Stopped      bool
}

type scryptPositionOption struct {
	computeProviderID uint

	commitment []byte
	salt       []byte

	startPosition uint64
	endPosition   uint64

	bitsPerLabel uint32

	computeLeafs bool
	computePow   bool

	n, r, p uint32
}

func (o *scryptPositionOption) optionBits() uint32 {
	var bits uint32
	if o.computeLeafs {
		bits |= (1 << 0)
	}
	if o.computePow {
		bits |= (1 << 1)
	}
	return bits
}

type scryptPositionOptionFunc func(*scryptPositionOption) error

func WithComputeProviderID(id uint) scryptPositionOptionFunc {
	return func(opts *scryptPositionOption) error {
		opts.computeProviderID = id
		return nil
	}
}

func WithCommitment(commitment []byte) scryptPositionOptionFunc {
	return func(opts *scryptPositionOption) error {
		if len(commitment) != 32 {
			return fmt.Errorf("invalid `commitment` length; expected: 32, given: %v", len(commitment))
		}

		opts.commitment = commitment
		return nil
	}
}

func WithSalt(salt []byte) scryptPositionOptionFunc {
	return func(opts *scryptPositionOption) error {
		if len(salt) != 32 {
			return fmt.Errorf("invalid `salt` length; expected: 32, given: %v", len(salt))
		}

		opts.salt = salt
		return nil
	}
}

func WithStartAndEndPosition(start, end uint64) scryptPositionOptionFunc {
	return func(opts *scryptPositionOption) error {
		opts.startPosition = start
		opts.endPosition = end
		return nil
	}
}

func WithBitsPerLabel(bitsPerLabel uint32) scryptPositionOptionFunc {
	return func(opts *scryptPositionOption) error {
		if bitsPerLabel < config.MinBitsPerLabel || bitsPerLabel > config.MaxBitsPerLabel {
			return fmt.Errorf("invalid `bitsPerLabel`; expected: %d-%d, given: %v", config.MinBitsPerLabel, config.MaxBitsPerLabel, bitsPerLabel)
		}
		opts.bitsPerLabel = bitsPerLabel
		return nil
	}
}

// WithComputeLeafs instructs scrypt to compute leafs or not.
// By default computing leafs is enabled.
func WithComputeLeafs(enabled bool) scryptPositionOptionFunc {
	return func(opts *scryptPositionOption) error {
		opts.computeLeafs = enabled
		return nil
	}
}

// WithComputePow instructs scrypt to compute a proof of work or not.
// By default computing proof of work is disabled.
func WithComputePow(enabled bool) scryptPositionOptionFunc {
	return func(opts *scryptPositionOption) error {
		opts.computePow = enabled
		return nil
	}
}

func ScryptPositions(opts ...scryptPositionOptionFunc) (*ScryptPositionsResult, error) {
	options := &scryptPositionOption{
		n:            512,
		r:            1,
		p:            1,
		computeLeafs: true,
	}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
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

	switch retVal {
	case 1:
		return &ScryptPositionsResult{output, idxSolution, hashesPerSec, false}, nil
	case 0:
		return &ScryptPositionsResult{output, idxSolution, hashesPerSec, false}, nil
	case -1:
		return nil, fmt.Errorf("gpu-post error")
	case -2:
		return nil, fmt.Errorf("gpu-post error: timeout")
	case -3:
		return nil, fmt.Errorf("gpu-post error: already stopped")
	case -4:
		return &ScryptPositionsResult{output, idxSolution, hashesPerSec, true}, nil
	case -5:
		return nil, fmt.Errorf("gpu-post error: no compute options")
	case -6:
		return nil, fmt.Errorf("gpu-post error: invalid param")
	case -7:
		return nil, fmt.Errorf("gpu-post error: invalid provider")
	default:
		panic(fmt.Sprintf("unreachable reVal %d", retVal))
	}
}

func Stop() StopResult {
	return cStop(20000)
}
