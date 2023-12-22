package verifying

import (
	"github.com/spacemeshos/post/config"
)

type option struct {
	powFlags config.PowFlags
	// scrypt parameters for labels initialization
	labelScrypt config.ScryptParams
	// whether to verify all indices
	allIndices bool
}

func applyOpts(options ...OptionFunc) *option {
	opts := &option{
		powFlags:    config.DefaultVerifyingPowFlags(),
		labelScrypt: config.DefaultLabelParams(),
	}
	for _, opt := range options {
		opt(opts)
	}
	return opts
}

type OptionFunc func(*option)

func WithLabelScryptParams(params config.ScryptParams) OptionFunc {
	return func(o *option) {
		o.labelScrypt = params
	}
}

func WithPowFlags(flags config.PowFlags) OptionFunc {
	return func(o *option) {
		o.powFlags = flags
	}
}

func AllIndices() OptionFunc {
	return func(o *option) {
		o.allIndices = true
	}
}
