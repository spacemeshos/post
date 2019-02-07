package datatypes

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/cbergoon/merkletree"
)

const LabelSize = 8

type Label []byte

// START merkletree.Content methods

func (l Label) CalculateHash() ([]byte, error) {
	return l, nil
}

func (l Label) Equals(other merkletree.Content) (bool, error) {
	return bytes.Compare(l, other.(Label)) == 0, nil
}

// END merkletree.Content methods

func (l Label) String() string {
	return hex.EncodeToString(l)[:5]
}

func NewLabel(cnt uint64) Label {
	b := make([]byte, LabelSize)
	binary.LittleEndian.PutUint64(b, cnt)
	return b
}
