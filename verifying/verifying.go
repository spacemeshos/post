package verifying

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

// VerifyVRFNonce ensures the validity of a nonce for a given node.
// AtxId is the id of the ATX that was selected by the node for its commitment.
func VerifyVRFNonce(nonce *uint64, m *shared.VRFNonceMetadata, opts ...OptionFunc) error {
	if nonce == nil {
		return errors.New("invalid `nonce` value; expected: non-nil, given: nil")
	}

	if len(m.NodeId) != 32 {
		return fmt.Errorf("invalid `nodeId` length; expected: 32, given: %v", len(m.NodeId))
	}

	if len(m.CommitmentAtxId) != 32 {
		return fmt.Errorf("invalid `commitmentAtxId` length; expected: 32, given: %v", len(m.CommitmentAtxId))
	}

	options := defaultOpts()
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return err
		}
	}

	numLabels := uint64(m.NumUnits) * uint64(m.LabelsPerUnit)
	difficulty := shared.PowDifficulty(numLabels)
	threshold := new(big.Int).SetBytes(difficulty)

	res, err := oracle.WorkOracle(
		oracle.WithCommitment(oracle.CommitmentBytes(m.NodeId, m.CommitmentAtxId)),
		oracle.WithPosition(*nonce),
		oracle.WithBitsPerLabel(256),
		oracle.WithScryptParams(options.labelScrypt),
	)
	if err != nil {
		return err
	}

	label := new(big.Int).SetBytes(res.Output)
	if label.Cmp(threshold) > 0 {
		return fmt.Errorf("label is above the threshold; label: %#32x, threshold: %#32x", label, threshold)
	}

	return nil
}

// Verify ensures the validity of a proof in respect to its metadata.
// It returns nil if the proof is valid or an error describing the failure, otherwise.
func Verify(p *shared.Proof, m *shared.ProofMetadata, cfg config.Config, opts ...OptionFunc) error {
	options := defaultOpts()
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return err
		}
	}
	if len(m.NodeId) != 32 {
		return fmt.Errorf("invalid `nodeId` length; expected: 32, given: %v", len(m.NodeId))
	}
	if len(m.CommitmentAtxId) != 32 {
		return fmt.Errorf("invalid `commitmentAtxId` length; expected: 32, given: %v", len(m.CommitmentAtxId))
	}

	return postrs.VerifyProof(p, m, cfg, 1, options.powScrypt, options.labelScrypt)
}
