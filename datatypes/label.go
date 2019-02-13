package datatypes

import (
	"encoding/binary"
	"encoding/hex"
)

const LabelSize = 8

type Label []byte

func (l Label) String() string {
	return hex.EncodeToString(l)[:5]
}

func NewLabel(cnt uint64) []byte {
	b := make([]byte, LabelSize)
	binary.LittleEndian.PutUint64(b, cnt)
	return b
}
