package proving

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
)

// TODO (mafa): make this configurable.
const numProvingWorkers = 1

// TODO (mafa): use functional options.
// TODO (mafa): replace Logger with zap.
func Generate(ctx context.Context, ch Challenge, cfg Config, datadir string, nodeId, commitmentAtxId []byte, logger Logger) (*Proof, *ProofMetadata, error) {
	m, err := initialization.LoadMetadata(datadir)
	if err != nil {
		return nil, nil, err
	}

	if err := verifyMetadata(m, cfg, datadir, nodeId, commitmentAtxId); err != nil {
		return nil, nil, err
	}

	if ok, err := initCompleted(datadir, m.NumUnits, cfg.BitsPerLabel, cfg.LabelsPerUnit); err != nil {
		return nil, nil, err
	} else if !ok {
		return nil, nil, shared.ErrInitNotCompleted
	}

	var reader io.Reader
	batchQueue := make(chan *batch)
	solutionQueue := make(chan *solution)

	difficulty := make([]byte, 8)
	binary.BigEndian.PutUint64(
		difficulty,
		shared.ProvingDifficulty(uint64(m.NumUnits), uint64(cfg.K1)),
	)

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, egCtx := errgroup.WithContext(workerCtx)
	eg.Go(func() error {
		return ioWorker(egCtx, batchQueue, reader)
	})

	for i := 0; i < numProvingWorkers; i++ {
		eg.Go(func() error {
			return labelWorker(egCtx, batchQueue, solutionQueue, ch, difficulty)
		})
	}

	eg.Go(func() error {
		// TODO(mafa): collect indices for proof from solution queue.
		// and signal stop via
		cancel()

		return nil
	})

	if err := eg.Wait(); err != nil && err != context.Canceled {
		return nil, nil, err
	}

	// TODO(mafa): close solution queue here.

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	// TODO(mafa): use proof collected in proof collector.
	solutionNonceResult := &struct {
		nonce   uint32
		indices []byte
		err     error
	}{}

	if solutionNonceResult == nil {
		return nil, nil, errors.New("no proof found")
	}

	logger.Info("proving: generated proof")

	proof := &Proof{
		Nonce:   solutionNonceResult.nonce,
		Indices: solutionNonceResult.indices,
	}
	proofMetadata := &ProofMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		Challenge:       ch,
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   cfg.LabelsPerUnit,
		NumUnits:        m.NumUnits,
		K1:              cfg.K1,
		K2:              cfg.K2,
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
