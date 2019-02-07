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

func TestNewBatchTree(t *testing.T) {
	tree := NewBatchTree(8)
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

func _TestNewBatchTreeBig(t *testing.T) {
	size := uint64(math.Pow(2, 25))
	tree := NewBatchTree(size)
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(datatypes.NewLabel(i))
	}
	expectedRoot, _ := hex.DecodeString("d359afe256ea0864223601b064d334ee3667923479a24a2df2daea31936d3779")
	require.Equal(t, expectedRoot, tree.Root())
	/*
	=== RUN   TestNewBatchTreeBig
	--- PASS: TestNewBatchTreeBig (58.90s)
	*/
}
