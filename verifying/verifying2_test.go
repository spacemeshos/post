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
	for _, k2 := range []uint32{170, 288, 500, 800} {
		testName := fmt.Sprintf("256GiB/k2=%d", k2)

		b.Run(testName, func(b *testing.B) {
			benchmarkVerifying(b, k2)
		})
	}
}

func benchmarkVerifying(b *testing.B, k2 uint32) {
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
		B:               cfg.B,
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
