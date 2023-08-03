package proof

type verifyOption struct{}

type VerifyOptionFunc func(*verifyOption) error
