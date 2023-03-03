package proving

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

func getTestConfig(tb testing.TB) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 4096

	opts := config.DefaultInitOpts()
	opts.DataDir = tb.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ComputeProviderID = int(initialization.CPUProviderID())

	return cfg, opts
}

type testLogger struct {
	shared.Logger

	tb testing.TB
}

func (l testLogger) Info(msg string, args ...any)  { l.tb.Logf("\tINFO\t"+msg, args...) }
func (l testLogger) Debug(msg string, args ...any) { l.tb.Logf("\tDEBUG\t"+msg, args...) }
func (l testLogger) Error(msg string, args ...any) { l.tb.Logf("\tERROR\t"+msg, args...) }

func Test_Generate(t *testing.T) {
	for numUnits := uint32(config.DefaultMinNumUnits); numUnits < 6; numUnits++ {
		numUnits := numUnits
		t.Run(fmt.Sprintf("numUnits=%d", numUnits), func(t *testing.T) {
			t.Parallel()
			r := require.New(t)
			log := testLogger{tb: t}

			nodeId := make([]byte, 32)
			commitmentAtxId := make([]byte, 32)
			ch := make(shared.Challenge, 32)

			cfg, opts := getTestConfig(t)
			opts.NumUnits = numUnits

			init, err := initialization.NewInitializer(
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
			r.Equal(cfg.B, proofMetaData.B)
			r.Equal(cfg.N, proofMetaData.N)

			numLabels := cfg.LabelsPerUnit * uint64(numUnits)
			indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
			r.Equal(shared.Size(indexBitSize, uint(cfg.K2)), uint(len(proof.Indices)))

			log.Info("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
			r.NoError(verifying.Verify(proof, proofMetaData, verifying.WithLogger(log)))
		})
	}
}

func Test_Generate_DetectInvalidParameters(t *testing.T) {
	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	ch := make(shared.Challenge, 32)
	cfg, opts := getTestConfig(t)
	init, err := initialization.NewInitializer(
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

func Test_Generate_TestNetSettings(t *testing.T) {
	r := require.New(t)
	log := testLogger{tb: t}

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)
	cfg := config.DefaultConfig()

	// https://colab.research.google.com/github/spacemeshos/notebooks/blob/main/post-proof-params.ipynb
	cfg.LabelsPerUnit = 2 << 16
	cfg.B = 16
	cfg.K1 = 279
	cfg.K2 = 287
	cfg.N = 24

	opts := config.DefaultInitOpts()
	opts.ComputeProviderID = int(initialization.CPUProviderID())
	opts.NumUnits = 2
	opts.DataDir = t.TempDir()

	init, err := initialization.NewInitializer(
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

	numLabels := cfg.LabelsPerUnit * uint64(opts.NumUnits)
	indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
	r.Equal(shared.Size(indexBitSize, uint(cfg.K2)), uint(len(proof.Indices)))

	log.Info("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
	r.NoError(verifying.Verify(proof, proofMetaData, verifying.WithLogger(log)))
}
