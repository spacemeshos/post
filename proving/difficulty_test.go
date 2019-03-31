package proving

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDifficulty_LabelsPerByte(t *testing.T) {
	r := require.New(t)
	r.Equal(uint64(1), Difficulty(5).LabelsPerByte())
	r.Equal(uint64(2), Difficulty(6).LabelsPerByte())
	r.Equal(uint64(4), Difficulty(7).LabelsPerByte())
	r.Equal(uint64(8), Difficulty(8).LabelsPerByte())
}

func TestDifficulty_LabelsPerGroup(t *testing.T) {
	r := require.New(t)
	r.Equal(uint64(32), Difficulty(5).LabelsPerGroup())
	r.Equal(uint64(64), Difficulty(6).LabelsPerGroup())
	r.Equal(uint64(128), Difficulty(7).LabelsPerGroup())
	r.Equal(uint64(256), Difficulty(8).LabelsPerGroup())
}

func TestDifficulty_LabelBits(t *testing.T) {
	r := require.New(t)
	r.Equal(uint64(8), Difficulty(5).LabelBits())
	r.Equal(uint64(4), Difficulty(6).LabelBits())
	r.Equal(uint64(2), Difficulty(7).LabelBits())
	r.Equal(uint64(1), Difficulty(8).LabelBits())
}

func TestDifficulty_LabelMask(t *testing.T) {
	r := require.New(t)
	r.Equal(uint8(0xFF), Difficulty(5).LabelMask()) // 1111 1111
	r.Equal(uint8(0x0F), Difficulty(6).LabelMask()) // 0000 1111
	r.Equal(uint8(0x03), Difficulty(7).LabelMask()) // 0000 0011
	r.Equal(uint8(0x01), Difficulty(8).LabelMask()) // 0000 0001
}

func TestDifficulty_ByteIndex(t *testing.T) {
	r := require.New(t)
	r.Equal(uint64(8), Difficulty(5).ByteIndex(8))
	r.Equal(uint64(4), Difficulty(6).ByteIndex(8))
	r.Equal(uint64(2), Difficulty(7).ByteIndex(8))
	r.Equal(uint64(1), Difficulty(8).ByteIndex(8))
	r.Equal(uint64(1), Difficulty(7).ByteIndex(4))
	r.Equal(uint64(1), Difficulty(6).ByteIndex(2))
	r.Equal(uint64(1), Difficulty(5).ByteIndex(1))
}

func TestDifficulty_LeafIndex(t *testing.T) {
	r := require.New(t)
	r.Equal(uint64(0), Difficulty(5).LeafIndex(32-1))
	r.Equal(uint64(0), Difficulty(6).LeafIndex(64-1))
	r.Equal(uint64(0), Difficulty(7).LeafIndex(128-1))
	r.Equal(uint64(0), Difficulty(8).LeafIndex(256-1))

	r.Equal(uint64(1), Difficulty(5).LeafIndex(32))
	r.Equal(uint64(1), Difficulty(6).LeafIndex(64))
	r.Equal(uint64(1), Difficulty(7).LeafIndex(128))
	r.Equal(uint64(1), Difficulty(8).LeafIndex(256))

	r.Equal(uint64(2), Difficulty(5).LeafIndex(2*32))
	r.Equal(uint64(2), Difficulty(6).LeafIndex(2*64))
	r.Equal(uint64(2), Difficulty(7).LeafIndex(2*128))
	r.Equal(uint64(2), Difficulty(8).LeafIndex(2*256))
}

func TestDifficulty_IndexInLeaf(t *testing.T) {
	r := require.New(t)
	r.Equal(uint64(1), Difficulty(5).IndexInLeaf(32+1))
	r.Equal(uint64(3), Difficulty(6).IndexInLeaf(64+2+1))
	r.Equal(uint64(5), Difficulty(7).IndexInLeaf(128+4+1))
	r.Equal(uint64(9), Difficulty(8).IndexInLeaf(256+8+1))
}

func TestDifficulty_IndexInByte(t *testing.T) {
	r := require.New(t)
	r.Equal(uint64(0), Difficulty(5).IndexInByte(32+1))
	r.Equal(uint64(1), Difficulty(6).IndexInByte(64+2+1))
	r.Equal(uint64(1), Difficulty(7).IndexInByte(128+4+1))
	r.Equal(uint64(1), Difficulty(8).IndexInByte(256+8+1))
}

func TestDifficulty_Validate(t *testing.T) {
	r := require.New(t)
	r.EqualError(Difficulty(4).Validate(), "difficulty must be between 5 and 8 (received 4)")
	r.NoError(Difficulty(5).Validate())
	r.NoError(Difficulty(8).Validate())
	r.EqualError(Difficulty(9).Validate(), "difficulty must be between 5 and 8 (received 9)")
}
