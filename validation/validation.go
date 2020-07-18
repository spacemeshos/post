package validation

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/proving"
)

type (
	Config = config.Config
)

var (
	WorkOracle = oracle.WorkOracle
	FastOracle = oracle.FastOracle
)

type Validator struct {
	cfg *Config
}

func NewValidator(cfg *Config) (*Validator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Validator{cfg}, nil
}

// Validate ensures the validity of the given proof. It returns nil if the proof is valid or an error describing the
// failure, otherwise.
func (v *Validator) Validate(identity []byte, p *proving.Proof) error {
	expectedNum := v.cfg.K2
	expectedSize := expectedNum * 8
	givenSize := len(p.Indices)
	if uint(givenSize) != expectedSize {
		return fmt.Errorf("invalid indices set size; expected %d, given: %d", expectedSize, givenSize)
	}

	difficulty := v.cfg.ProvingDifficulty()
	buf := bytes.NewBuffer(p.Indices)
	indicesSet := make(map[uint64]bool, expectedNum)

	for i := uint(0); i < expectedNum; i++ {
		indexBytes := buf.Next(8)
		index := binary.LittleEndian.Uint64(indexBytes)
		if indicesSet[index] {
			return fmt.Errorf("non-unique index: %d", index)
		}
		indicesSet[index] = true

		label := WorkOracle(identity, index, v.cfg.LabelSize)
		hash := FastOracle(p.Challenge, p.Nonce, label)
		hashNum := binary.LittleEndian.Uint64(hash[:])
		if hashNum > difficulty {
			return fmt.Errorf("fast oracle output is above the threshold; index: %d, label: %x, hash: %x, hashNum: %d, difficulty: %d",
				index, label, hash, hashNum, difficulty)
		}
	}

	return nil
}

//func validate(identity []byte, proof proving.Proof, numLabelGroups uint64, numProvenLabels uint8, difficulty proving.Difficulty) error {
//	labelIndices := shared.DrawProvenLabelIndices(proof.MerkleRoot, numLabelGroups*difficulty.LabelsPerGroup(),
//		numProvenLabels)
//	leafIndices := shared.ConvertLabelIndicesToLeafIndices(labelIndices, difficulty)
//	// The number of proven leaves could be less than the number of proven labels since more than one label could be in
//	// the same leaf. That's why we can only validate the number of proven leaves after drawing the proven labels and
//	// determining which leaf each one falls in.
//	if len(leafIndices) != len(proof.ProvenLeaves) {
//		return fmt.Errorf("number of derived leaf indices (%d) doesn't match number of included proven "+
//			"leaves (%d)", len(leafIndices), len(proof.ProvenLeaves))
//	}
//	valid, err := merkle.ValidatePartialTree(
//		leafIndices.AsSortedSlice(),
//		proof.ProvenLeaves,
//		proof.ProofNodes,
//		proof.MerkleRoot,
//		proof.Challenge.GenerateGetParentFunc(),
//	)
//	if err != nil {
//		return err
//	}
//	if !valid {
//		return errors.New("merkle root mismatch")
//	}
//	return validatePow(identity, proof.ProvenLeaves, labelIndices, difficulty)
//}

//func getLabelAtIndex(l byte, indexInByte uint64, difficulty proving.Difficulty) byte {
//	labelsToClear := difficulty.LabelsPerByte() - 1 - indexInByte
//	return l >> (labelsToClear * difficulty.LabelBits()) & difficulty.LabelMask()
//}
//
//func validatePow(identity []byte, provenLeaves [][]byte, labelIndices shared.Set, difficulty proving.Difficulty) error {
//	var currentLeafIndex uint64 = math.MaxUint64
//	var currentLeaf []byte
//	for labelIndexList := labelIndices.AsSortedSlice(); len(labelIndexList) > 0; labelIndexList = labelIndexList[1:] {
//		leafIndex := difficulty.LeafIndex(labelIndexList[0])
//		if leafIndex != currentLeafIndex {
//			// Proven leaves are expected to be sorted (or validation fails)
//			currentLeaf = provenLeaves[0]
//			provenLeaves = provenLeaves[1:]
//			currentLeafIndex = leafIndex
//		}
//		intraLeafIndex := difficulty.IndexInGroup(labelIndexList[0])
//		label := getLabelAtIndex(currentLeaf[difficulty.ByteIndex(intraLeafIndex)],
//			difficulty.IndexInByte(intraLeafIndex), difficulty)
//		expectedLabel := initialization.CalcLabel(identity, labelIndexList[0], difficulty)
//		if label != expectedLabel {
//			lBits := strconv.Itoa(int(difficulty.LabelBits()))
//			return fmt.Errorf("label at index %d should be %0"+lBits+"b, but found %0"+lBits+"b",
//				labelIndexList[0], label, expectedLabel)
//		}
//	}
//	return nil
//}
