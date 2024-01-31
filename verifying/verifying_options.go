package verifying

import (
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/internal/postrs"
)

type option struct {
	powFlags config.PowFlags
	// scrypt parameters for labels initialization
	labelScrypt config.ScryptParams

	internalOpts []postrs.VerifyOptionFunc
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

// Verify all indices in the proof.
func AllIndices() OptionFunc {
	return func(o *option) {
		o.internalOpts = append(o.internalOpts, postrs.VerifyAll())
	}
}

// Verify a subset of randomly selected K3 indices.
// The `id` is used to seed the random number generator.
func Subset(k3 uint, seed []byte) OptionFunc {
	return func(o *option) {
		o.internalOpts = append(o.internalOpts, postrs.VerifySubset(k3, seed))
	}
}

// Verify only the selected index.
// The `ord` is the ordinal number of the index in the proof to verify.
func SelectedIndex(ord int) OptionFunc {
	return func(o *option) {
		o.internalOpts = append(o.internalOpts, postrs.VerifyOne(ord))
	}
}
