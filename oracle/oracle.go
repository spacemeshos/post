package oracle

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/spacemeshos/post/shared"
)

type (
	Challenge = shared.Challenge
)

func WorkOracle(identity []byte, index uint64, size uint) []byte {
	input := make([]byte, len(identity)+binary.Size(index))
	copy(input, identity)
	binary.LittleEndian.PutUint64(input[len(identity):], index)

	sum256 := sha256.Sum256(input) // TODO(moshababo): use scrypt
	return sum256[:size]
}

func FastOracle(ch Challenge, nonce uint32, label []byte) [32]byte {
	input := make([]byte, 32+4+len(label))

	copy(input, ch)
	binary.LittleEndian.PutUint32(input[32:], nonce)
	copy(input[36:], label[:])

	return sha256.Sum256(input)
}
