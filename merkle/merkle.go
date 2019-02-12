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
	path          []node
	currentLeaf   uint64
	leavesToProve []uint64
	proof         []node
	nodes         [][]node // TODO @noam: Remove!
}

func NewTree(width uint64) Tree {
	depth := int(math.Log2(float64(width))) + 1
	return &incrementalTree{
		path:        make([]node, depth),
		currentLeaf: 0,
		nodes:       make([][]node, depth), // TODO @noam: Remove!
	}
}

func NewProvingTree(width uint64, leavesToProve []uint64) Tree {
	depth := int(math.Log2(float64(width))) + 1
	return &incrementalTree{
		path:          make([]node, depth),
		currentLeaf:   0,
		leavesToProve: leavesToProve,
		proof:         make([]node, 0, (depth-1)*len(leavesToProve)), // upper bound can be made much tighter
		nodes:         make([][]node, depth),                         // TODO @noam: Remove!
	}
}

func (t *incrementalTree) AddLeaf(label datatypes.Label) {
	activeNode := node(label)
	for i := range t.path {
		if len(t.path) < 5 {
			t.nodes[i] = append(t.nodes[i], activeNode) // TODO @noam: Remove!
		}
		if t.path[i] == nil {
			t.path[i] = activeNode
			break
		}
		t.addToProofIfNeeded(uint(i), t.path[i], activeNode)
		activeNode = getParent(t.path[i], activeNode)
		t.path[i] = nil
	}
	t.currentLeaf++
}

func (t *incrementalTree) addToProofIfNeeded(currentLayer uint, leftChild, rightChild node) {
	if len(t.leavesToProve) == 0 {
		return
	}
	parentPath, leftChildPath, rightChildPath := getPaths(t.currentLeaf, currentLayer)
	if t.isNodeInProvedPath(parentPath, currentLayer+1) {
		if !t.isNodeInProvedPath(leftChildPath, currentLayer) {
			t.proof = append(t.proof, leftChild)
		}
		if !t.isNodeInProvedPath(rightChildPath, currentLayer) {
			t.proof = append(t.proof, rightChild)
		}
	}
}

func getPaths(currentLeaf uint64, layer uint) (parentPath, leftChildPath, rightChildPath uint64) {
	parentPath = currentLeaf / (1 << (layer + 1))
	return parentPath, parentPath << 1, parentPath<<1 + 1
}

func getParent(leftChild node, rightChild node) node {
	res := sha256.Sum256(append(leftChild, rightChild...))
	return res[:]
}

func (t *incrementalTree) Root() []byte {
	return t.path[len(t.path)-1]
}

func (t *incrementalTree) Proof() []node {
	if len(t.path) < 5 {
		printTree(t.nodes) // TODO @noam: Remove!
	}
	return t.proof
}

func (t *incrementalTree) isNodeInProvedPath(path uint64, layer uint) bool {
	var divisor uint64 = 1 << layer
	for _, leafToProve := range t.leavesToProve {
		if leafToProve/divisor == path {
			return true
		}
	}
	return false
}

// TODO @noam: Remove!
func printTree(nodes [][]node) {
	for _, n := range nodes {
		defer fmt.Println(n)
	}
}
