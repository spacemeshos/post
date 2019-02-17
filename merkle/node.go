package merkle

import (
	"encoding/binary"
	"encoding/hex"
)

type Node []byte

func (l Node) String() string {
	return hex.EncodeToString(l)[:4]
}

func NewNodeFromUint64(i uint64) Node {
	const bytesInUint64 = 8
	b := make([]byte, bytesInUint64)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

func NewNodeFromHex(s string) (Node, error) {
	return hex.DecodeString(s)
}
