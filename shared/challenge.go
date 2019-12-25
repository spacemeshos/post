package shared

import "github.com/spacemeshos/sha256-simd"

type Challenge []byte

var (
	ZeroChallenge = make(Challenge, 0)
	buffer        []byte
)

// ⚠️ This method is NOT thread-safe. The code is optimized for performance and memory allocations.
func (ch Challenge) GetSha256Parent(lChild, rChild []byte) []byte {
	l := len(ch) + len(lChild) + len(rChild)
	if len(buffer) < l {
		buffer = make([]byte, l)
	}
	copy(buffer, ch)
	copy(buffer[len(ch):], lChild)
	copy(buffer[len(ch)+len(lChild):], rChild)
	res := sha256.Sum256(buffer[:l])
	return res[:]
}
