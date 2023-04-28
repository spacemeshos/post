package initialization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenchmark(t *testing.T) {
	providers, err := OpenCLProviders()
	require.NoError(t, err)

	for _, p := range providers {
		b, err := Benchmark(p)
		require.NoError(t, err)
		require.Greater(t, b, 0)
	}
}
