package proving

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
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
		proof, meta, err := Generate(context.Background(), challenge, cfg, testLogger{tb: b}, WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
		require.NoError(b, err)
		require.NotNil(b, proof)
		require.NotNil(b, meta)
	}
}

func BenchmarkProving(b *testing.B) {
	const MiB = uint64(1024 * 1024)
	const GiB = MiB * 1024
	const TiB = GiB * 1024
	const PiB = TiB * 1024

	startPos := 256 * MiB
	endPos := PiB

	for numLabels := startPos; numLabels <= endPos; numLabels *= 2 {
		d := oracle.CalcD(numLabels, config.DefaultAESBatchSize)
		testName := fmt.Sprintf("%.02fGiB/d=%d", float64(numLabels)/float64(GiB), d)

		b.Run(testName, func(b *testing.B) {
			benchedDataSize := uint64(math.Min(float64(numLabels), float64(2*GiB)))
			benchmarkProving(b, numLabels, benchedDataSize)
		})
	}
}

func benchmarkProving(b *testing.B, numLabels, benchedDataSize uint64) {
	challenge := []byte("hello world, challenge me!!!!!!!")

	// file := rand.New(rand.NewSource(0))
	file, err := os.Open("/dev/zero")
	require.NoError(b, err)
	defer file.Close()

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	cfg, _ := getTestConfig(b)
	cfg.LabelsPerUnit = numLabels

	b.SetBytes(int64(benchedDataSize))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := io.LimitReader(bufio.NewReader(file), int64(benchedDataSize))

		Generate(context.Background(), challenge, cfg, testLogger{tb: b}, withLabelsReader(reader, nodeId, commitmentAtxId, 1))
	}
}
