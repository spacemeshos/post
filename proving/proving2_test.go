package proving

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

func BenchmarkProving(b *testing.B) {
	const MiB = uint64(1024 * 1024)
	const GiB = MiB * 1024
	const TiB = GiB * 1024

	startPos := 256 * GiB
	endPos := 4 * TiB

	for _, numNonces := range []uint32{6, 12, 24} {
		for numLabels := startPos; numLabels <= endPos; numLabels *= 4 {
			d := shared.CalcD(numLabels, config.DefaultAESBatchSize)
			testName := fmt.Sprintf("%.02fGiB/d=%d/Nonces=%d", float64(numLabels)/float64(GiB), d, numNonces)

			b.Run(testName, func(b *testing.B) {
				benchedDataSize := uint64(math.Min(float64(numLabels), float64(2*GiB)))
				benchmarkProving(b, numLabels, numNonces, benchedDataSize)
			})
		}
	}
}

func benchmarkProving(b *testing.B, numLabels uint64, numNonces uint32, benchedDataSize uint64) {
	challenge := []byte("hello world, challenge me!!!!!!!")

	// file := rand.New(rand.NewSource(0))
	file, err := os.Open("/dev/zero")
	require.NoError(b, err)
	defer file.Close()

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	cfg, _ := getTestConfig(b)
	cfg.LabelsPerUnit = numLabels
	cfg.N = numNonces

	b.SetBytes(int64(benchedDataSize))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := io.LimitReader(bufio.NewReader(file), int64(benchedDataSize))

		Generate(context.Background(), challenge, cfg, testLogger{tb: b}, withLabelsReader(reader, nodeId, commitmentAtxId, 1))
	}
}

func Test_Generate(t *testing.T) {
	r := require.New(t)
	log := testLogger{tb: t}

	for numUnits := uint32(config.DefaultMinNumUnits); numUnits < 6; numUnits++ {
		numUnits := numUnits
		t.Run(fmt.Sprintf("numUnits=%d", numUnits), func(t *testing.T) {
			t.Parallel()

			nodeId := make([]byte, 32)
			commitmentAtxId := make([]byte, 32)
			ch := make(Challenge, 32)
			cfg := config.DefaultConfig()
			cfg.LabelsPerUnit = 1 << 15

			opts := config.DefaultInitOpts()
			opts.ComputeProviderID = int(CPUProviderID())
			opts.NumUnits = numUnits
			opts.DataDir = t.TempDir()

			init, err := NewInitializer(
				initialization.WithNodeId(nodeId),
				initialization.WithCommitmentAtxId(commitmentAtxId),
				initialization.WithConfig(cfg),
				initialization.WithInitOpts(opts),
				initialization.WithLogger(log),
			)
			r.NoError(err)
			r.NoError(init.Initialize(context.Background()))

			n, err := rand.Read(ch)
			r.NoError(err)
			r.Equal(len(ch), n)

			proof, proofMetaData, err := Generate(context.Background(), ch, cfg, log, WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
			r.NoError(err, "numUnits: %d", opts.NumUnits)
			r.NotNil(proof)
			r.NotNil(proofMetaData)

			r.Equal(nodeId, proofMetaData.NodeId)
			r.Equal(commitmentAtxId, proofMetaData.CommitmentAtxId)
			r.Equal(ch, proofMetaData.Challenge)
			r.Equal(cfg.BitsPerLabel, proofMetaData.BitsPerLabel)
			r.Equal(cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
			r.Equal(opts.NumUnits, proofMetaData.NumUnits)
			r.Equal(cfg.K1, proofMetaData.K1)
			r.Equal(cfg.K2, proofMetaData.K2)

			numLabels := cfg.LabelsPerUnit * uint64(numUnits)
			indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
			r.Equal(shared.Size(indexBitSize, uint(cfg.K2)), uint(len(proof.Indices)))

			log.Info("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
			r.NoError(verifying.VerifyNew(proof, proofMetaData, verifying.WithLogger(log)))
		})
	}
}
