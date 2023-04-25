package proving

import (
	"bytes"
	"context"
	"fmt"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/shared"
)

// TODO (mafa): these should be configurable by a node.
const (
	NumWorkers = 1 // Number of workers creating a proof in parallel. Each one will max out one CPU core.

	BlocksPerWorker = 1 << 24 // How many AES blocks are contained per batch sent to a worker. Larger values will increase memory usage, but speed up the proof generation.
)

// TODO (mafa): replace Logger with zap.
func Generate(ctx context.Context, ch shared.Challenge, cfg config.Config, logger shared.Logger, opts ...OptionFunc) (*shared.Proof, *shared.ProofMetadata, error) {
	options := option{
		threads:   1,
		nonces:    16,
		powScrypt: config.DefaultPowParams(),
	}
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return nil, nil, err
		}
	}
	if err := options.validate(); err != nil {
		return nil, nil, err
	}

	result, err := postrs.GenerateProof(options.datadir, ch, cfg, options.nonces, options.threads, options.powScrypt)
	if err != nil {
		return nil, nil, fmt.Errorf("generating proof: %w", err)
	}
	logger.Info("proving: generated proof")
	logger.Debug("Nonce: %v, Indices: %v, K2PoW: %d, K3PoW: %d", result.Nonce, result.Indices, result.K2Pow, result.K3Pow)

	proof := &shared.Proof{Nonce: result.Nonce, Indices: result.Indices, K2Pow: result.K2Pow, K3Pow: result.K3Pow}
	proofMetadata := &shared.ProofMetadata{
		NodeId:          options.nodeId,
		CommitmentAtxId: options.commitmentAtxId,
		Challenge:       ch,
		LabelsPerUnit:   cfg.LabelsPerUnit,
		NumUnits:        options.numUnits,
	}
	return proof, proofMetadata, nil
}

func verifyMetadata(m *shared.PostMetadata, cfg config.Config, datadir string, nodeId, commitmentAtxId []byte) error {
	if !bytes.Equal(nodeId, m.NodeId) {
		return shared.ConfigMismatchError{
			Param:    "NodeId",
			Expected: fmt.Sprintf("%x", nodeId),
			Found:    fmt.Sprintf("%x", m.NodeId),
			DataDir:  datadir,
		}
	}

	if !bytes.Equal(commitmentAtxId, m.CommitmentAtxId) {
		return shared.ConfigMismatchError{
			Param:    "CommitmentAtxId",
			Expected: fmt.Sprintf("%x", commitmentAtxId),
			Found:    fmt.Sprintf("%x", m.CommitmentAtxId),
			DataDir:  datadir,
		}
	}

	if cfg.LabelsPerUnit != m.LabelsPerUnit {
		return shared.ConfigMismatchError{
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
func initCompleted(datadir string, numUnits uint32, labelsPerUnit uint64) (bool, error) {
	diskState := initialization.NewDiskState(datadir, config.BitsPerLabel)
	numLabelsWritten, err := diskState.NumLabelsWritten()
	if err != nil {
		return false, err
	}

	target := uint64(numUnits) * labelsPerUnit
	return numLabelsWritten == target, nil
}
