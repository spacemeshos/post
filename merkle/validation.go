package merkle

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

const MaxUint = ^uint(0)

func ValidatePartialTree(leafIndices []uint64, leaves, proof []Node, expectedRoot Node) (bool, error) {
	v, err := newValidator(leafIndices, leaves, proof)
	if err != nil {
		return false, err
	}
	root := v.calcRoot(MaxUint)
	return bytes.Equal(root, expectedRoot), nil
}

func newValidator(leafIndices []uint64, leaves, proof []Node) (validator, error) {
	if len(leafIndices) != len(leaves) {
		return validator{}, fmt.Errorf("number of leaves (%d) must equal number of indices (%d)", len(leaves), len(leafIndices))
	}
	if len(leaves) == 0 {
		return validator{}, fmt.Errorf("at least one leaf is required for validation")
	}
	if len(leaves)+len(proof) == 1 {
		return validator{}, fmt.Errorf("tree of size 1 not supported")
	}
	proofNodes := &proofIterator{proof}
	leafIt := &leafIterator{leafIndices, leaves}

	return validator{leafIt, proofNodes}, nil
}

type validator struct {
	leaves     *leafIterator
	proofNodes *proofIterator
}

func (v *validator) calcRoot(stopAtLayer uint) Node {
	layer := uint(0)
	idx, activeNode, err := v.leaves.next()
	if err != nil {
		panic(err) // this should never happen since we verify there are more leaves before calling calcRoot
	}
	var leftChild, rightChild, sibling Node
	println()
	for {
		if layer == stopAtLayer {
			break
		}
		if v.shouldCalcSubtree(idx, layer) {
			sibling = v.calcRoot(layer)
		} else {
			var err error
			sibling, err = v.proofNodes.next()
			if err == noMoreItems {
				break
			}
		}
		if leftSibling(idx, layer) {
			leftChild, rightChild = sibling, activeNode
		} else {
			leftChild, rightChild = activeNode, sibling
		}
		activeNode = getParent(leftChild, rightChild)
		layer++
		fmt.Println(leftChild, " + ", rightChild, " => ", activeNode)
	}
	fmt.Println(hex.EncodeToString(activeNode))
	println()
	return activeNode
}

func leftSibling(idx uint64, layer uint) bool {
	return (idx/(1<<layer))%2 == 1
}

func (v *validator) shouldCalcSubtree(idx uint64, layer uint) bool {
	nextIdx, err := v.leaves.peek()
	if err == noMoreItems {
		return false
	}
	return (idx^nextIdx)/(1<<layer) == 1
}

var noMoreItems = errors.New("no more items")

type proofIterator struct {
	nodes []Node
}

func (it *proofIterator) next() (Node, error) {
	if len(it.nodes) == 0 {
		return nil, noMoreItems
	}
	n := it.nodes[0]
	it.nodes = it.nodes[1:]
	return n, nil
}

type leafIterator struct {
	indices []uint64
	leaves  []Node
}

func (it *leafIterator) next() (uint64, Node, error) {
	if len(it.indices) == 0 {
		return 0, nil, noMoreItems
	}
	idx := it.indices[0]
	leaf := it.leaves[0]
	it.indices = it.indices[1:]
	it.leaves = it.leaves[1:]
	return idx, leaf, nil
}

func (it *leafIterator) peek() (uint64, error) {
	if len(it.indices) == 0 {
		return 0, noMoreItems
	}
	idx := it.indices[0]
	return idx, nil
}
