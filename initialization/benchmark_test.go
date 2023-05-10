package initialization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenchmark(t *testing.T) {
	providers, err := OpenCLProviders()
	require.NoError(t, err)

	for _, p := range providers {
		hashes, err := Benchmark(p)
		require.NoError(t, err)
		require.Greater(t, hashes, 0)
	}
}
