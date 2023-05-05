package oracle

import (
	"errors"
	"fmt"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/internal/postrs"
)

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

	if o.vrfDifficulty == nil {
		return errors.New("`vrfDifficulty` is required")
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

// WithVRFDifficulty sets the difficulty for the VRF Nonce.
// It is used as a PoW to make creating identities expensive and thereby prevent Sybil attacks.
func WithVRFDifficulty(difficulty []byte) OptionFunc {
	return func(opts *option) error {
		if len(difficulty) != 32 {
			return fmt.Errorf("invalid `difficulty` length; expected: 32, given: %v", len(difficulty))
		}

		opts.vrfDifficulty = difficulty
		return nil
	}
}

// WithScryptParams sets the parameters for the scrypt algorithm.
// At the moment only configuring N is supported. r and p are fixed at 1 (due to limitations in the OpenCL implementation).
func WithScryptParams(params config.ScryptParams) OptionFunc {
	return func(opts *option) error {
		if params.P != 1 || params.R != 1 {
			return errors.New("invalid scrypt params: only r = 1, p = 1 are supported for initialization")
		}

		opts.n = params.N
		return nil
	}
}

// WorkOracleResult is the result of a call to WorkOracle.
// It contains the computed labels and the nonce for the a proof of work.
type WorkOracleResult struct {
	Output []byte  // Output are the computed labels
	Nonce  *uint64 // Nonce is the nonce of the proof of work
}

// WorkOracle computes labels for a given challenge for a Node with the provided CommitmentATX ID.
// The labels are computed using the specified compute provider (default: CPU).
func WorkOracle(opts ...OptionFunc) (WorkOracleResult, error) {
	options := &option{}
	options.providerID = new(uint)
	*options.providerID = postrs.CPUProviderID()

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return WorkOracleResult{}, err
		}
	}

	if err := options.validate(); err != nil {
		return WorkOracleResult{}, err
	}

	res, err := postrs.ScryptPositions(
		postrs.WithProviderID(*options.providerID),
		postrs.WithCommitment(options.commitment),
		postrs.WithStartAndEndPosition(options.startPosition, options.endPosition),
		postrs.WithScryptN(options.n),
		postrs.WithVRFDifficulty(options.vrfDifficulty),
	)
	if err != nil {
		return WorkOracleResult{}, err
	}

	return WorkOracleResult{
		Output: res.Output,
		Nonce:  res.IdxSolution,
	}, nil
}
