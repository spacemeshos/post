package merkle

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"math"
	"post-private/datatypes"
	"testing"
)

func TestNewTree(t *testing.T) {
	tree := NewTree(8)
	for i := 0; i < 8; i++ {
		tree.AddLeaf(datatypes.NewLabel(uint64(i)))
	}
	expectedRoot, _ := hex.DecodeString("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	require.Equal(t, expectedRoot, tree.Root())
}

func _TestNewTreeBig(t *testing.T) {
	size := uint64(math.Pow(2, 25))
	tree := NewTree(size)
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(datatypes.NewLabel(i))
	}
	expectedRoot, _ := hex.DecodeString("d359afe256ea0864223601b064d334ee3667923479a24a2df2daea31936d3779")
	require.Equal(t, expectedRoot, tree.Root())
	/*
		=== RUN   TestNewTreeBig
		--- PASS: TestNewTreeBig (11.90s)
	*/
}

func TestNewTreeWithProof(t *testing.T) {
	tree := NewTreeWithProof(8, 4)
	for i := 0; i < 8; i++ {
		tree.AddLeaf(datatypes.NewLabel(uint64(i)))
	}
	expectedRoot, _ := hex.DecodeString("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	require.Equal(t, expectedRoot, tree.Root())

	expectedProof := make([]node, 3)
	expectedProof[0], _ = hex.DecodeString("0500000000000000")
	expectedProof[1], _ = hex.DecodeString("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")
	expectedProof[2], _ = hex.DecodeString("13c04a6157aa640f711d230a4f04bc2b19e75df1127dfc899f025f3aa282912d")

	require.EqualValues(t, expectedProof, tree.Proof())
}
