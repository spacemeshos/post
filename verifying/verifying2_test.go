package verifying

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/shared"
)

func BenchmarkVerifying(b *testing.B) {
	for _, k2 := range []uint32{300, 500, 800, 1800} {
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
