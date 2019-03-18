package labels

import (
	"encoding/binary"
	"github.com/spacemeshos/sha256-simd"
)

const (
	LabelBytes  = 32
	Uint64Bytes = 8
	Uint8Bytes  = 1
)

func CalcLabelGroup(identity []byte, position uint64) []byte {
	label := make([]byte, 0, LabelBytes)
	for i := uint8(0); i < LabelBytes; i++ {
		label = append(label, calcLabel(identity, position, i))
	}
	return label
}

func calcLabel(identity []byte, position uint64, pieceId uint8) byte {
	posBytes := make([]byte, Uint64Bytes)
	binary.LittleEndian.PutUint64(posBytes, position)

	preimage := make([]byte, 0, len(identity)+Uint64Bytes+Uint8Bytes)
	preimage = append(identity, posBytes...)
	preimage = append(preimage, pieceId)

	sum256 := sha256.Sum256(preimage)
	return sum256[0]
}
