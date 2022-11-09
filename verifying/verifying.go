package verifying

import (
	"bytes"
	"fmt"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

type (
	Config = config.Config
)

var (
	WorkOracle = oracle.WorkOracle
	FastOracle = oracle.FastOracle
	UInt64LE   = shared.UInt64LE
)

// Verify ensures the validity of a proof in respect to its metadata.
// It returns nil if the proof is valid or an error describing the failure, otherwise.
func Verify(p *shared.Proof, m *shared.ProofMetadata) error {
	if len(m.Commitment) != 32 {
		return fmt.Errorf("invalid `commitment` length; expected: 32, given: %v", len(m.Commitment))
	}

	numLabels := uint64(m.NumUnits) * uint64(m.LabelsPerUnit)
	bitsPerIndex := uint(shared.BinaryRepresentationMinBits(numLabels))
	expectedSize := shared.Size(bitsPerIndex, uint(m.K2))
	if expectedSize != uint(len(p.Indices)) {
		return fmt.Errorf("invalid indices set size; expected %d, given: %d", expectedSize, len(p.Indices))
	}

	difficulty := shared.ProvingDifficulty(numLabels, uint64(m.K1))
	buf := bytes.NewBuffer(p.Indices)
	gsReader := shared.NewGranSpecificReader(buf, bitsPerIndex)
	indicesSet := make(map[uint64]bool, m.K2)

	for i := uint(0); i < uint(m.K2); i++ {
		index, err := gsReader.ReadNextUintBE()
		if err != nil {
			return err
		}
		if indicesSet[index] {
			return fmt.Errorf("non-unique index: %d", index)
		}
		indicesSet[index] = true

		// TODO(mafa): verification of nonce happens here
		// create a new verify method that checks if the index of PoW fullfils the difficulty
		res, err := WorkOracle(
			oracle.WithCommitment(m.Commitment),
			oracle.WithPosition(index),
			oracle.WithBitsPerLabel(uint32(m.BitsPerLabel)),
		)
		if err != nil {
			return err
		}
		hash := FastOracle(m.Challenge, p.Nonce, res.Output)
		hashNum := UInt64LE(hash[:])
		if hashNum > difficulty {
			return fmt.Errorf("fast oracle output is above the threshold; index: %d, label: %x, hash: %x, hashNum: %d, difficulty: %d",
				index, res.Output, hash, hashNum, difficulty)
		}
	}

	return nil
}
