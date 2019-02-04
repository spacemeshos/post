package merkle

import (
	"github.com/cbergoon/merkletree"
	"post-private/datatypes"
)

func CalcMerkleRoot(labels []datatypes.Label) []byte {
	contents := make([]merkletree.Content, len(labels))
	for i := range labels {
		contents[i] = labels[i]
	}
	merkleTree, err := merkletree.NewTree(contents)
	if err != nil {
		panic(err)
	}
	return merkleTree.Root.Hash
}
