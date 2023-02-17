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

	for _, numNonces := range []uint32{6, 12, 20} {
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
	for numUnits := uint32(config.DefaultMinNumUnits); numUnits < 6; numUnits++ {
		numUnits := numUnits
		t.Run(fmt.Sprintf("numUnits=%d", numUnits), func(t *testing.T) {
			log := testLogger{tb: t}

			nodeId := make([]byte, 32)
			commitmentAtxId := make([]byte, 32)
			ch := make(Challenge, 32)
			cfg := config.DefaultConfig()

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
			require.NoError(t, err)
			require.NoError(t, init.Initialize(context.Background()))

			n, err := rand.Read(ch)
			require.NoError(t, err)
			require.Equal(t, len(ch), n)

			proof, proofMetaData, err := Generate(context.Background(), ch, cfg, log, WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
			require.NoError(t, err, "numUnits: %d", opts.NumUnits)
			require.NotNil(t, proof)
			require.NotNil(t, proofMetaData)

			require.Equal(t, nodeId, proofMetaData.NodeId)
			require.Equal(t, commitmentAtxId, proofMetaData.CommitmentAtxId)
			require.Equal(t, ch, proofMetaData.Challenge)
			require.Equal(t, cfg.BitsPerLabel, proofMetaData.BitsPerLabel)
			require.Equal(t, cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
			require.Equal(t, opts.NumUnits, proofMetaData.NumUnits)
			require.Equal(t, cfg.K1, proofMetaData.K1)
			require.Equal(t, cfg.K2, proofMetaData.K2)

			numLabels := cfg.LabelsPerUnit * uint64(numUnits)
			indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
			require.Equal(t, shared.Size(indexBitSize, uint(cfg.K2)), uint(len(proof.Indices)))

			log.Info("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
			require.NoError(t, verifying.VerifyNew(proof, proofMetaData, verifying.WithLogger(log)))
		})
	}
}

func Test_Generate_DetectInvalidParameters(t *testing.T) {
	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	ch := make(Challenge, 32)
	cfg, opts := getTestConfig(t)
	init, err := NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithLogger(testLogger{tb: t}),
	)
	require.NoError(t, err)
	require.NoError(t, init.Initialize(context.Background()))

	t.Run("invalid nodeId", func(t *testing.T) {
		log := testLogger{tb: t}

		newNodeId := make([]byte, 32)
		copy(newNodeId, nodeId)
		newNodeId[0] = newNodeId[0] + 1

		_, _, err := Generate(context.Background(), ch, cfg, log, WithDataSource(cfg, newNodeId, commitmentAtxId, opts.DataDir))
		var errConfigMismatch initialization.ConfigMismatchError
		require.ErrorAs(t, err, &errConfigMismatch)
		require.Equal(t, "NodeId", errConfigMismatch.Param)
	})

	t.Run("invalid atxId", func(t *testing.T) {
		log := testLogger{tb: t}

		newAtxId := make([]byte, 32)
		copy(newAtxId, commitmentAtxId)
		newAtxId[0] = newAtxId[0] + 1

		_, _, err := Generate(context.Background(), ch, cfg, log, WithDataSource(cfg, nodeId, newAtxId, opts.DataDir))
		var errConfigMismatch initialization.ConfigMismatchError
		require.ErrorAs(t, err, &errConfigMismatch)
		require.Equal(t, "CommitmentAtxId", errConfigMismatch.Param)
	})

	t.Run("invalid BitsPerLabel", func(t *testing.T) {
		log := testLogger{tb: t}

		newCfg := cfg
		newCfg.BitsPerLabel++

		_, _, err := Generate(context.Background(), ch, newCfg, log, WithDataSource(newCfg, nodeId, commitmentAtxId, opts.DataDir))
		var errConfigMismatch initialization.ConfigMismatchError
		require.ErrorAs(t, err, &errConfigMismatch)
		require.Equal(t, "BitsPerLabel", errConfigMismatch.Param)
	})

	t.Run("invalid LabelsPerUnit", func(t *testing.T) {
		log := testLogger{tb: t}

		newCfg := cfg
		newCfg.LabelsPerUnit++

		_, _, err := Generate(context.Background(), ch, newCfg, log, WithDataSource(newCfg, nodeId, commitmentAtxId, opts.DataDir))
		var errConfigMismatch initialization.ConfigMismatchError
		require.ErrorAs(t, err, &errConfigMismatch)
		require.Equal(t, "LabelsPerUnit", errConfigMismatch.Param)
	})
}
