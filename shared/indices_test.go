package shared

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCalcProvenLeafIndices(t *testing.T) {
	root, _ := hex.DecodeString("1cedc0ffee")
	leafIndices := CalcProvenLeafIndices(root, 129, 3, 5)
	require.EqualValues(t, SetOf(1, 3), leafIndices)
}

func TestConvertLabelIndicesToLeafIndices(t *testing.T) {
	r := require.New(t)

	leafIndices := ConvertLabelIndicesToLeafIndices(SetOf(0), 5)
	r.EqualValues(SetOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(15), 5)
	r.EqualValues(SetOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(31), 5)
	r.EqualValues(SetOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(32), 5)
	r.EqualValues(SetOf(1), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(63), 5)
	r.EqualValues(SetOf(1), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(64), 5)
	r.EqualValues(SetOf(2), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(31, 32, 63), 5)
	r.EqualValues(SetOf(0, 1), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(63), 6)
	r.EqualValues(SetOf(0), leafIndices)

	leafIndices = ConvertLabelIndicesToLeafIndices(SetOf(64), 6)
	r.EqualValues(SetOf(1), leafIndices)
}

func TestDrawProvenLabelIndices(t *testing.T) {
	r := require.New(t)

	root, _ := hex.DecodeString("1cedc0ffee")
	labelIndices := DrawProvenLabelIndices(root, 32, 5)
	r.EqualValues(SetOf(16, 17, 23, 27, 28), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 32, 7)
	r.EqualValues(SetOf(10, 16, 17, 22, 23, 27, 28), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 29, 7)
	r.EqualValues(SetOf(10, 16, 17, 22, 23, 27, 28), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 28, 7)
	r.EqualValues(SetOf(10, 16, 17, 19, 22, 23, 27), labelIndices)

	root, _ = hex.DecodeString("1cec0ffee")

	labelIndices = DrawProvenLabelIndices(root, 32, 5)
	r.EqualValues(SetOf(0, 3, 14, 16, 22), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 5, 5)
	r.EqualValues(SetOf(0, 1, 2, 3, 4), labelIndices)

	labelIndices = DrawProvenLabelIndices(root, 5, 6)
	r.Nil(labelIndices)
}
