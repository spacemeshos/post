package initialization

import (
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post-private/challenge"
	"github.com/spacemeshos/post-private/config"
	"github.com/spacemeshos/post-private/persistence"
	"github.com/spacemeshos/post-private/proving"
)

// at 8 bits per label, this would be 1 peta-byte of storage
const maxWidth = 1 << 50

// Initialize takes an id (public key), width (number of labels) and difficulty. Difficulty sets the number of bits per
// label that are stored. Each leaf in the tree is 32 bytes = 256 bits -- the number of bits per label is
// 256/(1<<difficulty). Supported values range from 5 (8 bits per label) to 8 (1 bit per label).
func Initialize(id []byte, width uint64, numberOfProvenLabels, difficulty uint8) (proof proving.Proof, err error) {
	if difficulty < 5 || difficulty > 8 {
		return proving.Proof{}, fmt.Errorf("difficulty must be between 5 and 8 (received %d)", difficulty)
	}

	labelsWriter, err := persistence.NewPostLabelsFileWriter(id)
	if err != nil {
		return proving.Proof{}, err
	}
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
		merkleRoot, (leafReader.Width()<<difficulty)-1, numberOfProvenLabels, difficulty,
	)

	proof.MerkleRoot = merkleRoot
	proof.ProvenIndices, proof.ProvenLeaves, proof.ProofNodes, err = merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return proving.Proof{}, err
	}

	return proof, err
}

func initialize(id []byte, width uint64, difficulty uint8, labelsWriter *persistence.PostLabelsFileWriter) (merkleRoot []byte,
	cacheReader *cache.Reader, err error) {

	if width > maxWidth {
		return nil, nil,
			fmt.Errorf("requested width (%d) is greater than supported width (%d)", width, maxWidth)
	}
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(7), cache.MakeSliceReadWriterFactory())
	merkleTree := merkle.NewTreeBuilder().
		WithHashFunc(challenge.ZeroChallenge.GetSha256Parent).
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
