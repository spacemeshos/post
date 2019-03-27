package proving

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCalcProvenLeafIndices(t *testing.T) {
	root, _ := hex.DecodeString("1cedc0ffee")
	leafIndices := CalcProvenLeafIndices(root, 128, 3, 5)
	require.EqualValues(t, setOf(1, 3), leafIndices)
}

func TestConvertLabelIndicesToLeafIndices(t *testing.T) {
	r := require.New(t)

	leafIndices := ConvertLabelIndicesToLeafIndices(setOf(0), 5)
	r.EqualValues(setOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(15), 5)
	r.EqualValues(setOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(31), 5)
	r.EqualValues(setOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(32), 5)
	r.EqualValues(setOf(1), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(63), 5)
	r.EqualValues(setOf(1), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(64), 5)
	r.EqualValues(setOf(2), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(31, 32, 63), 5)
	r.EqualValues(setOf(0, 1), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(63), 6)
	r.EqualValues(setOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(setOf(64), 6)
	r.EqualValues(setOf(1), leafIndices)
}

func TestDrawProvenLabelIndices(t *testing.T) {
	r := require.New(t)

	root, _ := hex.DecodeString("1cedc0ffee")
	labelIndices := DrawProvenLabelIndices(root, 31, 5)
	r.EqualValues(setOf(16, 17, 23, 27, 28), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 31, 7)
	r.EqualValues(setOf(10, 16, 17, 22, 23, 27, 28), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 28, 7)
	r.EqualValues(setOf(10, 16, 17, 22, 23, 27, 28), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 27, 7)
	r.EqualValues(setOf(10, 16, 17, 19, 22, 23, 27), labelIndices)

	root, _ = hex.DecodeString("1cec0ffee")

	labelIndices = DrawProvenLabelIndices(root, 31, 5)
	r.EqualValues(setOf(0, 3, 14, 16, 22), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 4, 5)
	r.EqualValues(setOf(0, 1, 2, 3, 4), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 4, 6)
	r.Nil(labelIndices)
}

func setOf(members ...uint64) map[uint64]bool {
	ret := make(map[uint64]bool)
	for _, member := range members {
		ret[member] = true
	}
	return ret
}
