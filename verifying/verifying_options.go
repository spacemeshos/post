package verifying

import (
	"github.com/spacemeshos/post/config"
)

type option struct {
	// scrypt parameters for labels initialization
	labelScrypt config.ScryptParams
}

func defaultOpts() *option {
	return &option{
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
