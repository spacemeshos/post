package shared

import "github.com/spacemeshos/sha256-simd"

type Challenge []byte

var ZeroChallenge = make(Challenge, 0)

func (ch Challenge) GetSha256Parent(lChild, rChild []byte) []byte {
	children := append(lChild, rChild...)
	res := sha256.Sum256(append(ch, children...))
	return res[:]
}
