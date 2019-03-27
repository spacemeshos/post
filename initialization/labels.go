package initialization

import (
	"encoding/binary"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/sha256-simd"
)

func CalcLabelGroup(identity []byte, groupPosition uint64, difficulty uint8) []byte {
	labelGroup := make([]byte, merkle.NodeSize)
	offset := groupPosition << difficulty
	internalShift := uint8(1) << (8 - difficulty)
	labelMask := (uint8(1) << internalShift) - 1
	for position := uint64(0); position < 1<<difficulty; position++ {
		label := calcLabel(identity, position+offset)
		adjustedPos := position >> (difficulty - 5)
		labelGroup[adjustedPos] <<= internalShift
		labelGroup[adjustedPos] += label & labelMask
	}
	return labelGroup
}

func calcLabel(identity []byte, position uint64) byte {
	preimage := make([]byte, len(identity)+binary.Size(position))
	copy(preimage, identity)
	binary.LittleEndian.PutUint64(preimage[len(identity):], position)

	sum256 := sha256.Sum256(preimage)
	return sum256[0]
}
