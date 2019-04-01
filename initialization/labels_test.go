package initialization

import (
	"github.com/spacemeshos/post-private/proving"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCalcLabelGroupWholeByte(t *testing.T) {
	const difficulty = proving.Difficulty(5)
	id := []byte{0, 0, 0, 0}

	labels := make([]byte, 32)
	for i := 0; i < 32; i++ {
		labels[i] = CalcLabel(id, uint64(i), difficulty)
	}

	labelGroup := CalcLabelGroup(id, 0, difficulty)

	require.Equal(t, labels, labelGroup)
}

func TestCalcLabelGroupWholeByteWithOffset(t *testing.T) {
	const difficulty = proving.Difficulty(5)
	id := []byte{0, 0, 0, 0}

	labels := make([]byte, 32)
	for i := 32; i < 64; i++ {
		labels[i-32] = CalcLabel(id, uint64(i), difficulty)
	}

	labelGroup := CalcLabelGroup(id, 1, difficulty)

	require.Equal(t, labels, labelGroup)
}

func TestCalcLabelGroupHalfByte(t *testing.T) {
	const difficulty = proving.Difficulty(6)
	id := []byte{0, 0, 0, 0}

	labelGroup := CalcLabelGroup(id, 0, difficulty)

	require.Equal(t, hexDecode("51255c4d9310ef2d5846a0701a31104b53b43cdd19a6c6fcd33e92363af6c742"), labelGroup)

	//labels := make([]byte, 64)
	//for i := 0; i < 64; i++ {
	//	labels[i] = CalcLabel(id, uint64(i), difficulty)
	//}
	//
	//for _, label := range labels {
	//	fmt.Printf("%08b | %0"+strconv.Itoa(int(difficulty.LabelBits()))+"b\n", label,
	//		label&difficulty.LabelMask())
	//}
	//fmt.Println("-----------")
	//for _, label := range labelGroup {
	//	fmt.Printf("%08b\n", label)
	//}
	//
	//fmt.Println(hex.EncodeToString(labelGroup))
}

func TestCalcLabelGroupHalfByteWithOffset(t *testing.T) {
	const difficulty = proving.Difficulty(6)
	id := []byte{0, 0, 0, 0}

	labelGroup := CalcLabelGroup(id, 1, difficulty)

	require.Equal(t, hexDecode("b783ee4a02311c9a0dbbcf7b2b2d9e9145ed6ed8d07a50e6e061163a2764c19c"), labelGroup)

	//labels := make([]byte, 64)
	//for i := 64; i < 128; i++ {
	//	labels[i-64] = CalcLabel(id, uint64(i), difficulty)
	//}
	//
	//for _, label := range labels {
	//	fmt.Printf("%08b | %0"+strconv.Itoa(int(difficulty.LabelBits()))+"b\n", label,
	//		label&difficulty.LabelMask())
	//}
	//fmt.Println("-----------")
	//for _, label := range labelGroup {
	//	fmt.Printf("%08b\n", label)
	//}
	//
	//fmt.Println(hex.EncodeToString(labelGroup))
}

func TestCalcLabelGroupQuarterByte(t *testing.T) {
	const difficulty = proving.Difficulty(7)
	id := []byte{0, 0, 0, 0}

	labelGroup := CalcLabelGroup(id, 0, difficulty)

	require.Equal(t, hexDecode("594174b9428c6d437cc55a2c7e6eee32f3a22d461f3fb96519a44e4a896eb814"), labelGroup)

	//labels := make([]byte, 128)
	//for i := 0; i < 128; i++ {
	//	labels[i] = CalcLabel(id, uint64(i), difficulty)
	//}
	//
	//for _, label := range labels {
	//	fmt.Printf("%08b | %0"+strconv.Itoa(int(difficulty.LabelBits()))+"b\n", label,
	//		label&difficulty.LabelMask())
	//}
	//fmt.Println("-----------")
	//for _, label := range labelGroup {
	//	fmt.Printf("%08b\n", label)
	//}
	//
	//fmt.Println(hex.EncodeToString(labelGroup))
}
