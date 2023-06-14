package verifying

import (
	"github.com/spacemeshos/post/config"
)

type option struct {
	powFlags config.PowFlags
	// scrypt parameters for AES PoW
	powScrypt config.ScryptParams
	// scrypt parameters for labels initialization
	labelScrypt config.ScryptParams
}

func defaultOpts() *option {
	return &option{
		powFlags:    config.DefaultVerifyingPowFlags(),
		powScrypt:   config.DefaultPowParams(),
		labelScrypt: config.DefaultLabelParams(),
	}
}

func applyOpts(options ...OptionFunc) (*option, error) {
	opts := defaultOpts()
	for _, opt := range options {
		if err := opt(opts); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

type OptionFunc func(*option) error

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

func WithPowFlags(flags config.PowFlags) OptionFunc {
	return func(o *option) error {
		o.powFlags = flags
		return nil
	}
}
