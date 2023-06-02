package postrs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTranslateScryptParams(t *testing.T) {
	n := uint(1 << (15 + 1))
	r := uint(1 << 5)
	p := uint(1 << 1)

	cParams := TranslateScryptParams(n, r, p)

	require.EqualValues(t, 15, cParams.nfactor)
	require.EqualValues(t, 5, cParams.rfactor)
	require.EqualValues(t, 1, cParams.pfactor)
}
