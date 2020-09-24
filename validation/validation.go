package validation

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

type Validator struct {
	cfg *Config
}

// Validate ensures the validity of the given proof. It returns nil if the proof is valid or an error describing the
// failure, otherwise.
func Validate(id []byte, p *proving.Proof, m *proving.ProofMetadata) error {
	if len(id) != 32 {
		return fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(id))
	}

	expectedNum := m.K2
	expectedSize := expectedNum * 8
	if expectedSize != uint(len(p.Indices)) {
		return fmt.Errorf("invalid indices set size; expected %d, given: %d", expectedSize, len(p.Indices))
	}

	difficulty := shared.ProvingDifficulty(m.NumLabels, uint64(m.K1))
	buf := bytes.NewBuffer(p.Indices)
	indicesSet := make(map[uint64]bool, expectedNum)

	for i := uint(0); i < expectedNum; i++ {
		indexBytes := buf.Next(8)
		index := binary.LittleEndian.Uint64(indexBytes)
		if indicesSet[index] {
			return fmt.Errorf("non-unique index: %d", index)
		}
		indicesSet[index] = true

		label := WorkOracleOne(initialization.CPUProviderID(), id, index, uint8(m.LabelSize))
		hash := FastOracle(m.Challenge, p.Nonce, label)
		hashNum := binary.LittleEndian.Uint64(hash[:])
		if hashNum > difficulty {
			return fmt.Errorf("fast oracle output is above the threshold; index: %d, label: %x, hash: %x, hashNum: %d, difficulty: %d",
				index, label, hash, hashNum, difficulty)
		}
	}

	return nil
}
