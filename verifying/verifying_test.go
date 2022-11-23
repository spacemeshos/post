package verifying

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
)

var (
	commitment = make([]byte, 32)
	ch         = make(proving.Challenge, 32)

	NewInitializer = initialization.NewInitializer
	NewProver      = proving.NewProver
	CPUProviderID  = initialization.CPUProviderID
)

func getTestConfig(t *testing.T) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	return cfg, opts
}

func TestVerify(t *testing.T) {
	req := require.New(t)

	cfg, opts := getTestConfig(t)
	init, err := NewInitializer(
		initialization.WithCommitment(commitment),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
	)
	req.NoError(err)
	req.NoError(init.Initialize(context.Background()))

	p, err := NewProver(cfg, opts.DataDir, commitment)
	req.NoError(err)
	proof, proofMetadata, err := p.GenerateProof(ch)
	req.NoError(err)

	req.NoError(Verify(proof, proofMetadata))
}

// TestLabelsCorrectness tests, for variation of label sizes, the correctness of
// reading labels from disk (written in multiple files) when compared to a single label compute.
// It is covers the following components: labels compute lib (package: oracle), labels writer (package: persistence),
// labels reader (package: persistence), and the granularity-specific reader (package: shared).
// it proceeds as follows:
//  1. Compute labels, in batches, and write them into multiple files (prover).
//  2. Read the sequence of labels from the files according to the specified label size (prover),
//     and ensure that each one equals a single label compute (verifier).
func TestLabelsCorrectness(t *testing.T) {
	req := require.New(t)
	if testing.Short() {
		t.Skip("long test")
	}

	numFiles := 2
	numFileBatches := 2
	batchSize := 256
	commitment := make([]byte, 32)
	datadir := t.TempDir()

	for bitsPerLabel := uint32(config.MinBitsPerLabel); bitsPerLabel <= config.MaxBitsPerLabel; bitsPerLabel++ {
		t.Logf("bitsPerLabel: %v\n", bitsPerLabel)

		// Write.
		for i := 0; i < numFiles; i++ {
			writer, err := persistence.NewLabelsWriter(datadir, i, uint(bitsPerLabel))
			req.NoError(err)
			for j := 0; j < numFileBatches; j++ {
				numBatch := i*numFileBatches + j
				startPosition := uint64(numBatch * batchSize)
				endPosition := startPosition + uint64(batchSize) - 1

				res, err := oracle.WorkOracle(
					oracle.WithComputeProviderID(CPUProviderID()),
					oracle.WithCommitment(commitment),
					oracle.WithStartAndEndPosition(startPosition, endPosition),
					oracle.WithBitsPerLabel(bitsPerLabel),
				)
				req.NoError(err)
				req.NoError(writer.Write(res.Output))
			}
			_, err = writer.Close()
			req.NoError(err)
		}

		// Read.
		reader, err := persistence.NewLabelsReader(datadir, uint(bitsPerLabel))
		req.NoError(err)
		defer reader.Close()
		gsReader := shared.NewGranSpecificReader(reader, uint(bitsPerLabel))
		var position uint64
		for {
			label, err := gsReader.ReadNext()
			if err != nil {
				if err == io.EOF {
					req.Equal(uint64(numFiles*numFileBatches*batchSize), position)
					break
				}
				req.Fail(err.Error())
			}

			// Verify correctness.
			labelCompute, err := oracle.WorkOracle(
				oracle.WithCommitment(commitment),
				oracle.WithPosition(position),
				oracle.WithBitsPerLabel(bitsPerLabel),
			)
			req.NoError(err)
			req.Equal(labelCompute, label, fmt.Sprintf("position: %v, bitsPerLabel: %v", position, bitsPerLabel))

			position++
		}
	}
}
