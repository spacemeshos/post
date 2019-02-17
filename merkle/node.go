package merkle

import (
	"encoding/binary"
	"encoding/hex"
)

type node []byte

func (l node) String() string {
	return hex.EncodeToString(l)[:4]
}

func newNodeFromUint64(i uint64) node {
	const bytesInUint64 = 8
	b := make([]byte, bytesInUint64)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

func newNodeFromHex(s string) (node, error) {
	return hex.DecodeString(s)
}
