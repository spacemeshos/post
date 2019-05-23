package proving

import (
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"io"
	"math"
)

const (
	LabelGroupSize                          = shared.LabelGroupSize
	MaxSpace                                = shared.MaxSpace
	MaxNumOfFiles                           = shared.MaxNumOfFiles
	LowestLayerToCacheDuringProofGeneration = shared.LowestLayerToCacheDuringProofGeneration
)

type (
	Difficulty = shared.Difficulty
	Challenge  = shared.Challenge
)

func GenerateProof(id []byte, challenge Challenge, numOfProvenLabels uint8, difficulty Difficulty, dir string) (proof *Proof,
	err error) {

	proof, err = generateProof(id, challenge, numOfProvenLabels, difficulty, dir)
	if err != nil {
		err = fmt.Errorf("proof generation failed: %v", err)
		log.Error(err.Error())
	}
	return proof, err
}

func generateProof(id []byte, challenge Challenge, numOfProvenLabels uint8, difficulty Difficulty, dir string) (*Proof, error) {
	err := difficulty.Validate()
	if err != nil {
		return nil, err
	}

	proof := new(Proof)
	proof.Challenge = challenge
	proof.Identity = id

	reader, err := persistence.NewLabelsReader(dir)
	if err != nil {
		return nil, err
	}
	width, err := reader.Width()
	if err != nil {
		return nil, err
	}
	if width*difficulty.LabelsPerGroup() >= math.MaxUint64 {
		return nil, fmt.Errorf("leaf reader too big, number of label groups (%d) * labels per group (%d) "+
			"overflows uint64", width, difficulty.LabelsPerGroup())
	}
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(LowestLayerToCacheDuringProofGeneration),
		cache.MakeSliceReadWriterFactory())

	tree, err := merkle.NewTreeBuilder().WithHashFunc(challenge.GetSha256Parent).WithCacheWriter(cacheWriter).Build()
	if err != nil {
		return nil, err
	}
	for {
		leaf, err := reader.ReadNext()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		err = tree.AddLeaf(leaf)
		if err != nil {
			return nil, err
		}
	}
	proof.MerkleRoot = tree.Root()

	cacheWriter.SetLayer(0, reader)
	cacheReader, err := cacheWriter.GetReader()

	numOfLabels := width * difficulty.LabelsPerGroup()
	provenLeafIndices := CalcProvenLeafIndices(
		proof.MerkleRoot, numOfLabels, numOfProvenLabels, difficulty)

	_, proof.ProvenLeaves, proof.ProofNodes, err = merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return nil, err
	}

	return proof, nil
}
