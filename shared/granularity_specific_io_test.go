package shared_test

import (
	"bytes"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	NewGranSpecificReader = shared.NewGranSpecificReader
)

// change to simple array instead of file writing
func TestGranSpecificReader_BitGranular(t *testing.T) {
	req := require.New(t)

	// Write one byte ([0b11111111])
	buf := bytes.NewBuffer(nil)
	_, err := buf.Write([]byte{0xFF})
	req.NoError(err)

	// Read one bit.
	gsReader := NewGranSpecificReader(buf, uint(1))
	label, err := gsReader.ReadNext()
	req.NoError(err)
	req.Len(label, 1)
	req.Equal(byte(0x01), label[0])
}

func TestGranSpecificReader_ByteGranular(t *testing.T) {
	req := require.New(t)

	// Write two bytes ([0b11111111, 0b11111111])
	buf := bytes.NewBuffer(nil)
	_, err := buf.Write([]byte{0xFF, 0xFF})
	req.NoError(err)

	// Read 16 bits.
	gsReader := NewGranSpecificReader(buf, uint(16))
	label, err := gsReader.ReadNext()
	req.NoError(err)
	req.Len(label, 2)
	req.Equal([]byte{0xFF, 0xFF}, label)
}
