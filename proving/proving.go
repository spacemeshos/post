package proving

import (
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post-private/persistence"
	"io"
)

func GenerateProof(id []byte, challenge Challenge, numberOfProvenLabels uint8, difficulty Difficulty) (proof Proof,
	err error) {

	proof.Challenge = challenge
	proof.Identity = id

	leafReader, err := persistence.NewLeafReader(id)
	if err != nil {
		return Proof{}, err
	}
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(7), cache.MakeSliceReadWriterFactory())

	tree := merkle.NewTreeBuilder().WithHashFunc(challenge.GetSha256Parent).WithCacheWriter(cacheWriter).Build()
	for {
		leaf, err := leafReader.ReadNext()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Proof{}, err
		}
		err = tree.AddLeaf(leaf)
		if err != nil {
			return Proof{}, err
		}
	}
	proof.MerkleRoot = tree.Root()

	cacheWriter.SetLayer(0, leafReader)
	cacheReader, err := cacheWriter.GetReader()

	provenLeafIndices := CalcProvenLeafIndices(
		proof.MerkleRoot, leafReader.Width()*difficulty.LabelsPerGroup(), numberOfProvenLabels, difficulty)

	_, proof.ProvenLeaves, proof.ProofNodes, err = merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return Proof{}, err
	}

	return proof, nil
}
