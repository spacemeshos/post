package oracle

import (
	"crypto/aes"
	"crypto/cipher"
	"math"
)

// CalcD calculates the number of bytes to use for the difficulty check.
// numLabels is the number of labels contained in the PoST data.
// B is a network parameter that defines the number of labels used in one AES Block.
func CalcD(numLabels uint64, B uint32) uint {
	return uint(math.Ceil((math.Log2(float64(numLabels)) - math.Log2(float64(B))) / 8))
}

// CreateBlockCipher creates an AES cipher for given fast oracle block.
// A cipher is created using an idx encrypted with challenge:
//
//	cipher = AES(AES(ch).Encrypt(i))
func CreateBlockCipher(ch Challenge, nonce uint8) (cipher.Block, error) {
	// A temporary cipher used only to create key.
	// The key is a block encrypted with AES which key is the challenge.
	keyCipher, err := aes.NewCipher(ch)
	if err != nil {
		return nil, err
	}

	keyBuffer := make([]byte, aes.BlockSize)
	keyBuffer[0] = nonce
	key := make([]byte, aes.BlockSize)
	keyCipher.Encrypt(key, keyBuffer)
	return aes.NewCipher(key)
}
