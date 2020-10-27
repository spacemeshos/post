package verifying

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
)

type (
	Config = config.Config
)

var (
	WorkOracleOne = oracle.WorkOracleOne
	FastOracle    = oracle.FastOracle
)

// Verify ensures the validity of the given proof. It returns nil if the proof is valid or an error describing the
// failure, otherwise.
func Verify(p *proving.Proof, m *proving.ProofMetadata) error {
	if len(m.ID) != 32 {
		return fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(m.ID))
	}

	var indexBitSize = uint(shared.NumBits(m.NumLabels))
	var expectedSize = shared.Size(indexBitSize, m.K2)
	if expectedSize != uint(len(p.Indices)) {
		return fmt.Errorf("invalid indices set size; expected %d, given: %d", expectedSize, len(p.Indices))
	}

	difficulty := shared.ProvingDifficulty(m.NumLabels, uint64(m.K1))
	buf := bytes.NewBuffer(p.Indices)
	gsReader := shared.NewGranSpecificReader(buf, indexBitSize)
	indicesSet := make(map[uint64]bool, m.K2)

	for i := uint(0); i < m.K2; i++ {
		index, err := gsReader.ReadNextUintBE()
		if err != nil {
			return err
		}
		if indicesSet[index] {
			return fmt.Errorf("non-unique index: %d", index)
		}
		indicesSet[index] = true

		label := WorkOracleOne(initialization.CPUProviderID(), m.ID, index, uint32(m.LabelSize))
		hash := FastOracle(m.Challenge, p.Nonce, label)
		hashNum := binary.LittleEndian.Uint64(hash[:])
		if hashNum > difficulty {
			return fmt.Errorf("fast oracle output is above the threshold; index: %d, label: %x, hash: %x, hashNum: %d, difficulty: %d",
				index, label, hash, hashNum, difficulty)
		}
	}

	return nil
}
