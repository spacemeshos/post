package shared

import "github.com/spacemeshos/sha256-simd"

type Challenge []byte

var ZeroChallenge = make(Challenge, 0)

// ⚠️ The resulting method is NOT thread-safe, however different generated instances are independent.
// The code is optimized for performance and memory allocations.
func (ch Challenge) GenerateGetParentFunc() func(lChild, rChild []byte) []byte {
	var buffer []byte
	return func(lChild, rChild []byte) []byte {
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
}
