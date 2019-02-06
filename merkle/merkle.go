package merkle

import (
	"github.com/cbergoon/merkletree"
	"post-private/datatypes"
)

type Tree interface {
	AddLeaf(label datatypes.Label)
	Root() []byte
}

type merkleTree struct {
	labels []merkletree.Content
}

func NewTree(width uint64) Tree {
	return &merkleTree{labels: make([]merkletree.Content, 0, width)}
}

func (t *merkleTree) AddLeaf(label datatypes.Label) {
	t.labels = append(t.labels, label)
}

func (t *merkleTree) Root() []byte {
	return calcMerkleRoot(t.labels)
}

func calcMerkleRoot(labels []merkletree.Content) []byte {
	merkleTree, err := merkletree.NewTree(labels)
	if err != nil {
		panic(err)
	}
	return merkleTree.Root.Hash
}
