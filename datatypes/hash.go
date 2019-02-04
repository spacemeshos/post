package datatypes

import (
	"bytes"
	"github.com/spacemeshos/sha256-simd"
)

type Hash []byte

func (h Hash) IsLessThan(other Hash) bool {
	return bytes.Compare(h, other) <= 0
}

func CalcHash(id []byte, l Label) Hash {
	h := sha256.New()
	h.Write(id)
	h.Write(l)
	return h.Sum(nil)[:]
}
