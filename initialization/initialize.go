package initialization

import (
	"encoding/hex"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"os"
	"runtime"
)

const (
	LabelGroupSize = shared.LabelGroupSize
	MaxSpace       = shared.MaxSpace
)

type (
	Config      = shared.Config
	Logger      = shared.Logger
	Difficulty  = shared.Difficulty
	CacheReader = cache.CacheReader
)

var (
	VerifyNotInitialized = shared.VerifyNotInitialized
	VerifyInitialized    = shared.VerifyInitialized
	ValidateSpace        = shared.ValidateSpace
	NumOfFiles           = shared.NumOfFiles
	NumOfLabelGroups     = shared.NumOfLabelGroups
)

type Initializer struct {
	cfg    *Config
	logger Logger
}

func NewInitializer(cfg *Config, logger Logger) *Initializer { return &Initializer{cfg, logger} }

// Initialize takes an id (public key), space (in bytes), numOfProvenLabels and difficulty.
// Difficulty determines the number of bits per label that are stored. Each leaf in the tree is 32 bytes = 256 bits.
// The number of bits per label is 256 / LabelsPerGroup. LabelsPerGroup = 1 << difficulty.
// Supported values range from 5 (8 bits per label) to 8 (1 bit per label).
func (init *Initializer) Initialize(id []byte) (*proving.Proof, error) {
	if err := VerifyNotInitialized(init.cfg, id); err != nil {
		return nil, err
	}

	if err := ValidateSpace(init.cfg.SpacePerUnit); err != nil {
		return nil, err
	}

	difficulty := Difficulty(init.cfg.Difficulty)
	if err := difficulty.Validate(); err != nil {
		return nil, err
	}

	numOfChunks, err := NumOfFiles(init.cfg.SpacePerUnit, init.cfg.FileSize)
	if err != nil {
		return nil, err
	}
	labelGroupsPerChunk := NumOfLabelGroups(init.cfg.FileSize)

	dir := shared.GetInitDir(init.cfg.DataDir, id)
	results, err := initChunks(id, difficulty, init.cfg.LowestLayerToCacheDuringProofGeneration, numOfChunks, labelGroupsPerChunk, init.cfg.EnableParallelism, dir, init.cfg.LabelsLogRate, init.logger)
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

	provenLeafIndices := proving.CalcProvenLeafIndices(result.root, width<<difficulty, uint8(init.cfg.NumOfProvenLabels), difficulty)
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

func (init *Initializer) Reset(id []byte) error {
	if err := VerifyInitialized(init.cfg, id); err != nil {
		return err
	}

	dir := shared.GetInitDir(init.cfg.DataDir, id)
	err := os.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("failed to delete directory (%v)", dir)
	}

	init.logger.Info("id %v reset, directory %v deleted", hex.EncodeToString(id), dir)

	return nil
}

type initResult struct {
	reader CacheReader
	root   []byte
}

type initChunkResult struct {
	index int
	*initResult
}

func initChunks(id []byte, difficulty proving.Difficulty, lowestLayerToCacheDuringProofGeneration uint, numOfChunks int,
	labelGroupsPerChunk uint64, parallel bool, dir string, logRate uint64, logger Logger) ([]*initResult, error) {
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

				res, err := initChunk(id, difficulty, lowestLayerToCacheDuringProofGeneration, index, labelGroupsPerChunk, dir, logRate, logger)
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

func initChunk(id []byte, difficulty proving.Difficulty, lowestLayerToCacheDuringProofGeneration uint, chunkIndex int, labelGroupsPerChunk uint64, dir string, logRate uint64, logger Logger) (*initResult, error) {
	// Initialize the labels file writer.
	labelsWriter, err := persistence.NewLabelsWriter(id, chunkIndex, dir, logger)
	if err != nil {
		return nil, err
	}

	// Initialize the labels merkle tree with the execution-phase zero challenge.
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(lowestLayerToCacheDuringProofGeneration), cache.MakeSliceReadWriterFactory())
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
		if (position+1)%logRate == 0 {
			logger.Info("completed %v labels", position+1)
		}
	}

	logger.Info("completed PoST label list construction")

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
