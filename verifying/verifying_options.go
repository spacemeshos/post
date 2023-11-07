package verifying

import (
	"errors"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/shared"
)

type option struct {
	powFlags config.PowFlags
	// scrypt parameters for labels initialization
	labelScrypt shared.ScryptParams

	powCreatorId []byte
}

func defaultOpts() *option {
	return &option{
		powFlags:    config.DefaultVerifyingPowFlags(),
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

func WithLabelScryptParams(params shared.ScryptParams) OptionFunc {
	return func(o *option) error {
		o.labelScrypt = params
		return nil
	}
}

func WithPowFlags(flags config.PowFlags) OptionFunc {
	return func(o *option) error {
		o.powFlags = flags
		return nil
	}
}

func WithPowCreator(id []byte) OptionFunc {
	return func(o *option) error {
		if len(id) != 32 {
			return errors.New("pow creator id must be 32 bytes")
		}
		o.powCreatorId = id
		return nil
	}
}
