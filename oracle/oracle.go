package oracle

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/shared"
)

type Challenge = shared.Challenge

type workOracleOption struct {
	computeProviderID uint

	commitment []byte
	salt       []byte

	startPosition uint64
	endPosition   uint64

	bitsPerLabel uint32

	computeLeaves bool
	difficulty    []byte
}

type workOracleOptionFunc func(*workOracleOption) error

func WithComputeProviderID(id uint) workOracleOptionFunc {
	return func(opts *workOracleOption) error {
		opts.computeProviderID = id
		return nil
	}
}

func WithCommitment(commitment []byte) workOracleOptionFunc {
	return func(opts *workOracleOption) error {
		if len(commitment) != 32 {
			return fmt.Errorf("invalid `commitment` length; expected: 32, given: %v", len(commitment))
		}

		opts.commitment = commitment
		return nil
	}
}

func WithPosition(position uint64) workOracleOptionFunc {
	return func(opts *workOracleOption) error {
		opts.startPosition = position
		opts.endPosition = position
		return nil
	}
}

func WithStartAndEndPosition(start, end uint64) workOracleOptionFunc {
	return func(opts *workOracleOption) error {
		opts.startPosition = start
		opts.endPosition = end
		return nil
	}
}

func WithBitsPerLabel(bitsPerLabel uint32) workOracleOptionFunc {
	return func(opts *workOracleOption) error {
		if bitsPerLabel < config.MinBitsPerLabel || bitsPerLabel > config.MaxBitsPerLabel {
			return fmt.Errorf("invalid `bitsPerLabel`; expected: %d-%d, given: %v", config.MinBitsPerLabel, config.MaxBitsPerLabel, bitsPerLabel)
		}
		opts.bitsPerLabel = bitsPerLabel
		return nil
	}
}

// WithComputeLeaves instructs the oracle to compute the labels for PoST or not.
// By default computing leaves is enabled. It can be switched off to save time
// when continuing a run to compute a proof of work.
func WithComputeLeaves(enabled bool) workOracleOptionFunc {
	return func(opts *workOracleOption) error {
		opts.computeLeaves = enabled
		return nil
	}
}

// WithComputePow instructs the oracle to compute a proof of work or not.
// If difficulty is nil, no PoW will be computed. Otherwise it specifies the difficulty
// of the PoW to be computed (higher values are more difficult).
// By default computing proof of work is disabled.
func WithComputePow(difficulty []byte) workOracleOptionFunc {
	return func(opts *workOracleOption) error {
		if difficulty != nil && len(difficulty) != 32 {
			return fmt.Errorf("invalid `difficulty` length; expected: 32, given: %v", len(difficulty))
		}

		opts.difficulty = difficulty
		return nil
	}
}

type WorkOracleResult struct {
	Output []byte
	Nonce  *uint64
}

func WorkOracle(opts ...workOracleOptionFunc) (WorkOracleResult, error) {
	options := &workOracleOption{
		computeProviderID: *gpu.CPUProviderID(),
		salt:              make([]byte, 32), // TODO(moshababo): apply salt
		computeLeaves:     true,
	}

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return WorkOracleResult{}, err
		}
	}

	res, err := gpu.ScryptPositions(
		gpu.WithComputeProviderID(options.computeProviderID),
		gpu.WithCommitment(options.commitment),
		gpu.WithSalt(options.salt),
		gpu.WithStartAndEndPosition(options.startPosition, options.endPosition),
		gpu.WithBitsPerLabel(options.bitsPerLabel),
		gpu.WithComputeLeaves(options.computeLeaves),
		gpu.WithComputePow(options.difficulty),
	)
	if err != nil {
		return WorkOracleResult{}, err
	}

	return WorkOracleResult{
		Output: res.Output,
		Nonce:  res.IdxSolution,
	}, nil
}

func FastOracle(ch Challenge, nonce uint32, label []byte) [32]byte {
	input := make([]byte, 32+4+len(label))

	copy(input, ch)
	binary.LittleEndian.PutUint32(input[32:], nonce)
	copy(input[36:], label[:])

	return sha256.Sum256(input)
}
