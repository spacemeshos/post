package proving

import (
	"bytes"
	"context"
	"crypto/aes"
	"errors"
	"fmt"
	"math"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

// TODO (mafa): first two could be configuration options.
const (
	NumWorkers = 1 // Number of workers creating a proof in parallel. Each one will max out one CPU core.

	BlocksPerWorker = 1 << 24 // How many AES blocks are contained per batch sent to a worker. Larger values will increase memory usage, but speed up the proof generation.
	batchSize       = BlocksPerWorker * aes.BlockSize
)

// TODO (mafa): use functional options.
// TODO (mafa): replace Logger with zap.
// TODO (mafa): replace datadir with functional option for data provider. `verifyMetadata` and `initCompleted` should be part of the `WithDataDir` option.
func Generate(ctx context.Context, ch Challenge, cfg Config, logger Logger, opts ...OptionFunc) (*Proof, *ProofMetadata, error) {
	options := &option{}
	for _, opt := range opts {
		opt(options)
	}
	if err := options.validate(); err != nil {
		return nil, nil, err
	}

	batchChan := make(chan *batch)
	solutionChan := make(chan *solution)

	numLabels := uint64(options.numUnits) * uint64(cfg.LabelsPerUnit)

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, egCtx := errgroup.WithContext(workerCtx)
	eg.Go(func() error {
		return ioWorker(egCtx, batchChan, options.reader)
	})

	var wg sync.WaitGroup
	for i := 0; i < NumWorkers; i++ {
		wg.Add(1)
		eg.Go(func() error {
			defer wg.Done()

			d := oracle.CalcD(numLabels, cfg.B)
			difficulty := shared.ProvingDifficulty2(numLabels, uint64(d), uint64(cfg.K1))
			numOuts := uint8(math.Ceil(float64(cfg.N) * float64(d) / aes.BlockSize))
			return labelWorker(egCtx, batchChan, solutionChan, ch, numOuts, cfg.N, d, difficulty)
		})
	}

	result := &nonceResult{}
	eg.Go(func() error {
		var err error
		result, err = solutionWorker(egCtx, solutionChan, numLabels, cfg.K2, logger)
		cancel()
		return err
	})

	wg.Wait()
	close(solutionChan)
	if err := eg.Wait(); err != nil && err != context.Canceled {
		return nil, nil, err
	}

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	if result == nil {
		return nil, nil, errors.New("no proof found")
	}

	logger.Info("proving: generated proof")

	proof := &Proof{
		Nonce:   result.nonce,
		Indices: result.indices,
	}
	proofMetadata := &ProofMetadata{
		NodeId:          options.nodeId,
		CommitmentAtxId: options.commitmentAtxId,
		Challenge:       ch,
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   cfg.LabelsPerUnit,
		NumUnits:        options.numUnits,
		K1:              cfg.K1,
		K2:              cfg.K2,
		N:               cfg.N,
		B:               cfg.B,
	}
	return proof, proofMetadata, nil
}

func verifyMetadata(m *Metadata, cfg Config, datadir string, nodeId, commitmentAtxId []byte) error {
	if !bytes.Equal(nodeId, m.NodeId) {
		return ConfigMismatchError{
			Param:    "NodeId",
			Expected: fmt.Sprintf("%x", nodeId),
			Found:    fmt.Sprintf("%x", m.NodeId),
			DataDir:  datadir,
		}
	}

	if !bytes.Equal(commitmentAtxId, m.CommitmentAtxId) {
		return ConfigMismatchError{
			Param:    "CommitmentAtxId",
			Expected: fmt.Sprintf("%x", commitmentAtxId),
			Found:    fmt.Sprintf("%x", m.CommitmentAtxId),
			DataDir:  datadir,
		}
	}

	if cfg.BitsPerLabel != m.BitsPerLabel {
		return ConfigMismatchError{
			Param:    "BitsPerLabel",
			Expected: fmt.Sprintf("%d", cfg.BitsPerLabel),
			Found:    fmt.Sprintf("%d", m.BitsPerLabel),
			DataDir:  datadir,
		}
	}

	if cfg.LabelsPerUnit != m.LabelsPerUnit {
		return ConfigMismatchError{
			Param:    "LabelsPerUnit",
			Expected: fmt.Sprintf("%d", cfg.LabelsPerUnit),
			Found:    fmt.Sprintf("%d", m.LabelsPerUnit),
			DataDir:  datadir,
		}
	}

	return nil
}

// TODO(mafa): this should be part of the new persistence package
// missing data should be ignored up to a certain threshold.
func initCompleted(datadir string, numUnits uint32, bitsPerLabel uint8, labelsPerUnit uint64) (bool, error) {
	diskState := initialization.NewDiskState(datadir, uint(bitsPerLabel))
	numLabelsWritten, err := diskState.NumLabelsWritten()
	if err != nil {
		return false, err
	}

	target := uint64(numUnits) * labelsPerUnit
	return numLabelsWritten == target, nil
}
