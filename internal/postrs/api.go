package postrs

import (
	"errors"
	"fmt"
)

func OpenCLProviders() ([]ComputeProvider, error) {
	return cGetProviders()
}

func CPUProviderID() (uint, error) {
	providers, err := OpenCLProviders()
	if err != nil {
		return 0, err
	}
	for _, p := range providers {
		if p.DeviceType == ClassCPU {
			return p.ID, nil
		}
	}
	return 0, errors.New("no CPU provider available")
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

func WithCommitment(commitment []byte) OptionFunc {
	return func(opts *option) error {
		if len(commitment) != 32 {
			return fmt.Errorf("invalid `commitment` length; expected: 32, given: %v", len(commitment))
		}

		opts.commitment = commitment
		return nil
	}
}

func WithStartAndEndPosition(start, end uint64) OptionFunc {
	return func(opts *option) error {
		opts.startPosition = start
		opts.endPosition = end
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

// ScryptPositions computes the scrypt output for the given options.
func ScryptPositions(opts ...OptionFunc) (ScryptPositionsResult, error) {
	options := &option{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return ScryptPositionsResult{}, err
		}
	}

	if err := options.validate(); err != nil {
		return ScryptPositionsResult{}, err
	}

	output, idxSolution, err := cScryptPositions(options)
	return ScryptPositionsResult{
		Output:      output,
		IdxSolution: idxSolution,
	}, err
}
