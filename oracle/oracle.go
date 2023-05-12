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

type WorkOracle struct {
	options *option
	scrypt  *postrs.Scrypt
}

// New returns a WorkOracle that can compute labels for a given challenge for a Node with the provided CommitmentATX ID.
// The labels are computed using the specified compute provider (default: CPU).
func New(opts ...OptionFunc) (*WorkOracle, error) {
	options := &option{}
	options.providerID = new(uint)
	*options.providerID = postrs.CPUProviderID()

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	if err := options.validate(); err != nil {
		return nil, err
	}

	scrypt, err := postrs.NewScrypt(
		postrs.WithProviderID(*options.providerID),
		postrs.WithCommitment(options.commitment),
		postrs.WithScryptN(options.n),
		postrs.WithVRFDifficulty(options.vrfDifficulty),
	)
	if err != nil {
		return nil, err
	}

	return &WorkOracle{
		options: options,
		scrypt:  scrypt,
	}, nil
}

func (w *WorkOracle) Close() {
	w.scrypt.Close()
}

// WorkOracleResult is the result of a call to WorkOracle.
// It contains the computed labels and a nonce for a proof of work.
type WorkOracleResult struct {
	Output []byte  // Output are the computed labels
	Nonce  *uint64 // Nonce is the nonce of the proof of work
}

func (w *WorkOracle) Position(p uint64) (WorkOracleResult, error) {
	return w.Positions(p, p)
}

func (w *WorkOracle) Positions(start, end uint64) (WorkOracleResult, error) {
	if start > end {
		return WorkOracleResult{}, fmt.Errorf("invalid `startPosition` and `endPosition`; expected: start <= end, given: %v > %v", start, end)
	}

	res, err := w.scrypt.Positions(start, end)
	if err != nil {
		return WorkOracleResult{}, err
	}

	return WorkOracleResult{
		Output: res.Output,
		Nonce:  res.IdxSolution,
	}, nil
}
