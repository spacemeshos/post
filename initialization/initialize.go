package initialization

import (
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"runtime"
)

const (
	NumOfProvenLabels                       = shared.NumOfProvenLabels
	LabelGroupSize                          = shared.LabelGroupSize
	MaxSpace                                = shared.MaxSpace
	LowestLayerToCacheDuringProofGeneration = shared.LowestLayerToCacheDuringProofGeneration
)

type (
	CacheReader = cache.CacheReader
)

// Initialize takes an id (public key), space (in bytes), numOfProvenLabels and difficulty.
// Difficulty determines the number of bits per label that are stored. Each leaf in the tree is 32 bytes = 256 bits.
// The number of bits per label is 256 / LabelsPerGroup. LabelsPerGroup = 1 << difficulty.
// Supported values range from 5 (8 bits per label) to 8 (1 bit per label).
func Initialize(id []byte, space uint64, numOfProvenLabels uint8, difficulty proving.Difficulty, parallel bool) (*proving.Proof, error) {
	return initialize(id, space, space, numOfProvenLabels, difficulty, parallel)
}

func initialize(id []byte, space uint64, filesize uint64, numOfProvenLabels uint8, difficulty proving.Difficulty, parallel bool) (*proving.Proof, error) {
	if err := proving.ValidateSpace(space); err != nil {
		return nil, err
	}

	if err := difficulty.Validate(); err != nil {
		return nil, err
	}

	numOfChunks, err := proving.NumOfFiles(space, filesize)
	if err != nil {
		return nil, err
	}
	labelGroupsPerChunk := proving.NumOfLabelGroups(filesize)

	results, err := initializeChunks(id, difficulty, numOfChunks, labelGroupsPerChunk, parallel)
	if err != nil {
		return nil, err
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
		Challenge:    proving.ZeroChallenge,
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

type initChunkResult struct {
	index int
	*initResult
}

func initializeChunks(id []byte, difficulty proving.Difficulty, numOfChunks int, labelGroupsPerChunk uint64, parallel bool) ([]*initResult, error) {
	var numOfWorkers int
	if parallel {
		numOfWorkers = min(runtime.NumCPU(), numOfChunks)
	} else {
		numOfWorkers = 1
	}

	jobsChan := make(chan int, numOfChunks)
	resultsChan := make(chan *initChunkResult, numOfChunks)
	errChan := make(chan error, 0)

	for i := 0; i < numOfChunks; i++ {
		jobsChan <- i
	}
	close(jobsChan)

	for i := 0; i < numOfWorkers; i++ {
		go func() {
			for {
				index, more := <-jobsChan
				if !more {
					return
				}

				res, err := initializeChunk(id, index, labelGroupsPerChunk, difficulty)
				if err != nil {
					errChan <- err
					return
				}

				resultsChan <- &initChunkResult{index, res}
			}
		}()
	}

	results := make([]*initResult, numOfChunks)
	for i := 0; i < numOfChunks; i++ {
		select {
		case res := <-resultsChan:
			results[res.index] = res.initResult
		case err := <-errChan:
			return nil, err
		}
	}

	return results, nil
}

func initializeChunk(id []byte, chunkIndex int, labelGroupsPerChunk uint64, difficulty proving.Difficulty) (*initResult, error) {
	// Initialize the labels file writer.
	labelsWriter, err := persistence.NewLabelsWriter(id, chunkIndex)
	if err != nil {
		return nil, err
	}

	// Initialize the labels merkle tree with the execution-phase zero challenge.
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(LowestLayerToCacheDuringProofGeneration), cache.MakeSliceReadWriterFactory())
	tree, err := merkle.NewTreeBuilder().
		WithHashFunc(proving.ZeroChallenge.GetSha256Parent).
		WithCacheWriter(cacheWriter).
		Build()
	if err != nil {
		return nil, err
	}

	// Calculate labels in groups, write them to disk
	// and append them as leaves in the merkle tree.
	for position := uint64(0); position < labelGroupsPerChunk; position++ {
		offset := uint64(chunkIndex) * labelGroupsPerChunk
		lg := CalcLabelGroup(id, position+offset, difficulty)
		err := labelsWriter.Write(lg)
		if err != nil {
			return nil, err
		}
		err = tree.AddLeaf(lg)
		if err != nil {
			return nil, err
		}
		if (position+1)%config.Post.LogEveryXLabels == 0 {
			log.Info("found %v labels", position+1)
		}
	}

	log.With().Info("completed PoST label list construction")

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

	res := &initResult{cacheReader, tree.Root()}

	return res, nil
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

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
