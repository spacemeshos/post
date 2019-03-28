package proving

import (
	"encoding/binary"
	"github.com/spacemeshos/sha256-simd"
	"math/bits"
)

func CalcProvenLeafIndices(merkleRoot []byte, numberOfLabels uint64, numberOfProvenLabels uint8,
	difficulty Difficulty) Set {

	provenLabelIndices := DrawProvenLabelIndices(merkleRoot, numberOfLabels, numberOfProvenLabels)
	return ConvertLabelIndicesToLeafIndices(provenLabelIndices, difficulty)
}

func ConvertLabelIndicesToLeafIndices(labelIndices Set, difficulty Difficulty) (leafIndices Set) {
	leafIndices = make(Set)
	for key, value := range labelIndices {
		leafIndices[key>>difficulty] = value
	}
	return leafIndices
}

func DrawProvenLabelIndices(merkleRoot []byte, numberOfLabels uint64, numberOfProvenLabels uint8) (labelIndices Set) {
	if numberOfLabels < uint64(numberOfProvenLabels) {
		return nil
	}
	bitsRequiredForIndex := uint(bits.Len64(numberOfLabels - 1))
	indexMask := (uint64(1) << bitsRequiredForIndex) - 1
	labelIndices = make(Set)
	for i := uint8(0); len(labelIndices) < int(numberOfProvenLabels); i++ {
		result := sha256.Sum256(append(merkleRoot, i))
		masked := binary.LittleEndian.Uint64(result[:]) & indexMask
		if masked > numberOfLabels-1 {
			continue
		}
		labelIndices[masked] = true
	}
	return labelIndices
}
