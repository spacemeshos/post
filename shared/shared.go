package shared

import (
	"math"
	"os"
)

var (
	// OwnerReadWriteExec is a standard owner read / write / exec file permission.
	OwnerReadWriteExec = os.FileMode(0700)

	// OwnerReadWrite is a standard owner read / write file permission.
	OwnerReadWrite = os.FileMode(0600)
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

func Uint64MulOverflow(a, b uint64) bool {
	if a == 0 || b == 0 {
		return false
	}
	c := a * b
	return c/b != a
}

func NumBits(val uint64) int {
	return int(math.Log2(float64(val))) + 1
}

func Size(itemBitSize uint, numItems uint) uint {
	bitSize := itemBitSize * numItems
	return (bitSize + 7) / 8 // Integer ceil of (indicesBitSize / 8).
}

// PutUintBE
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
