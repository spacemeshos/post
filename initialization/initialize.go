package initialization

import (
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
)

// Initialize takes an id (public key), space (in bytes) and difficulty.
// Difficulty determines the number of bits per label that are stored. Each leaf in the tree is 32 bytes = 256 bits.
// The number of bits per label is 256 / LabelsPerGroup. LabelsPerGroup = 1 << difficulty.
// Supported values range from 5 (8 bits per label) to 8 (1 bit per label).
func Initialize(id []byte, space proving.Space, numberOfProvenLabels uint8, difficulty proving.Difficulty) (
	proof proving.Proof, err error) {

	if err = space.Validate(LabelGroupSize); err != nil {
		return proving.Proof{}, err
	}
	if err = difficulty.Validate(); err != nil {
		return proving.Proof{}, err
	}

	proof.Challenge = proving.ZeroChallenge
	proof.Identity = id

	labelsWriter, err := persistence.NewPostLabelsFileWriter(id)
	if err != nil {
		return proving.Proof{}, err
	}

	width := uint64(space) / LabelGroupSize
	merkleRoot, cacheReader, err := initialize(id, width, difficulty, labelsWriter)
	if err2 := labelsWriter.Close(); err2 != nil {
		if err != nil {
			err = fmt.Errorf("%v, %v", err, err2)
		} else {
			err = err2
		}
	}
	if err != nil {
		err = fmt.Errorf("failed to initialize post: %v", err)
		log.Error(err.Error())
		return proving.Proof{}, err
	}

	leafReader := cacheReader.GetLayerReader(0)
	provenLeafIndices := proving.CalcProvenLeafIndices(
		merkleRoot, leafReader.Width()<<difficulty, numberOfProvenLabels, difficulty)

	proof.MerkleRoot = merkleRoot
	_, proof.ProvenLeaves, proof.ProofNodes, err = merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return proving.Proof{}, err
	}

	return proof, err
}

func initialize(id []byte, width uint64, difficulty proving.Difficulty,
	labelsWriter *persistence.PostLabelsFileWriter) (merkleRoot []byte, cacheReader *cache.Reader, err error) {

	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(proving.LowestLayerToCacheDuringProofGeneration), cache.MakeSliceReadWriterFactory())
	merkleTree := merkle.NewTreeBuilder().
		WithHashFunc(proving.ZeroChallenge.GetSha256Parent).
		WithCacheWriter(cacheWriter).
		Build()

	for position := uint64(0); position < width; position++ {
		lg := CalcLabelGroup(id, position, difficulty)
		err := labelsWriter.Write(lg)
		if err != nil {
			return nil, nil, err
		}
		err = merkleTree.AddLeaf(lg)
		if err != nil {
			return nil, nil, err
		}
		if (position+1)%config.Post.LogEveryXLabels == 0 {
			log.Info("found %v labels", position+1)
		}
	}

	log.With().Info("completed PoST label list construction")

	leafReader, err := labelsWriter.GetLeafReader()
	if err != nil {
		return nil, nil, err
	}
	cacheWriter.SetLayer(0, leafReader)
	cacheReader, err = cacheWriter.GetReader()
	if err != nil {
		return nil, nil, err
	}
	return merkleTree.Root(), cacheReader, nil
}
