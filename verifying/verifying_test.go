package verifying

import (
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

var (
	id = make([]byte, 32)
	ch = make(proving.Challenge, 32)

	cfg  config.Config
	opts config.InitOpts

	debug = flag.Bool("debug", false, "")

	NewInitializer = initialization.NewInitializer
	NewProver      = proving.NewProver
	CPUProviderID  = initialization.CPUProviderID
)

func TestMain(m *testing.M) {
	cfg = config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts = config.DefaultInitOpts()
	opts.DataDir, _ = ioutil.TempDir("", "post-test")
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = CPUProviderID()

	res := m.Run()
	os.Exit(res)
}

func TestVerify(t *testing.T) {
	req := require.New(t)

	init, err := NewInitializer(cfg, opts, id)
	req.NoError(err)
	err = init.Initialize()
	req.NoError(err)

	p, err := NewProver(cfg, opts.DataDir, id)
	req.NoError(err)
	proof, proofMetadata, err := p.GenerateProof(ch)
	req.NoError(err)

	err = Verify(proof, proofMetadata)
	req.NoError(err)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

// TestLabelsCorrectness tests, for variation of label sizes, the correctness of
// reading labels from disk (written in multiple files) when compared to a single label compute.
// It is covers the following components: labels compute lib (package: oracle), labels writer (package: persistence),
// labels reader (package: persistence), and the granularity-specific reader (package: shared).
// it proceeds as follows:
// 1. Compute labels, in batches, and write them into multiple files (prover).
// 2. Read the sequence of labels from the files according to the specified label size (prover),
//    and ensure that each one equals a single label compute (verifier).
func TestLabelsCorrectness(t *testing.T) {
	req := require.New(t)
	if testing.Short() {
		t.Skip()
	}

	numFiles := 2
	numFileBatches := 2
	batchSize := 256
	id := make([]byte, 32)
	datadir, _ := ioutil.TempDir("", "post-test")

	for bitsPerLabel := uint32(config.MinBitsPerLabel); bitsPerLabel <= config.MaxBitsPerLabel; bitsPerLabel++ {
		if *debug {
			fmt.Printf("bitsPerLabel: %v\n", bitsPerLabel)
		}

		// Write.
		for i := 0; i < numFiles; i++ {
			writer, err := persistence.NewLabelsWriter(datadir, i, uint(bitsPerLabel))
			req.NoError(err)
			for j := 0; j < numFileBatches; j++ {
				numBatch := i*numFileBatches + j
				startPosition := uint64(numBatch * batchSize)
				endPosition := startPosition + uint64(batchSize) - 1

				labels, err := oracle.WorkOracle(uint(CPUProviderID()), id, startPosition, endPosition, bitsPerLabel)
				req.NoError(err)
				err = writer.Write(labels)
				req.NoError(err)

			}
			_, err = writer.Close()
			req.NoError(err)
		}

		// Read.
		reader, err := persistence.NewLabelsReader(datadir, uint(bitsPerLabel))
		gsReader := shared.NewGranSpecificReader(reader, uint(bitsPerLabel))
		req.NoError(err)
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
			labelCompute := oracle.WorkOracleOne(uint(CPUProviderID()), id, position, bitsPerLabel)
			req.Equal(labelCompute, label, fmt.Sprintf("position: %v, bitsPerLabel: %v", position, bitsPerLabel))

			position++
		}

		// Cleanup.
		_ = os.RemoveAll(datadir)
	}
}
