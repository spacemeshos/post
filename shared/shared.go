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

// PowDifficulty returns the target difficulty of finding a nonce in `numLabels` labels.
// It is calculated such that one computed label is expected to be below the difficulty threshold.
// The difficulty is calculated as follows:
//
//	difficulty = 2^256 / numLabels
//
// TODO(mafa): this difficulty calculation is unfit. There is only a ~ 63% chance to find a nonce
// in the first `numLabels`	labels and only a ~ 90% chance to find a nonce in the first `2*numLabels` labels.
func PowDifficulty(numLabels uint64) []byte {
	difficulty := make([]byte, 33)
	difficulty[0] = 0x01
	x := new(big.Int).SetBytes(difficulty)
	x.Div(x, big.NewInt(int64(numLabels)))
	return x.FillBytes(difficulty[1:])
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
