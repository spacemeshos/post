package verifying

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
)

func getTestConfig(tb testing.TB) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()

	opts := config.DefaultInitOpts()
	opts.DataDir = tb.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ComputeProviderID = int(initialization.CPUProviderID())
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

func Test_Verify(t *testing.T) {
	r := require.New(t)
	log := testLogger{tb: t}

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
		log,
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
	)
	r.NoError(err)

	r.NoError(Verify(proof, proofMetadata, cfg))
}

func Test_Verify_Detects_invalid_proof(t *testing.T) {
	r := require.New(t)
	log := testLogger{tb: t}

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
		log,
		proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir),
	)
	r.NoError(err)

	for i := range proof.Indices {
		proof.Indices[i] ^= 255 // flip bits in all indices
	}

	r.ErrorContains(Verify(proof, proofMetadata, cfg), "invalid proof")
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
	p, m, err := proving.Generate(context.Background(), ch, cfg, &shared.DisabledLogger{}, proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
	require.NoError(b, err)

	for _, k3 := range []uint32{5, 25, 50, 100} {
		testName := fmt.Sprintf("k3=%d", k3)

		cfg.K3 = k3

		b.Run(testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				start := time.Now()
				err := Verify(p, m, cfg)
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
		proof, proofMetadata, err := proving.Generate(context.Background(), ch, cfg, &shared.DisabledLogger{}, proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
		r.NoError(err)

		b.StartTimer()
		start := time.Now()
		r.NoError(Verify(proof, proofMetadata, cfg))
		b.ReportMetric(time.Since(start).Seconds(), "sec/proof")
		b.StopTimer()
	}
}
