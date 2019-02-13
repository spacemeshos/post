package merkle

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"post-private/datatypes"
	"testing"
)

func TestValidatePartialTree(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{3}
	leaves := []node{datatypes.NewLabel(3)}
	proof := []node{
		datatypes.NewLabel(0),
		datatypes.NewLabel(0),
		datatypes.NewLabel(0),
	}
	root, _ := hex.DecodeString("62b525ec807e21a1fd12d06905d85c4b7bc1feacfa57789d95702f6b69ce129f")
	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")
}

func TestValidatePartialTreeForRealz(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{4}
	leaves := []node{datatypes.NewLabel(4)}
	tree := NewProvingTree(8, leafIndices)
	for i := 0; i < 8; i++ {
		tree.AddLeaf(datatypes.NewLabel(uint64(i)))
	}
	root := tree.Root() // 4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0
	proof := tree.Proof() // 05 6b 13

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***********************************
	|                4a                |
	|       .13.             6c        |
	|    9d      fe      3d     .6b.   |
	|  00  01  02  03 =04=.05. 06  07  |
	***********************************/
}

func TestValidatePartialTreeMulti(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{1, 4}
	leaves := []node{
		datatypes.NewLabel(1),
		datatypes.NewLabel(4),
	}
	tree := NewProvingTree(8, leafIndices)
	for i := 0; i < 8; i++ {
		tree.AddLeaf(datatypes.NewLabel(uint64(i)))
	}
	root := tree.Root() // 4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0
	proof := tree.Proof() // 05 6b 13

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***********************************
	|                4a                |
	|        13              6c        |
	|    9d     .fe.     3d     .6b.   |
	| .00.=01= 02  03 =04=.05. 06  07  |
	***********************************/
}

func TestValidatePartialTreeMulti2(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{0, 1, 4}
	leaves := []node{
		datatypes.NewLabel(0),
		datatypes.NewLabel(1),
		datatypes.NewLabel(4),
	}
	tree := NewProvingTree(8, leafIndices)
	for i := 0; i < 8; i++ {
		tree.AddLeaf(datatypes.NewLabel(uint64(i)))
	}
	root := tree.Root() // 4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0
	proof := tree.Proof() // 05 6b 13

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***********************************
	|                4a                |
	|        13              6c        |
	|    9d     .fe.     3d     .6b.   |
	| =00==01= 02  03 =04=.05. 06  07  |
	***********************************/
}