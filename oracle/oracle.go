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

func WorkOracle(opts ...workOracleOptionFunc) ([]byte, error) {
	options := &workOracleOption{
		computeProviderID: uint(gpu.CPUProviderID()),
		salt:              make([]byte, 32), // TODO(moshababo): apply salt
	}

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	res, err := gpu.ScryptPositions(
		gpu.WithComputeProviderID(options.computeProviderID),
		gpu.WithCommitment(options.commitment),
		gpu.WithSalt(options.salt),
		gpu.WithStartAndEndPosition(options.startPosition, options.endPosition),
		gpu.WithBitsPerLabel(options.bitsPerLabel),
	)
	if err != nil {
		return nil, err
	}

	return res.Output, nil
}

func FastOracle(ch Challenge, nonce uint32, label []byte) [32]byte {
	input := make([]byte, 32+4+len(label))

	copy(input, ch)
	binary.LittleEndian.PutUint32(input[32:], nonce)
	copy(input[36:], label[:])

	return sha256.Sum256(input)
}
