package shared

import (
	"testing"

	"github.com/spacemeshos/go-scale/tester"
	"github.com/stretchr/testify/require"
)

func TestProof_Encode_Decode(t *testing.T) {
	req := require.New(t)

	vBase := Proof{
		Nonce:   256,
		Indices: makeNonEmptyBytes(100),
	}
	v := Proof{}
	err := v.Decode(vBase.Encode())
	req.NoError(err)
	req.Equal(vBase, v)
}

func makeNonEmptyBytes(size int) []byte {
	b := make([]byte, size)
	for i := 0; i < size; i++ {
		b[0] = byte(i) // // Assign some arbitrary value.
	}
	return b
}

func FuzzProofConsistency(f *testing.F) {
	tester.FuzzConsistency[Proof](f)
}

func FuzzMerkleProofSafety(f *testing.F) {
	tester.FuzzSafety[Proof](f)
}
