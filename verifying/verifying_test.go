package verifying

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
)

func getTestConfig(tb testing.TB) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()

	cfg.K1 = 3
	cfg.K2 = 3

	opts := config.DefaultInitOpts()
	opts.DataDir = tb.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = new(uint32)
	*opts.ProviderID = postrs.CPUProviderID()
	opts.ComputeBatchSize = 1 << 14
	return cfg, opts
}

func Test_Verify(t *testing.T) {
	r := require.New(t)

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)

	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
	cfg, opts := getTestConfig(t)
	init, err := initialization.NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithLogger(logger),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	proof, proofMetadata, err := proving.Generate(
		context.Background(),
		ch,
		cfg,
		logger,
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
		proving.LightMode(),
	)
	r.NoError(err)

	verifier, err := NewProofVerifier()
	r.NoError(err)
	defer verifier.Close()

	r.NoError(verifier.Verify(proof, proofMetadata, cfg, logger))
}

func Test_Verify_NoRace_On_Close(t *testing.T) {
	r := require.New(t)

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)

	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
	cfg, opts := getTestConfig(t)
	init, err := initialization.NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithLogger(logger),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	proof, proofMetadata, err := proving.Generate(
		context.Background(),
		ch,
		cfg,
		logger,
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
		proving.LightMode(),
	)
	r.NoError(err)

	verifier, err := NewProofVerifier()
	r.NoError(err)
	defer verifier.Close()

	var eg errgroup.Group
	eg.Go(func() error {
		time.Sleep(50 * time.Millisecond)
		return verifier.Close()
	})

	for i := 0; i < 10; i++ {
		ms := 10 * i
		eg.Go(func() error {
			time.Sleep(time.Duration(ms) * time.Millisecond)
			return verifier.Verify(proof, proofMetadata, cfg, logger)
		})
	}

	r.ErrorIs(eg.Wait(), postrs.ErrVerifierClosed)
}

func Test_Verifier_NoError_On_DoubleClose(t *testing.T) {
	verifier, err := NewProofVerifier()
	require.NoError(t, err)

	require.NoError(t, verifier.Close())
	require.NoError(t, verifier.Close())
}

func Test_Verify_Detects_invalid_proof(t *testing.T) {
	r := require.New(t)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)

	cfg, opts := getTestConfig(t)
	init, err := initialization.NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))
	proof, proofMetadata, err := proving.Generate(
		context.Background(),
		ch,
		cfg,
		logger,
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
		proving.LightMode(),
	)
	r.NoError(err)

	// modify one of proof.Indices by zeroing out some bits
	index := 1
	numLabels := proofMetadata.LabelsPerUnit * uint64(proofMetadata.NumUnits)
	bitsPerIndex := int(math.Log2(float64(numLabels))) + 1
	mask := byte(1<<bitsPerIndex - 1)
	offset := index * bitsPerIndex / 8
	proof.Indices[offset] &= ^(mask << (index * bitsPerIndex % 8))

	verifier, err := NewProofVerifier()
	r.NoError(err)
	defer verifier.Close()

	// Verify selected index (valid)
	err = verifier.Verify(proof, proofMetadata, cfg, logger, SelectedIndex(2))
	require.NoError(t, err)

	// Defaults to verifying all indices
	err = verifier.Verify(proof, proofMetadata, cfg, logger)
	expected := &ErrInvalidIndex{}
	r.ErrorAs(err, &expected)
	r.Equal(expected.Index, index)

	// Verify with AllIndices option
	err = verifier.Verify(proof, proofMetadata, cfg, logger, AllIndices())
	r.ErrorAs(err, &expected)
	r.Equal(expected.Index, index)

	// Verify only 1 index with K3 = 1, the `index` was empirically picked to pass verification
	err = verifier.Verify(proof, proofMetadata, cfg, logger, Subset(1, nodeId))
	require.NoError(t, err)

	// Verify selected index (invalid)
	err = verifier.Verify(proof, proofMetadata, cfg, logger, SelectedIndex(index))
	r.ErrorAs(err, &expected)
	r.Equal(expected.Index, index)
}

func TestVerifyPow(t *testing.T) {
	r := require.New(t)

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	cfg, opts := getTestConfig(t)
	opts.Scrypt.N = 16
	init, err := initialization.NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	m := &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		LabelsPerUnit:   cfg.LabelsPerUnit,
	}
	r.NoError(VerifyVRFNonce(init.Nonce(), m, WithLabelScryptParams(opts.Scrypt)))
}

func BenchmarkVerifying(b *testing.B) {
	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	cfg, opts := getTestConfig(b)

	init, err := initialization.NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
	)
	require.NoError(b, err)
	require.NoError(b, init.Initialize(context.Background()))

	ch := make(shared.Challenge, 32)
	rand.Read(ch)
	p, m, err := proving.Generate(
		context.Background(),
		ch, cfg,
		zaptest.NewLogger(b),
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
		proving.LightMode(),
	)
	require.NoError(b, err)

	verifier, err := NewProofVerifier()
	require.NoError(b, err)
	defer verifier.Close()

	for _, k3 := range []uint{5, 25, 50, 100} {
		testName := fmt.Sprintf("k3=%d", k3)

		b.Run(testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				start := time.Now()
				err := verifier.Verify(p, m, cfg, zaptest.NewLogger(b), Subset(k3, nodeId))
				require.NoError(b, err)
				b.ReportMetric(time.Since(start).Seconds(), "sec/proof")
			}
		})
	}
}
