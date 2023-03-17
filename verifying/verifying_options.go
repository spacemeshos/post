package verifying

import (
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/shared"
)

type option struct {
	logger shared.Logger
	// scrypt parameters for AES PoW
	powScrypt config.ScryptParams
	// scrypt parameters for labels initialization
	labelScrypt config.ScryptParams
}

func defaultOpts() *option {
	return &option{
		logger:      &shared.DisabledLogger{},
		powScrypt:   config.DefaultPowScryptParams(),
		labelScrypt: config.DefaultLabelsScryptParams(),
	}
}

type OptionFunc func(*option) error

// WithLogger adds a logger to the verifier to log debug messages.
func WithLogger(logger shared.Logger) OptionFunc {
	return func(o *option) error {
		o.logger = logger
		return nil
	}
}

func WithLabelScryptParams(params config.ScryptParams) OptionFunc {
	return func(o *option) error {
		o.labelScrypt = params
		return nil
	}
}

func WithPowScryptParams(params config.ScryptParams) OptionFunc {
	return func(o *option) error {
		o.powScrypt = params
		return nil
	}
}
