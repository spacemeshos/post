package initialization

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
)

const (
	NumOfProvenLabels                       = shared.NumOfProvenLabels
	LabelGroupSize                          = shared.LabelGroupSize
	MaxSpace                                = shared.MaxSpace
	LowestLayerToCacheDuringProofGeneration = shared.LowestLayerToCacheDuringProofGeneration
)

type (
	Difficulty  = shared.Difficulty
	CacheReader = cache.CacheReader
)

var (
	ErrIdNotInitialized = errors.New("id not initialized")
)

// Initialize takes an id (public key), space (in bytes), numOfProvenLabels and difficulty.
// Difficulty determines the number of bits per label that are stored. Each leaf in the tree is 32 bytes = 256 bits.
// The number of bits per label is 256 / LabelsPerGroup. LabelsPerGroup = 1 << difficulty.
// Supported values range from 5 (8 bits per label) to 8 (1 bit per label).
func Initialize(id []byte, space uint64, numOfProvenLabels uint8, difficulty Difficulty, dir string, lograte uint64) (*proving.Proof, error) {
	return initialize(id, space, space, numOfProvenLabels, difficulty, dir, lograte)
}

func initialize(id []byte, space uint64, filesize uint64, numOfProvenLabels uint8, difficulty Difficulty, dir string, lograte uint64) (*proving.Proof, error) {
	if err := proving.ValidateSpace(space); err != nil {
		return nil, err
	}

	if err := difficulty.Validate(); err != nil {
		return nil, err
	}

	chunks, err := proving.NumOfFiles(space, filesize)
	if err != nil {
		return nil, err
	}
	labelGroupsPerChunk := proving.NumOfLabelGroups(filesize)

	results := make([]*initResult, chunks)
	for i := 0; i < chunks; i++ {
		result, err := initializeChunk(id, i, labelGroupsPerChunk, difficulty, dir, lograte)
		if err != nil {
			return nil, err
		}

		results[i] = result
	}

	result, err := merge(results)
	if err != nil {
		return nil, err
	}

	width, err := result.reader.GetLayerReader(0).Width()
	if err != nil {
		err = fmt.Errorf("failed to get leaves reader width: %v", err)
		log.Error(err.Error())
		return nil, err
	}

	provenLeafIndices := proving.CalcProvenLeafIndices(result.root, width<<difficulty, numOfProvenLabels, difficulty)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, result.reader)
	if err != nil {
		return nil, err
	}

	proof := &proving.Proof{
		Challenge:    shared.ZeroChallenge,
		Identity:     id,
		MerkleRoot:   result.root,
		ProvenLeaves: provenLeaves,
		ProofNodes:   proofNodes,
	}

	return proof, err
}

type initResult struct {
	reader CacheReader
	root   []byte
}

func initializeChunk(id []byte, chunkPosition int, labelGroupsPerChunk uint64, difficulty proving.Difficulty, dir string, lograte uint64) (*initResult, error) {
	// Initialize the labels file writer.
	labelsWriter, err := persistence.NewLabelsWriter(id, chunkPosition, dir)
	if err != nil {
		return nil, err
	}

	// Initialize the labels merkle tree with the execution-phase zero challenge.
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(LowestLayerToCacheDuringProofGeneration), cache.MakeSliceReadWriterFactory())
	tree, err := merkle.NewTreeBuilder().
		WithHashFunc(shared.ZeroChallenge.GetSha256Parent).
		WithCacheWriter(cacheWriter).
		Build()
	if err != nil {
		return nil, err
	}

	// Calculate labels in groups, write them to disk
	// and append them as leaves in the merkle tree.
	for position := uint64(0); position < labelGroupsPerChunk; position++ {
		offset := uint64(chunkPosition) * labelGroupsPerChunk
		lg := CalcLabelGroup(id, position+offset, difficulty)
		err := labelsWriter.Write(lg)
		if err != nil {
			return nil, err
		}
		err = tree.AddLeaf(lg)
		if err != nil {
			return nil, err
		}
		if (position+1)%lograte == 0 {
			log.Infof("completed %v labels", position+1)
		}
	}

	log.Info("completed PoST label list construction")

	labelsReader, err := labelsWriter.GetReader()
	if err != nil {
		return nil, err
	}

	if err := labelsWriter.Close(); err != nil {
		return nil, err
	}

	cacheWriter.SetLayer(0, labelsReader)
	cacheReader, err := cacheWriter.GetReader()
	if err != nil {
		return nil, err
	}

	return &initResult{reader: cacheReader, root: tree.Root()}, nil
}

func merge(results []*initResult) (*initResult, error) {
	switch len(results) {
	case 0:
		return nil, nil
	case 1:
		return results[0], nil
	default:
		readers := make([]CacheReader, len(results))
		for i, result := range results {
			readers[i] = result.reader
		}

		reader, err := cache.Merge(readers)
		if err != nil {
			return nil, err
		}

		reader, root, err := cache.BuildTop(reader)
		if err != nil {
			return nil, err
		}

		return &initResult{reader, root}, nil
	}
}

func Reset(dir string) (*persistence.ResetResult, error) {
	res, err := persistence.Reset(dir)
	if err != nil {
		if err == persistence.ErrDirNotFound {
			return nil, ErrIdNotInitialized
		}

		return nil, fmt.Errorf("reset failure: %v", err)
	}

	return res, nil
}
