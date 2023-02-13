package proving

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/initialization"
)

func Benchmark_Proof(b *testing.B) {
	challenge := []byte("hello world, challenge me!!!!!!!")

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	cfg, opts := getTestConfig(b)
	init, err := NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
	)
	require.NoError(b, err)
	require.NoError(b, init.Initialize(context.Background()))

	b.ResetTimer()
	b.SetBytes(int64(cfg.LabelsPerUnit) * int64(opts.NumUnits))
	for i := 0; i < b.N; i++ {
		proof, meta, err := Generate(context.Background(), challenge, cfg, opts.DataDir, nodeId, commitmentAtxId, testLogger{tb: b})
		require.NoError(b, err)
		require.NotNil(b, proof)
		require.NotNil(b, meta)
	}
}
