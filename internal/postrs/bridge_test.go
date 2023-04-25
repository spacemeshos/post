package postrs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
)

func TestTranslateScryptParams(t *testing.T) {
	params := config.ScryptParams{
		N: 1 << (15 + 1),
		R: 1 << 5,
		P: 1 << 1,
	}

	cParams := translateScryptParams(params)

	require.EqualValues(t, 15, cParams.nfactor)
	require.EqualValues(t, 5, cParams.rfactor)
	require.EqualValues(t, 1, cParams.pfactor)
}

func TestGetProviders(t *testing.T) {
	providers, err := cGetProviders()
	require.NoError(t, err)
	require.NotNil(t, providers)
}

func TestInitialize(t *testing.T) {
	t.Skip("skipping test - panics if no GPU is available")

	err := Initialize()
	require.NoError(t, err)
}
