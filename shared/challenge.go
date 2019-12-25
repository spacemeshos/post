package shared

import "github.com/spacemeshos/sha256-simd"

type Challenge []byte

var (
	ZeroChallenge = make(Challenge, 0)
	message []byte
)

// ⚠️ This method is NOT thread-safe
func (ch Challenge) GetSha256Parent(lChild, rChild []byte) []byte {
	if len(message) != len(ch)+len(lChild)+len(rChild) {
		message = make([]byte, len(ch)+len(lChild)+len(rChild))
	}
	copy(message, ch)
	copy(message[len(ch):], lChild)
	copy(message[len(ch)+len(lChild):], rChild)
	res := sha256.Sum256(message)
	return res[:]
}
