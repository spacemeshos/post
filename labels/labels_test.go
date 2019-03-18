package labels

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCalcLabelGroup(t *testing.T) {
	id := []byte{0, 0, 0, 0}
	label := CalcLabelGroup(id, 0)

	println(hex.EncodeToString(label))

	expectedLabel, _ := hex.DecodeString("dd9c96ebfe6d5ee548fd35d2b8c75b6fb85f633936a00841e5459c98c09b0653")
	require.Equal(t, expectedLabel, label)
}
