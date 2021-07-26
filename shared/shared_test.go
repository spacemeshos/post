package shared

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUintBE(t *testing.T) {
	req := require.New(t)

	for i := 0; i < 8; i++ {
		var v uint64 = 1 << uint64(i*8)
		b := make([]byte, i+1)
		PutUintBE(b, v)
		req.Equal(v, UintBE(b))
	}
}

func TestDataSize(t *testing.T) {
	req := require.New(t)

	req.Equal(uint64(0), DataSize(0, 8))

	req.Equal(uint64(0), DataSize(1, 0))
	req.Equal(uint64(1), DataSize(1, 1))
	req.Equal(uint64(1), DataSize(1, 4))
	req.Equal(uint64(1), DataSize(1, 5))
	req.Equal(uint64(1), DataSize(1, 8))
	req.Equal(uint64(2), DataSize(1, 10))

	req.Equal(uint64(0), DataSize(2, 0))
	req.Equal(uint64(1), DataSize(2, 1))
	req.Equal(uint64(1), DataSize(2, 4))
	req.Equal(uint64(2), DataSize(2, 5))
	req.Equal(uint64(2), DataSize(2, 8))
	req.Equal(uint64(3), DataSize(2, 10))
}

func TestNumLabels(t *testing.T) {
	req := require.New(t)

	req.Equal(uint64(0), NumLabels(0, 8))

	req.Equal(uint64(8), NumLabels(1, 1))
	req.Equal(uint64(2), NumLabels(1, 4))
	req.Equal(uint64(1), NumLabels(1, 5))
	req.Equal(uint64(1), NumLabels(1, 8))
	req.Equal(uint64(0), NumLabels(1, 10))

	req.Equal(uint64(16), NumLabels(2, 1))
	req.Equal(uint64(4), NumLabels(2, 4))
	req.Equal(uint64(3), NumLabels(2, 5))
	req.Equal(uint64(2), NumLabels(2, 8))
	req.Equal(uint64(1), NumLabels(2, 10))
}

func TestNumLabelsDataSize(t *testing.T) {
	req := require.New(t)

	dataSize := uint64(1)
	labelSize := uint(1)
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 1
	labelSize = 4
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 1
	labelSize = 5
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 1
	labelSize = 8
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 2
	labelSize = 1
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 2
	labelSize = 4
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 2
	labelSize = 5
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 2
	labelSize = 8
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))

	dataSize = 2
	labelSize = 10
	req.Equal(dataSize, DataSize(NumLabels(dataSize, labelSize), labelSize))
}
