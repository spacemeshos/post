package proving

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/shared"
)

func Generate(
	ctx context.Context,
	ch shared.Challenge,
	cfg config.Config,
	logger *zap.Logger,
	opts ...OptionFunc,
) (*shared.Proof, *shared.ProofMetadata, error) {
	options := option{
		threads:  1,
		nonces:   16,
		powFlags: config.DefaultProvingPowFlags(),
	}
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return nil, nil, err
		}
	}
	if err := options.validate(); err != nil {
		return nil, nil, err
	}

	result, err := postrs.GenerateProof(
		options.datadir,
		ch, logger,
		options.nonces,
		options.threads,
		cfg.K1, cfg.K2,
		cfg.PowDifficulty, options.powFlags,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("generating proof: %w", err)
	}
	logger.Info("proving: generated proof")
	logger.Debug("proving: generated proof",
		zap.Uint32("Nonce", result.Nonce),
		zap.String("Indices", hex.EncodeToString(result.Indices)),
		zap.Uint64("PoW", result.Pow),
	)

	proof := &shared.Proof{Nonce: result.Nonce, Indices: result.Indices, Pow: result.Pow}
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
			Expected: hex.EncodeToString(nodeId),
			Found:    hex.EncodeToString(m.NodeId),
			DataDir:  datadir,
		}
	}

	if !bytes.Equal(commitmentAtxId, m.CommitmentAtxId) {
		return shared.ConfigMismatchError{
			Param:    "CommitmentAtxId",
			Expected: hex.EncodeToString(commitmentAtxId),
			Found:    hex.EncodeToString(m.CommitmentAtxId),
			DataDir:  datadir,
		}
	}

	if cfg.LabelsPerUnit != m.LabelsPerUnit {
		return shared.ConfigMismatchError{
			Param:    "LabelsPerUnit",
			Expected: strconv.FormatUint(cfg.LabelsPerUnit, 10),
			Found:    strconv.FormatUint(m.LabelsPerUnit, 10),
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
