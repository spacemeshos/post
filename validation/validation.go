package validation

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"math"
	"strconv"
)

// Validate ensures the validity of the given proof. It returns nil if the proof is valid or an error describing the
// failure, otherwise.
func Validate(proof proving.Proof, space proving.Space, numberOfProvenLabels uint8, difficulty proving.Difficulty) error {
	if err := space.Validate(initialization.LabelGroupSize); err != nil {
		log.Error(err.Error())
		return err
	}
	if err := difficulty.Validate(); err != nil {
		log.Error(err.Error())
		return err
	}

	numOfLabelGroups := space.LabelGroups(initialization.LabelGroupSize)
	err := validate(proof, numOfLabelGroups, numberOfProvenLabels, difficulty)
	if err != nil {
		err = fmt.Errorf("validation failed: %v", err)
		log.Info(err.Error())
	}
	return err
}

func validate(proof proving.Proof, numOfLabelGroups uint64, numberOfProvenLabels uint8, difficulty proving.Difficulty) error {
	labelIndices := proving.DrawProvenLabelIndices(proof.MerkleRoot, numOfLabelGroups*difficulty.LabelsPerGroup(),
		numberOfProvenLabels)
	leafIndices := proving.ConvertLabelIndicesToLeafIndices(labelIndices, difficulty)
	// The number of proven leaves could be less than the number of proven labels since more than one label could be in
	// the same leaf. That's why we can only validate the number of proven leaves after drawing the proven labels and
	// determining which leaf each one falls in.
	if len(leafIndices) != len(proof.ProvenLeaves) {
		return fmt.Errorf("number of derived leaf indices (%d) doesn't match number of included proven "+
			"leaves (%d)", len(leafIndices), len(proof.ProvenLeaves))
	}
	valid, err := merkle.ValidatePartialTree(
		leafIndices.AsSortedSlice(),
		proof.ProvenLeaves,
		proof.ProofNodes,
		proof.MerkleRoot,
		proof.Challenge.GetSha256Parent,
	)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("merkle root mismatch")
	}
	return validatePow(proof.Identity, proof.ProvenLeaves, labelIndices, difficulty)
}

func getLabelAtIndex(l byte, indexInByte uint64, difficulty proving.Difficulty) byte {
	labelsToClear := difficulty.LabelsPerByte() - 1 - indexInByte
	return l >> (labelsToClear * difficulty.LabelBits()) & difficulty.LabelMask()
}

func validatePow(identity []byte, provenLeaves [][]byte, labelIndices proving.Set, difficulty proving.Difficulty) error {
	var currentLeafIndex uint64 = math.MaxUint64
	var currentLeaf []byte
	for labelIndexList := labelIndices.AsSortedSlice(); len(labelIndexList) > 0; labelIndexList = labelIndexList[1:] {
		leafIndex := difficulty.LeafIndex(labelIndexList[0])
		if leafIndex != currentLeafIndex {
			// Proven leaves are expected to be sorted (or validation fails)
			currentLeaf = provenLeaves[0]
			provenLeaves = provenLeaves[1:]
			currentLeafIndex = leafIndex
		}
		intraLeafIndex := difficulty.IndexInGroup(labelIndexList[0])
		label := getLabelAtIndex(currentLeaf[difficulty.ByteIndex(intraLeafIndex)],
			difficulty.IndexInByte(intraLeafIndex), difficulty)
		expectedLabel := initialization.CalcLabel(identity, labelIndexList[0], difficulty)
		if label != expectedLabel {
			lBits := strconv.Itoa(int(difficulty.LabelBits()))
			return fmt.Errorf("label at index %d should be %0"+lBits+"b, but found %0"+lBits+"b",
				labelIndexList[0], label, expectedLabel)
		}
	}
	return nil
}
