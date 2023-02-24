package shared

import (
	"encoding/binary"
	"math"
	"math/big"
	"os"
)

var (
	// OwnerReadWriteExec is a standard owner read / write / exec file permission.
	OwnerReadWriteExec = os.FileMode(0o700)

	// OwnerReadWrite is a standard owner read / write file permission.
	OwnerReadWrite = os.FileMode(0o600)
)

func DataSize(numLabels uint64, labelSize uint) uint64 {
	dataBitSize := numLabels * uint64(labelSize)
	dataSize := dataBitSize / 8
	if dataBitSize%8 > 0 {
		dataSize++
	}
	return dataSize
}

func NumLabels(dataSize uint64, labelSize uint) uint64 {
	dataBitSize := dataSize * 8
	return dataBitSize / uint64(labelSize)
}

func ProvingDifficulty(numLabels uint64, k1 uint64) uint64 {
	const maxTarget = math.MaxUint64
	x := maxTarget / numLabels
	y := maxTarget % numLabels
	return x*k1 + (y*k1)/numLabels
}

// ProvingDifficulty2 returns the target difficulty of finding a nonce in `numLabels` labels.
// d is the number of bytes in the AES output that count for one nonce.
func ProvingDifficulty2(numLabels uint64, B, k1 uint32) uint64 {
	// calculate the maximum value that a nonce can have based on the number of labels and B
	d := CalcD(numLabels, B)
	maxTarget := uint64(1<<(d*8)) - 1

	numIn := numLabels / uint64(B)
	x := maxTarget / numIn
	y := maxTarget % numIn
	return x*uint64(k1) + y*uint64(k1)/numIn
}

// CalcD calculates the number of bytes to use for the difficulty check.
// numLabels is the number of labels contained in the PoST data.
// B is a network parameter that defines the number of labels used in one AES Block.
func CalcD(numLabels uint64, B uint32) uint {
	return uint(math.Ceil((math.Log2(float64(numLabels)) - math.Log2(float64(B))) / 8))
}

// PowDifficulty returns the target difficulty of finding a nonce in `numLabels` labels.
// It is calculated such that a high percentage of smeshers find at least one computed label
// below the difficulty threshold. The difficulty is calculated as follows:
//
//	difficulty = 8 * 2^256 / numLabels
//
// The probability of finding a label below the difficulty threshold within numLabels
// approaches ~ 99.97% the bigger numLabels gets. Within 1.15 * numLabels the probability
// approaches 99.99% of finding at least one label below the difficulty threshold.
func PowDifficulty(numLabels uint64) []byte {
	x := new(big.Int).Lsh(big.NewInt(1), 256)
	x.Div(x, big.NewInt(int64(numLabels)))
	x.Lsh(x, 3) // reduce difficulty by a factor of 8

	difficulty := make([]byte, 32)
	return x.FillBytes(difficulty)
}

func Uint64MulOverflow(a, b uint64) bool {
	if a == 0 || b == 0 {
		return false
	}
	c := a * b
	return c/b != a
}

func BinaryRepresentationMinBits(val uint64) int {
	return int(math.Log2(float64(val))) + 1
}

func Size(itemBitSize uint, numItems uint) uint {
	bitSize := itemBitSize * numItems
	return (bitSize + 7) / 8 // Integer ceil of (indicesBitSize / 8).
}

// PutUintBE encodes a uint64 into a big-endian byte slice.
func PutUintBE(b []byte, v uint64) {
	numBits := len(b) * 8

	// Eliminate unnecessary MS bits.
	v <<= 64 - uint(numBits)

	for i := 0; i < len(b); i++ {
		b[i] = byte(v >> uint64(56-(8*i)))
	}
}

func UintBE(b []byte) uint64 {
	var v uint64
	for i := 0; i < len(b); i++ {
		v <<= 8
		v |= uint64(b[i])
	}
	return v
}

func UInt64LE(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
