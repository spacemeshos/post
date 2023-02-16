package verifying

type option struct {
	verifyFunc func(val uint64) bool
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
