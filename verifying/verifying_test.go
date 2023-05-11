package verifying

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
)

func getTestConfig(tb testing.TB) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()

	id := postrs.CPUProviderID()
	require.NotZero(tb, id)

	opts := config.DefaultInitOpts()
	opts.DataDir = tb.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(id)
	opts.ComputeBatchSize = 1 << 14
	return cfg, opts
}

func Test_Verify(t *testing.T) {
	r := require.New(t)

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
		zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)),
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
	)
	r.NoError(err)

	r.NoError(Verify(proof, proofMetadata, cfg, zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))))
}

func Test_Verify_Detects_invalid_proof(t *testing.T) {
	r := require.New(t)

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
		zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)),
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
	)
	r.NoError(err)

	for i := range proof.Indices {
		proof.Indices[i] ^= 255 // flip bits in all indices
	}

	r.ErrorContains(Verify(proof, proofMetadata, cfg, zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))), "invalid proof")
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
		LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
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
	p, m, err := proving.Generate(context.Background(), ch, cfg, zaptest.NewLogger(b), proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
	require.NoError(b, err)

	for _, k3 := range []uint32{5, 25, 50, 100} {
		testName := fmt.Sprintf("k3=%d", k3)

		cfg.K3 = k3

		b.Run(testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				start := time.Now()
				err := Verify(p, m, cfg, zaptest.NewLogger(b))
				require.NoError(b, err)
				b.ReportMetric(time.Since(start).Seconds(), "sec/proof")
			}
		})
	}
}

func Benchmark_Verify_Fastnet(b *testing.B) {
	r := require.New(b)
	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)

	cfg, opts := getTestConfig(b)
	cfg.K1 = 12
	cfg.K2 = 4
	cfg.K3 = 2
	cfg.LabelsPerUnit = 32
	cfg.MaxNumUnits = 4
	cfg.MinNumUnits = 2

	opts.NumUnits = cfg.MinNumUnits

	init, err := initialization.NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	for i := 0; i < b.N; i++ {
		rand.Read(ch)
		proof, proofMetadata, err := proving.Generate(context.Background(), ch, cfg, zaptest.NewLogger(b), proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
		r.NoError(err)

		b.StartTimer()
		start := time.Now()
		r.NoError(Verify(proof, proofMetadata, cfg, zaptest.NewLogger(b)))
		b.ReportMetric(time.Since(start).Seconds(), "sec/proof")
		b.StopTimer()
	}
}
