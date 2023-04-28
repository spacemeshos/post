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

const KiB = 1024

func getTestConfig(tb testing.TB) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()

	id, err := initialization.CPUProviderID()
	require.NoError(tb, err)

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16 // speed up initialization
	opts.DataDir = tb.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ComputeProviderID = int(id)
	opts.ComputeBatchSize = 1 << 14
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
	for numUnits := uint32(1); numUnits < 6; numUnits++ {
		numUnits := numUnits
		t.Run(fmt.Sprintf("numUnits=%d", numUnits), func(t *testing.T) {
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

			proof, proofMetaData, err := Generate(
				context.Background(),
				ch,
				cfg,
				log,
				WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
				WithThreads(2),
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

			log.Info("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
			r.NoError(verifying.Verify(
				proof,
				proofMetaData,
				cfg,
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

	// Test-net settings:
	cfg.LabelsPerUnit = 20 * KiB / 16 // 20kB unit
	cfg.K1 = 273
	cfg.K2 = 300
	cfg.K3 = 100

	id, err := initialization.CPUProviderID()
	require.NoError(t, err)

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.ComputeProviderID = int(id)
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
	r.Equal(cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
	r.Equal(opts.NumUnits, proofMetaData.NumUnits)

	numLabels := cfg.LabelsPerUnit * uint64(opts.NumUnits)
	indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
	r.Equal(shared.Size(indexBitSize, uint(cfg.K2)), uint(len(proof.Indices)))

	log.Info("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
	r.NoError(verifying.Verify(
		proof,
		proofMetaData,
		cfg,
		verifying.WithLabelScryptParams(opts.Scrypt)),
	)
}
