package postrs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/internal/gpu"
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
	err := Initialize()
	require.NoError(t, err)

	commitment := make([]byte, 32)
	salt := make([]byte, 32)
	providers := gpu.Providers()

	res, err := gpu.ScryptPositions(
		gpu.WithComputeProviderID(providers[0].ID),
		gpu.WithCommitment(commitment),
		gpu.WithSalt(salt),
		gpu.WithStartAndEndPosition(1, 2),
		gpu.WithBitsPerLabel(128), // 16 byte labels
		gpu.WithScryptParams(8192, 1, 1),
	)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Output)
	fmt.Printf("res.Output: %x\n", res.Output)
}
