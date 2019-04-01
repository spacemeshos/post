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
	for labelIndex, value := range labelIndices {
		leafIndices[difficulty.LeafIndex(labelIndex)] = value
	}
	return leafIndices
}

// DrawProvenLabelIndices returns a set containing numberOfProvenLabels label indices to prove. The indices are derived
// deterministically from merkleRoot. The indices are uniformly distributed in the range 0-(numberOfLabels-1).
//
// To ensure a uniform distribution, the minimal number of bits required to represent a number in the target range is
// taken from a hash of the merkleRoot and a running counter. If the drawn number is still outside the range bounds,
// it's discarded and a new number is drawn in its place (with a higher counter value).
//
// The expected number of drawn indices (including the discarded ones) is, at most, twice the numberOfProvenLabels (this
// happens when it falls in the middle of the range between two powers of 2).
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
