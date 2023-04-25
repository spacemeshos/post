package initialization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenchmark(t *testing.T) {
	for _, p := range Providers() {
		b, err := Benchmark(p)
		require.NoError(t, err)
		require.Greater(t, b, 0)
	}
}
