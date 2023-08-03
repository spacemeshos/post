package proof

import "context"

type Verifier struct{}

func NewVerifier(opts ...VerifyOptionFunc) *Verifier {
	return nil
}

func (v *Verifier) Verify(ctx context.Context, challenge, proof []byte) (bool, error) {
	return false, nil
}
