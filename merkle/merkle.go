package merkle

import (
	"fmt"
	"github.com/spacemeshos/sha256-simd"
)

type Tree interface {
	AddLeaf(leaf Node)
	Root() Node
	Proof() []Node
}

type incrementalTree struct {
	path          []Node
	currentLeaf   uint64
	leavesToProve []uint64
	proof         []Node
	nodes         [][]Node // TODO @noam: Remove!
}

func NewTree() Tree {
	return &incrementalTree{
		path:        make([]Node, 0),
		currentLeaf: 0,
		nodes:       make([][]Node, 0), // TODO @noam: Remove!
	}
}

func NewProvingTree(leavesToProve []uint64) Tree {
	return &incrementalTree{
		path:          make([]Node, 0),
		currentLeaf:   0,
		leavesToProve: leavesToProve,
		proof:         make([]Node, 0),
		nodes:         make([][]Node, 0), // TODO @noam: Remove!
	}
}

func (t *incrementalTree) AddLeaf(leaf Node) {
	activeNode := leaf
	for i := 0; true; i++ {
		if len(t.path) == i {
			t.path = append(t.path, nil)
		}
		if len(t.path) < 5 {
			if len(t.nodes) == i {
				t.nodes = append(t.nodes, nil)
			}
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

func (t *incrementalTree) addToProofIfNeeded(currentLayer uint, leftChild, rightChild Node) {
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

func getParent(leftChild, rightChild Node) Node {
	res := sha256.Sum256(append(leftChild, rightChild...))
	return res[:]
}

func (t *incrementalTree) Root() Node {
	return t.path[len(t.path)-1]
}

func (t *incrementalTree) Proof() []Node {
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
func printTree(nodes [][]Node) {
	for _, n := range nodes {
		defer fmt.Println(n)
	}
}
