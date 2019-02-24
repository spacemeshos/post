package util

import (
	"bytes"
	"github.com/spacemeshos/sha256-simd"
)

type Hash []byte

func (h Hash) IsLessThan(other Hash) bool {
	return bytes.Compare(h, other) <= 0
}

func CalcHash(byteArrays ...[]byte) Hash {
	h := sha256.New()
	for _, ba := range byteArrays {
		h.Write(ba)
	}
	return h.Sum(nil)[:]
}
