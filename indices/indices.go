package indices

import (
	"encoding/binary"
	"github.com/spacemeshos/sha256-simd"
	"math/bits"
)

func CalcProvenLeafIndices(merkleRoot []byte, maxLabelIndex uint64, numberOfProvenLabels, difficulty uint8) map[uint64]bool {
	provenLabelIndices := DrawProvenLabelIndices(merkleRoot, maxLabelIndex, numberOfProvenLabels)
	return ConvertLabelIndicesToLeafIndices(provenLabelIndices, difficulty)
}

func ConvertLabelIndicesToLeafIndices(labelIndices map[uint64]bool, difficulty uint8) (leafIndices map[uint64]bool) {
	leafIndices = make(map[uint64]bool)
	for key, value := range labelIndices {
		leafIndices[key>>difficulty] = value
	}
	return leafIndices
}

func DrawProvenLabelIndices(merkleRoot []byte, maxIndex uint64, numberOfProvenLabels uint8) (labelIndices map[uint64]bool) {
	if maxIndex+1 < uint64(numberOfProvenLabels) {
		return nil
	}
	bitsRequiredForIndex := uint(bits.Len64(maxIndex))
	indexMask := (uint64(1) << bitsRequiredForIndex) - 1
	labelIndices = make(map[uint64]bool)
	for i := uint8(0); len(labelIndices) < int(numberOfProvenLabels); i++ {
		result := sha256.Sum256(append(merkleRoot, i))
		masked := binary.LittleEndian.Uint64(result[:]) & indexMask
		if masked > maxIndex {
			continue
		}
		labelIndices[masked] = true
	}
	return labelIndices
}
