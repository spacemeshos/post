package merkle

import (
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
}

func NewTree(width uint64) Tree {
	return &incrementalTree{path: make([]node, int(math.Log2(float64(width)))+1)}
}

func (t incrementalTree) AddLeaf(label datatypes.Label) {
	activeNode := node(label)
	for i, n := range t.path {
		if n == nil {
			t.path[i] = activeNode
			break
		}
		activeNode = sum(n, activeNode)
		t.path[i] = nil
	}
}

func sum(left node, right node) node {
	res := sha256.Sum256(append(left, right...))
	return res[:]
}

func (t incrementalTree) Root() []byte {
	return t.path[len(t.path)-1]
}
