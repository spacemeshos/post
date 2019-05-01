package proving

import (
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/persistence"
	"io"
	"math"
)

// NumOfProvenLabels is the recommended setting for this argument to ensure proof safety.
const NumOfProvenLabels = 100

const LowestLayerToCacheDuringProofGeneration = 11

func GenerateProof(id []byte, challenge Challenge, numOfProvenLabels uint8, difficulty Difficulty) (proof Proof,
	err error) {

	proof, err = generateProof(id, challenge, numOfProvenLabels, difficulty)
	if err != nil {
		err = fmt.Errorf("proof generation failed: %v", err)
		log.Error(err.Error())
	}
	return proof, err
}

func generateProof(id []byte, challenge Challenge, numOfProvenLabels uint8, difficulty Difficulty) (proof Proof,
	err error) {

	err = difficulty.Validate()
	if err != nil {
		return Proof{}, err
	}

	proof.Challenge = challenge
	proof.Identity = id

	leafReader, err := persistence.NewLeafReader(id)
	if err != nil {
		return Proof{}, err
	}
	if leafReader.Width()*difficulty.LabelsPerGroup() >= math.MaxUint64 {
		return Proof{}, fmt.Errorf("leaf reader too big, number of label groups (%d) * labels per group (%d) "+
			"overflows uint64", leafReader.Width(), difficulty.LabelsPerGroup())
	}
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(LowestLayerToCacheDuringProofGeneration),
		cache.MakeSliceReadWriterFactory())

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

	numOfLabels := leafReader.Width() * difficulty.LabelsPerGroup()
	provenLeafIndices := CalcProvenLeafIndices(
		proof.MerkleRoot, numOfLabels, numOfProvenLabels, difficulty)

	_, proof.ProvenLeaves, proof.ProofNodes, err = merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return Proof{}, err
	}

	return proof, nil
}
