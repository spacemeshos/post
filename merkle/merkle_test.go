package merkle

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"post-private/util"
	"testing"
)

/*

	8-leaf tree (1st byte of each node):

	+----------------------------------+
	|                4a                |
	|        13              6c        |
	|    9d      fe      3d      6b    |
	|  00  01  02  03  04  05  06  07  |
	+----------------------------------+

*/

func TestNewTree(t *testing.T) {
	tree := NewTree()
	for i := 0; i < 8; i++ {
		tree.AddLeaf(util.NewLabel(uint64(i)))
	}
	expectedRoot, _ := hex.DecodeString("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	require.Equal(t, expectedRoot, tree.Root())
}

func BenchmarkNewTree(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewTree()
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(util.NewLabel(i))
	}
	/*
		goos: darwin
		goarch: amd64
		pkg: post-private/merkle
		BenchmarkNewTree-8   	       1	94453361478 ns/op
		PASS
	*/
	// Overhead (no hashing) is 8056887277 ns/op (8 seconds)
	// 94 seconds to construct a 28 layer tree, 86 seconds without overhead (8.5GB @ 32b leaves).
	// Extrapolated to 256GB -> ~43 minutes + ~4 minutes overhead.
	// Reading 256GB from a magnetic disk should take ~30 minutes.
}

func TestNewProvingTree(t *testing.T) {
	tree := NewProvingTree([]uint64{4})
	for i := 0; i < 8; i++ {
		tree.AddLeaf(util.NewLabel(uint64(i)))
	}
	expectedRoot, _ := hex.DecodeString("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	require.Equal(t, expectedRoot, tree.Root())

	expectedProof := make([]node, 3)
	expectedProof[0], _ = hex.DecodeString("0500000000000000")
	expectedProof[1], _ = hex.DecodeString("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")
	expectedProof[2], _ = hex.DecodeString("13c04a6157aa640f711d230a4f04bc2b19e75df1127dfc899f025f3aa282912d")

	require.EqualValues(t, expectedProof, tree.Proof())

	/***********************************
	|                4a                |
	|       .13.             6c        |
	|    9d      fe      3d     .6b.   |
	|  00  01  02  03 =04=.05. 06  07  |
	***********************************/
}

func TestNewProvingTreeMultiProof(t *testing.T) {
	tree := NewProvingTree([]uint64{1, 4})
	for i := 0; i < 8; i++ {
		tree.AddLeaf(util.NewLabel(uint64(i)))
	}
	expectedRoot, _ := hex.DecodeString("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	require.Equal(t, expectedRoot, tree.Root())

	expectedProof := make([]node, 4)
	expectedProof[0], _ = hex.DecodeString("0000000000000000")
	expectedProof[1], _ = hex.DecodeString("fe6d3d3bb5dd778af1128cc7b2b33668d51b9a52dfc8f2342be37ddc06a0072d")
	expectedProof[2], _ = hex.DecodeString("0500000000000000")
	expectedProof[3], _ = hex.DecodeString("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")

	require.EqualValues(t, expectedProof, tree.Proof())

	/***********************************
	|                4a                |
	|        13              6c        |
	|    9d     .fe.     3d     .6b.   |
	| .00.=01= 02  03 =04=.05. 06  07  |
	***********************************/
}

func TestNewProvingTreeMultiProof2(t *testing.T) {
	tree := NewProvingTree([]uint64{0, 1, 4})
	for i := 0; i < 8; i++ {
		tree.AddLeaf(util.NewLabel(uint64(i)))
	}
	expectedRoot, _ := hex.DecodeString("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	require.Equal(t, expectedRoot, tree.Root())

	expectedProof := make([]node, 3)
	expectedProof[0], _ = hex.DecodeString("fe6d3d3bb5dd778af1128cc7b2b33668d51b9a52dfc8f2342be37ddc06a0072d")
	expectedProof[1], _ = hex.DecodeString("0500000000000000")
	expectedProof[2], _ = hex.DecodeString("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")

	require.EqualValues(t, expectedProof, tree.Proof())

	/***********************************
	|                4a                |
	|        13              6c        |
	|    9d     .fe.     3d     .6b.   |
	| =00==01= 02  03 =04=.05. 06  07  |
	***********************************/
}
