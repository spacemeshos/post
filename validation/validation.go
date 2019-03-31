package validation

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/post-private/initialization"
	"github.com/spacemeshos/post-private/proving"
	"math"
	"strconv"
)

func Validate(proof proving.Proof, leafCount uint64, numberOfProvenLabels uint8, difficulty proving.Difficulty) error {
	labelIndices := proving.DrawProvenLabelIndices(proof.MerkleRoot, leafCount*difficulty.LabelsPerGroup(),
		numberOfProvenLabels)
	leafIndices := proving.ConvertLabelIndicesToLeafIndices(labelIndices, difficulty)
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
		return fmt.Errorf("validation failed: %v", err)
	}
	if !valid {
		return errors.New("validation failed: merkle root mismatch")
	}
	return validatePow(proof.Identity, proof.ProvenLeaves, labelIndices, difficulty)
}

type labelsByte uint8

func (l labelsByte) GetLabelAtIndex(indexInByte uint64, difficulty proving.Difficulty) byte {
	labelsToClear := difficulty.LabelsPerByte() - 1 - indexInByte
	return byte(l) >> (labelsToClear * difficulty.LabelBits()) & difficulty.LabelMask()
}

func validatePow(identity []byte, provenLeaves [][]byte, labelIndices proving.Set, difficulty proving.Difficulty) error {
	var currentLeafIndex uint64 = math.MaxUint64
	var currentLeaf []byte
	for labelIndexList := labelIndices.AsSortedSlice(); len(labelIndexList) > 0; labelIndexList = labelIndexList[1:] {
		leafIndex := difficulty.LeafIndex(labelIndexList[0])
		if leafIndex != currentLeafIndex {
			currentLeaf = provenLeaves[0]
			provenLeaves = provenLeaves[1:]
			currentLeafIndex = leafIndex
		}
		intraLeafIndex := difficulty.IndexInLeaf(labelIndexList[0])
		b := labelsByte(currentLeaf[difficulty.ByteIndex(intraLeafIndex)])
		label := b.GetLabelAtIndex(difficulty.IndexInByte(intraLeafIndex), difficulty)
		expectedLabel := initialization.CalcLabel(identity, labelIndexList[0], difficulty)
		if label != expectedLabel {
			lBits := strconv.Itoa(int(difficulty.LabelBits()))
			return fmt.Errorf("label at index %d should be %0"+lBits+"b, but found %0"+lBits+"b",
				labelIndexList[0], label, expectedLabel)
		}
	}
	return nil
}
