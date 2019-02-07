package merkle

import (
	"github.com/cbergoon/merkletree"
	"github.com/spacemeshos/sha256-simd"
	"math"
	"post-private/datatypes"
)

type Tree interface {
	AddLeaf(label datatypes.Label)
	Root() []byte
}

type node []byte

type incrementalTree struct {
	path []node
	hash func(left node, right node) node
}

func NewTree(width uint64) Tree {
	return &incrementalTree{path: make([]node, int(math.Log2(float64(width)))+1), hash: sha256Hash}
}

func (t incrementalTree) AddLeaf(label datatypes.Label) {
	activeNode := node(label)
	for i, n := range t.path {
		if n == nil {
			t.path[i] = activeNode
			break
		}
		activeNode = t.hash(n, activeNode)
		t.path[i] = nil
	}
}

func sha256Hash(left node, right node) node {
	res := sha256.Sum256(append(left, right...))
	return res[:]
}

func (t incrementalTree) Root() []byte {
	return t.path[len(t.path)-1]
}

type batchTree struct {
	labels []merkletree.Content
}

func NewBatchTree(width uint64) Tree {
	return &batchTree{labels: make([]merkletree.Content, 0, width)}
}

func (t *batchTree) AddLeaf(label datatypes.Label) {
	t.labels = append(t.labels, label)
}

func (t *batchTree) Root() []byte {
	return calcMerkleRoot(t.labels)
}

func calcMerkleRoot(labels []merkletree.Content) []byte {
	merkleTree, err := merkletree.NewTree(labels)
	if err != nil {
		panic(err)
	}
	return merkleTree.Root.Hash
}
