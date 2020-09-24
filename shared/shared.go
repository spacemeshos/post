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
	dataSizeBits := numLabels * uint64(labelSize)
	dataSize := dataSizeBits / 8
	if dataSizeBits%8 > 0 {
		dataSize++
	}
	return dataSize
}

func NumLabels(dataSize uint64, labelSize uint) uint64 {
	dataSizeBits := dataSize * 8
	return dataSizeBits / uint64(labelSize)
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
