package verifying

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
)

type testLogger struct {
	shared.Logger

	tb testing.TB
}

func (l testLogger) Info(msg string, args ...any)  { l.tb.Logf("\tINFO\t"+msg, args...) }
func (l testLogger) Debug(msg string, args ...any) { l.tb.Logf("\tDEBUG\t"+msg, args...) }
func (l testLogger) Error(msg string, args ...any) { l.tb.Logf("\tERROR\t"+msg, args...) }

func BenchmarkVerifying(b *testing.B) {
	for _, mB := range []uint32{8, 16} {
		for _, k2 := range []uint32{170, 288, 500, 800} {
			testName := fmt.Sprintf("256GiB/B=%d/k2=%d", mB, k2)

			b.Run(testName, func(b *testing.B) {
				benchmarkVerifying(b, mB, k2)
			})
		}
	}
}

func benchmarkVerifying(b *testing.B, mB, k2 uint32) {
	cfg, opts := getTestConfig(b)
	m := &shared.ProofMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		Challenge:       []byte("hello world, challenge me!!!!!!!"),
		NumUnits:        opts.NumUnits,
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   256 * 1024 * 1024 * 1024, // 256GiB
		K1:              cfg.K1,
		K2:              k2,
		B:               mB,
		N:               cfg.N,
	}

	numLabels := uint64(m.NumUnits) * uint64(m.LabelsPerUnit)
	bitsPerIndex := uint(shared.BinaryRepresentationMinBits(numLabels))

	var buf bytes.Buffer
	gsWriter := shared.NewGranSpecificWriter(&buf, bitsPerIndex)
	for i := uint32(0); i < m.K2; i++ {
		require.NoError(b, gsWriter.WriteUintBE(uint64(i)))
	}
	require.NoError(b, gsWriter.Flush())

	p := &shared.Proof{
		Nonce:   rand.Uint32(),
		Indices: buf.Bytes(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		err := VerifyNew(p, m, withVerifyFunc(func(val uint64) bool { return true }))
		require.NoError(b, err)
		b.ReportMetric(time.Since(start).Seconds(), "sec/proof")
	}
}

func Test_VerifyNew(t *testing.T) {
	r := require.New(t)
	log := testLogger{tb: t}

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)

	cfg, opts := getTestConfig(t)
	init, err := NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	proof, proofMetadata, err := proving.Generate(context.Background(), ch, cfg, log, proving.WithDataSource(cfg, nodeId, commitmentAtxId, opts.DataDir))
	r.NoError(err)

	r.NoError(VerifyNew(proof, proofMetadata))
}

func Benchmark_VerifyNew_Fastnet(b *testing.B) {
	r := require.New(b)

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)
	ch := make(shared.Challenge, 32)

	cfg, opts := getTestConfig(b)
	cfg.BitsPerLabel = 8
	cfg.K1 = 12
	cfg.K2 = 4
	cfg.LabelsPerUnit = 32 // bytes
	cfg.MaxNumUnits = 4
	cfg.MinNumUnits = 2
	cfg.N = 32
	cfg.B = 2

	opts.NumUnits = cfg.MinNumUnits

	init, err := NewInitializer(
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
		r.NoError(VerifyNew(proof, proofMetadata))
		b.ReportMetric(time.Since(start).Seconds(), "sec/proof")
		b.StopTimer()
	}
}
