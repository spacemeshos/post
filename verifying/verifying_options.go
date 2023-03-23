package verifying

import (
	"github.com/spacemeshos/post/config"
)

type option struct {
	// scrypt parameters for AES PoW
	powScrypt config.ScryptParams
	// scrypt parameters for labels initialization
	labelScrypt config.ScryptParams
}

func defaultOpts() *option {
	return &option{
		powScrypt:   config.DefaultPowParams(),
		labelScrypt: config.DefaultLabelParams(),
	}
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
