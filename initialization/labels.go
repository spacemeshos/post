package initialization

import (
	"encoding/binary"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/sha256-simd"
)

func CalcLabelGroup(identity []byte, groupPosition uint64, difficulty proving.Difficulty) []byte {
	labelGroup := make([]byte, merkle.NodeSize)
	labelsPerGroup := difficulty.LabelsPerGroup()
	offset := groupPosition * labelsPerGroup
	for labelIndex := uint64(0); labelIndex < labelsPerGroup; labelIndex++ {
		label := CalcLabel(identity, labelIndex+offset, difficulty)
		byteIndex := difficulty.ByteIndex(labelIndex)
		// This causes labels to be added in LIFO order:
		labelGroup[byteIndex] <<= difficulty.LabelBits()
		labelGroup[byteIndex] += label
	}
	return labelGroup
}

func CalcLabel(identity []byte, position uint64, difficulty proving.Difficulty) byte {
	preimage := make([]byte, len(identity)+binary.Size(position))
	copy(preimage, identity)
	binary.LittleEndian.PutUint64(preimage[len(identity):], position)

	sum256 := sha256.Sum256(preimage)
	return sum256[0] & difficulty.LabelMask()
}
