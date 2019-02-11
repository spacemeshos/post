package merkle

import (
	"encoding/hex"
	"fmt"
	"github.com/spacemeshos/sha256-simd"
	"math"
	"post-private/datatypes"
)

type Tree interface {
	AddLeaf(label datatypes.Label)
	Root() []byte
	Proof() []node
}

type node []byte

func (n node) String() string {
	return hex.EncodeToString(n)[:4]
}

type incrementalTree struct {
	path        []node
	currentLeaf uint64
	leafToProve *uint64
	proof       []node
	nodes       [][]node // TODO @noam: Remove!
}

func NewTree(width uint64) Tree {
	depth := int(math.Log2(float64(width))) + 1
	return &incrementalTree{
		path:        make([]node, depth),
		currentLeaf: 0,
		leafToProve: nil,
		proof:       nil,
		nodes:       make([][]node, depth, width), // TODO @noam: Remove!
	}
}

func NewTreeWithProof(width, leafToProve uint64) Tree {
	depth := int(math.Log2(float64(width))) + 1
	return &incrementalTree{
		path:        make([]node, depth),
		currentLeaf: 0,
		leafToProve: &leafToProve,
		proof:       make([]node, depth-1),
		nodes:       make([][]node, depth, width), // TODO @noam: Remove!
	}
}

func (t *incrementalTree) AddLeaf(label datatypes.Label) {
	activeNode := node(label)
	for i := range t.path {
		t.nodes[i] = append(t.nodes[i], activeNode) // TODO @noam: Remove!
		if t.isNodeInProof(uint(i)) {
			t.proof[i] = activeNode
		}
		if t.path[i] == nil {
			t.path[i] = activeNode
			break
		}
		activeNode = sum(t.path[i], activeNode)
		t.path[i] = nil
	}
	t.currentLeaf++
}

func sum(left node, right node) node {
	res := sha256.Sum256(append(left, right...))
	return res[:]
}

func (t *incrementalTree) Root() []byte {
	return t.path[len(t.path)-1]
}

func (t *incrementalTree) Proof() []node {
	printTree(t.nodes) // TODO @noam: Remove!
	return t.proof
}

func (t *incrementalTree) isNodeInProof(layer uint) bool {
	if t.leafToProve == nil {
		return false
	}

	pathDiff := t.currentLeaf ^ *t.leafToProve
	samePathAboveCurrentLayer := pathDiff/(1<<(layer+1)) == 0
	differentAtCurrentLayer := pathDiff/(1<<layer)%2 == 1

	return samePathAboveCurrentLayer && differentAtCurrentLayer

	/* Explanation:

	The index in binary form (most- to least-significant) represents the path from the top (root) of the tree to the
	bottom (0=left, 1=right).
	We require that the path from the root to the current leaf and the path to the proved leaf are identical up to the
	current layer.
	We also require that the current layer is different - so the currently handled node is a sibling of one of the nodes
	in the path to the proved node -- so we want it in the proof.

	*/
}

// TODO @noam: Remove!
func printTree(nodes [][]node) {
	for _, n := range nodes {
		defer fmt.Println(n)
	}
}
