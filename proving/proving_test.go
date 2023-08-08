package proving

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

const KiB = 1024

func getTestConfig(tb testing.TB) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16 // speed up initialization
	opts.DataDir = tb.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = new(uint32)
	*opts.ProviderID = postrs.CPUProviderID()
	opts.ComputeBatchSize = 1 << 14
	return cfg, opts
}

func Test_Generate(t *testing.T) {
	for numUnits := uint32(1); numUnits < 6; numUnits++ {
		numUnits := numUnits
		t.Run(fmt.Sprintf("numUnits=%d", numUnits), func(t *testing.T) {
			r := require.New(t)
			log := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))

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

			proof, proofMetaData, err := Generate(
				context.Background(),
				ch,
				cfg,
				log,
				WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
				WithThreads(2),
				WithPowFlags(postrs.GetRecommendedPowFlags()),
			)
			r.NoError(err, "numUnits: %d", opts.NumUnits)
			r.NotNil(proof)
			r.NotNil(proofMetaData)

			r.Equal(nodeId, proofMetaData.NodeId)
			r.Equal(commitmentAtxId, proofMetaData.CommitmentAtxId)
			r.Equal(ch, proofMetaData.Challenge)
			r.Equal(cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
			r.Equal(opts.NumUnits, proofMetaData.NumUnits)

			numLabels := cfg.LabelsPerUnit * uint64(numUnits)
			indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
			r.Equal(shared.Size(indexBitSize, uint(cfg.K2)), uint(len(proof.Indices)))

			log.Info("post status",
				zap.Uint64("numLabels", numLabels),
				zap.Int("indices size", len(proof.Indices)),
			)
			verifier, err := verifying.NewProofVerifier()
			r.NoError(err)
			defer verifier.Close()
			r.NoError(verifier.Verify(
				proof,
				proofMetaData,
				cfg,
				zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)),
				verifying.WithLabelScryptParams(opts.Scrypt)),
			)
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
		initialization.WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	require.NoError(t, err)
	require.NoError(t, init.Initialize(context.Background()))

	t.Run("invalid nodeId", func(t *testing.T) {
		newNodeId := make([]byte, 32)
		copy(newNodeId, nodeId)
		newNodeId[0] = newNodeId[0] + 1

		_, _, err := Generate(
			context.Background(),
			ch,
			cfg,
			zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)),
			WithDataSource(cfg, newNodeId, commitmentAtxId, opts.DataDir),
			WithPowFlags(postrs.GetRecommendedPowFlags()),
		)
		var errConfigMismatch initialization.ConfigMismatchError
		require.ErrorAs(t, err, &errConfigMismatch)
		require.Equal(t, "NodeId", errConfigMismatch.Param)
	})

	t.Run("invalid atxId", func(t *testing.T) {
		newAtxId := make([]byte, 32)
		copy(newAtxId, commitmentAtxId)
		newAtxId[0] = newAtxId[0] + 1

		_, _, err := Generate(
			context.Background(),
			ch,
			cfg,
			zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)),
			WithDataSource(cfg, nodeId, newAtxId, opts.DataDir),
			WithPowFlags(postrs.GetRecommendedPowFlags()),
		)
		var errConfigMismatch initialization.ConfigMismatchError
		require.ErrorAs(t, err, &errConfigMismatch)
		require.Equal(t, "CommitmentAtxId", errConfigMismatch.Param)
	})

	t.Run("invalid LabelsPerUnit", func(t *testing.T) {
		newCfg := cfg
		newCfg.LabelsPerUnit++

		_, _, err := Generate(
			context.Background(),
			ch,
			newCfg,
			zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)),
			WithDataSource(newCfg, nodeId, commitmentAtxId, opts.DataDir),
			WithPowFlags(postrs.GetRecommendedPowFlags()),
		)
		var errConfigMismatch initialization.ConfigMismatchError
		require.ErrorAs(t, err, &errConfigMismatch)
		require.Equal(t, "LabelsPerUnit", errConfigMismatch.Param)
	})
}

func Test_Generate_TestNetSettings(t *testing.T) {
	r := require.New(t)
	log := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)
	cfg := config.DefaultConfig()

	// Test-net settings:
	cfg.LabelsPerUnit = 20 * KiB / postrs.LabelLength // 20kB unit
	cfg.K1 = 273
	cfg.K2 = 300
	cfg.K3 = 100

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.ProviderID = new(uint32)
	*opts.ProviderID = postrs.CPUProviderID()
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

	proof, proofMetaData, err := Generate(
		context.Background(),
		ch,
		cfg,
		log,
		WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
		WithPowFlags(postrs.GetRecommendedPowFlags()),
	)
	r.NoError(err, "numUnits: %d", opts.NumUnits)
	r.NotNil(proof)
	r.NotNil(proofMetaData)

	r.Equal(nodeId, proofMetaData.NodeId)
	r.Equal(commitmentAtxId, proofMetaData.CommitmentAtxId)
	r.Equal(ch, proofMetaData.Challenge)
	r.Equal(cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
	r.Equal(opts.NumUnits, proofMetaData.NumUnits)

	numLabels := cfg.LabelsPerUnit * uint64(opts.NumUnits)
	indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
	r.Equal(shared.Size(indexBitSize, uint(cfg.K2)), uint(len(proof.Indices)))

	log.Info("post status",
		zap.Uint64("numLabels", numLabels),
		zap.Int("indices size", len(proof.Indices)),
	)
	verifier, err := verifying.NewProofVerifier()
	r.NoError(err)
	defer verifier.Close()
	r.NoError(verifier.Verify(
		proof,
		proofMetaData,
		cfg,
		zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)),
		verifying.WithLabelScryptParams(opts.Scrypt)),
	)
}
