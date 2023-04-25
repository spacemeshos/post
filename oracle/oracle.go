package oracle

import (
	"errors"
	"fmt"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/internal/gpu"
)

type option struct {
	computeProviderID uint

	commitment []byte
	salt       []byte

	startPosition uint64
	endPosition   uint64

	bitsPerLabel  uint32
	computeLeaves bool

	difficulty []byte

	scrypt *config.ScryptParams
}

func (o *option) validate() error {
	if o.commitment == nil {
		return errors.New("`commitment` is required")
	}

	if o.computeLeaves && (o.bitsPerLabel < config.MinBitsPerLabel || o.bitsPerLabel > config.MaxBitsPerLabel) {
		return fmt.Errorf("invalid `bitsPerLabel`; expected: %d-%d, given: %v", config.MinBitsPerLabel, config.MaxBitsPerLabel, o.bitsPerLabel)
	}

	if o.scrypt == nil {
		return errors.New("scrypt parameters are required")
	}

	return nil
}

type OptionFunc func(*option) error

// WithComputeProviderID sets the ID of the compute provider to use.
func WithComputeProviderID(id uint) OptionFunc {
	return func(opts *option) error {
		opts.computeProviderID = id
		return nil
	}
}

// WithCommitment sets the commitment to use for the oracle.
func WithCommitment(commitment []byte) OptionFunc {
	return func(opts *option) error {
		if len(commitment) != 32 {
			return fmt.Errorf("invalid `commitment` length; expected: 32, given: %v", len(commitment))
		}

		opts.commitment = commitment
		return nil
	}
}

// WithSalt sets the salt to use for the oracle.
func WithSalt(salt []byte) OptionFunc {
	return func(opts *option) error {
		if len(salt) != 32 {
			return fmt.Errorf("invalid `salt` length; expected: 32, given: %v", len(salt))
		}

		opts.salt = salt
		return nil
	}
}

// WithPosition sets the index of one label to compute.
func WithPosition(position uint64) OptionFunc {
	return func(opts *option) error {
		opts.startPosition = position
		opts.endPosition = position
		return nil
	}
}

// WithStartAndEndPosition sets the range of indices of labels for the oracle to compute.
func WithStartAndEndPosition(start, end uint64) OptionFunc {
	return func(opts *option) error {
		opts.startPosition = start
		opts.endPosition = end
		return nil
	}
}

// WithBitsPerLabel sets the number of bits per label.
func WithBitsPerLabel(bitsPerLabel uint32) OptionFunc {
	return func(opts *option) error {
		opts.bitsPerLabel = bitsPerLabel
		return nil
	}
}

// WithComputeLeaves instructs the oracle to compute the labels for PoST or not.
// By default computing leaves is enabled. It can be switched off to save time
// when continuing a run to compute a proof of work.
func WithComputeLeaves(enabled bool) OptionFunc {
	return func(opts *option) error {
		opts.computeLeaves = enabled
		return nil
	}
}

// WithComputePow instructs the oracle to compute a proof of work or not.
// If difficulty is nil, no PoW will be computed. Otherwise it specifies the difficulty
// of the PoW to be computed (higher values are more difficult).
// By default computing proof of work is disabled.
func WithComputePow(difficulty []byte) OptionFunc {
	return func(opts *option) error {
		if difficulty != nil && len(difficulty) != 32 {
			return fmt.Errorf("invalid `difficulty` length; expected: 32, given: %v", len(difficulty))
		}

		opts.difficulty = difficulty
		return nil
	}
}

func WithScryptParams(params config.ScryptParams) OptionFunc {
	return func(opts *option) error {
		opts.scrypt = &params
		return nil
	}
}

// WorkOracleResult is the result of a call to WorkOracle.
// It contains the computed labels and the nonce as a proof of work.
type WorkOracleResult struct {
	Output []byte  // Output are the computed labels (only if `WithComputeLeaves` is true - default yes).
	Nonce  *uint64 // Nonce is the nonce of the proof of work (only if `WithComputePow` is true - default no).
}

// WorkOracle computes labels for a given challenge for a Node with the provided CommitmentATX ID.
// The labels are computed using the specified compute provider (default: CPU).
func WorkOracle(opts ...OptionFunc) (WorkOracleResult, error) {
	options := &option{
		computeProviderID: gpu.CPUProviderID(),
		salt:              make([]byte, 32), // TODO(moshababo): apply salt
		computeLeaves:     true,
		bitsPerLabel:      config.BitsPerLabel,
	}

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return WorkOracleResult{}, err
		}
	}

	if err := options.validate(); err != nil {
		return WorkOracleResult{}, err
	}

	res, err := gpu.ScryptPositions(
		gpu.WithComputeProviderID(options.computeProviderID),
		gpu.WithCommitment(options.commitment),
		gpu.WithSalt(options.salt),
		gpu.WithStartAndEndPosition(options.startPosition, options.endPosition),
		gpu.WithBitsPerLabel(options.bitsPerLabel),
		gpu.WithComputeLeaves(options.computeLeaves),
		gpu.WithComputePow(options.difficulty),
		gpu.WithScryptParams(options.scrypt.N, options.scrypt.R, options.scrypt.P),
	)
	if err != nil {
		return WorkOracleResult{}, err
	}

	return WorkOracleResult{
		Output: res.Output,
		Nonce:  res.IdxSolution,
	}, nil
}
