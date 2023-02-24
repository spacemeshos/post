package verifying

import "github.com/spacemeshos/post/shared"

type option struct {
	verifyFunc func(val uint64) bool

	logger shared.Logger
}

func (o *option) validate() error {
	return nil
}

type OptionFunc func(*option) error

// withVerifyFunc sets a custom verify function. This is provided for testing purposes, and should not be used in production.
func withVerifyFunc(f func(val uint64) bool) OptionFunc {
	return func(o *option) error {
		o.verifyFunc = f
		return nil
	}
}

// WithLogger adds a logger to the verifier to log debug messages.
func WithLogger(logger shared.Logger) OptionFunc {
	return func(o *option) error {
		o.logger = logger
		return nil
	}
}
