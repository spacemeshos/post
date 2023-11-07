package oracle

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/shared"
)

// ErrWorkOracleClosed is returned when calling a method on an already closed WorkOracle instance.
var ErrWorkOracleClosed = errors.New("work oracle has been closed")

type option struct {
	providerID *uint32

	commitment    []byte
	n             uint
	vrfDifficulty []byte

	logger *zap.Logger

	maxRetries int
	retryDelay time.Duration

	scrypter postrs.Scrypter
}

func (o *option) validate() error {
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

// OptionFunc is a function that sets an option for a WorkOracle instance.
type OptionFunc func(*option) error

// WithProviderID sets the ID of the OpenCL provider to use.
func WithProviderID(id *uint32) OptionFunc {
	return func(opts *option) error {
		opts.providerID = id
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
func WithScryptParams(params shared.ScryptParams) OptionFunc {
	return func(opts *option) error {
		if params.P != 1 || params.R != 1 {
			return errors.New("invalid scrypt params: only r = 1, p = 1 are supported for initialization")
		}

		opts.n = params.N
		return nil
	}
}

// WithLogger sets the logger to use.
func WithLogger(logger *zap.Logger) OptionFunc {
	return func(opts *option) error {
		opts.logger = logger
		return nil
	}
}

// WithRetryDelay sets the delay between retries for a single initialization invocation.
func WithRetryDelay(retryDelay time.Duration) OptionFunc {
	return func(opts *option) error {
		opts.retryDelay = retryDelay
		return nil
	}
}

// WithMaxRetries sets the maximum number of retries for a single initialization invocation.
func WithMaxRetries(maxRetries int) OptionFunc {
	return func(opts *option) error {
		opts.maxRetries = maxRetries
		return nil
	}
}

func withScrypter(scrypter postrs.Scrypter) OptionFunc {
	return func(opts *option) error {
		opts.scrypter = scrypter
		return nil
	}
}

// WorkOracle is a service that can compute labels for a given Node ID and CommitmentATX ID.
type WorkOracle struct {
	options *option
	scrypt  postrs.Scrypter
}

// Lazy initialized Scrypter.
type LazyScrypter struct {
	init     func() (postrs.Scrypter, error)
	initOnce sync.Once
	scrypt   postrs.Scrypter
	err      error
}

func (l *LazyScrypter) Positions(start, end uint64) (postrs.ScryptPositionsResult, error) {
	l.initOnce.Do(func() {
		l.scrypt, l.err = l.init()
	})
	if l.err != nil {
		return postrs.ScryptPositionsResult{}, fmt.Errorf("initializing scrypter: %w", l.err)
	}
	return l.scrypt.Positions(start, end)
}

func (l *LazyScrypter) Close() error {
	if l.scrypt != nil {
		return l.scrypt.Close()
	}
	return nil
}

// New returns a WorkOracle. If not specified, the labels are computed using the default (CPU) provider.
func New(opts ...OptionFunc) (*WorkOracle, error) {
	options := &option{
		maxRetries: 10,
		retryDelay: time.Second,
		logger:     zap.NewNop(),
	}

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	if err := options.validate(); err != nil {
		return nil, err
	}

	scrypt := options.scrypter
	if scrypt == nil {
		scrypt = &LazyScrypter{init: func() (postrs.Scrypter, error) {
			if options.providerID == nil {
				return nil, errors.New("no provider specified")
			}

			return postrs.NewScrypt(
				postrs.WithProviderID(*options.providerID),
				postrs.WithCommitment(options.commitment),
				postrs.WithScryptN(options.n),
				postrs.WithVRFDifficulty(options.vrfDifficulty),
				postrs.WithLogger(options.logger),
			)
		}}
	}

	return &WorkOracle{
		options: options,
		scrypt:  scrypt,
	}, nil
}

// Close the WorkOracle.
func (w *WorkOracle) Close() error {
	if w.scrypt == nil {
		return ErrWorkOracleClosed
	}
	if err := w.scrypt.Close(); err != nil && !errors.Is(err, postrs.ErrScryptClosed) {
		return fmt.Errorf("failed to close scrypt: %w", err)
	}
	w.scrypt = nil
	return nil
}

// WorkOracleResult is the result of a call to WorkOracle.
// It contains the computed labels and a nonce for a proof of work.
type WorkOracleResult struct {
	Output []byte  // Output are the computed labels
	Nonce  *uint64 // Nonce is the nonce of the proof of work
}

// Position computes the label for a given position.
func (w *WorkOracle) Position(p uint64) (WorkOracleResult, error) {
	return w.Positions(p, p)
}

// Positions computes the labels for a given range of positions.
func (w *WorkOracle) Positions(start, end uint64) (WorkOracleResult, error) {
	if w.scrypt == nil {
		return WorkOracleResult{}, ErrWorkOracleClosed
	}

	if start > end {
		return WorkOracleResult{}, fmt.Errorf("invalid `start` and `end`; expected: start <= end, given: %v > %v", start, end)
	}

	tries := 0
	for {
		res, err := w.scrypt.Positions(start, end)
		tries += 1
		switch {
		case errors.Is(err, postrs.ErrInitializationFailed):
			w.options.logger.With().Warn("failure during initialization", zap.Error(err))
			if tries > w.options.maxRetries {
				return WorkOracleResult{}, fmt.Errorf("failed to initialize scrypt after %v tries", tries)
			}
			w.options.logger.With().Warn("retrying initialization", zap.Int("tries", tries))
			time.Sleep(w.options.retryDelay)
		case err != nil:
			return WorkOracleResult{}, err
		default:
			return WorkOracleResult{
				Output: res.Output,
				Nonce:  res.IdxSolution,
			}, nil
		}
	}
}
