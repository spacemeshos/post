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

	numOfFiles, err := NumOfFiles(init.cfg.SpacePerUnit, init.cfg.FileSize)
	if err != nil {
		return nil, err
	}
	labelGroupsPerFile := NumOfLabelGroups(init.cfg.FileSize)

	results, err := init.initFiles(id, numOfFiles, labelGroupsPerFile)
	if err != nil {
		return nil, err
	}

	result, err := merge(results)
	if err != nil {
		return nil, err
	}

	leafReader := result.reader.GetLayerReader(0)
	width, err := leafReader.Width()
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

	err = leafReader.Close()
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

type initFileResult struct {
	index int
	*initResult
}

func (init *Initializer) initFiles(id []byte, numOfFiles int, labelGroupsPerFile uint64) ([]*initResult, error) {
	filesParallelism, infileParallelism := init.CalcParallelism(runtime.NumCPU())

	numOfWorkers := filesParallelism
	jobsChan := make(chan int, numOfFiles)
	resultsChan := make(chan *initFileResult, numOfFiles)
	errChan := make(chan error, 0)
	dir := shared.GetInitDir(init.cfg.DataDir, id)

	init.logger.Info("initialization: start writing %v files, parallelism degree: %v, dir: %v", numOfFiles, numOfWorkers, dir)

	for i := 0; i < numOfFiles; i++ {
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

				res, err := init.initFile(id, index, labelGroupsPerFile, dir, infileParallelism)
				if err != nil {
					errChan <- err
					return
				}

				resultsChan <- &initFileResult{index, res}
			}
		}()
	}

	results := make([]*initResult, numOfFiles)
	for i := 0; i < numOfFiles; i++ {
		select {
		case res := <-resultsChan:
			results[res.index] = res.initResult
		case err := <-errChan:
			return nil, err
		}
	}

	return results, nil
}

func (init *Initializer) initFile(id []byte, fileIndex int, labelGroupsPerFile uint64, dir string, infileParallelism int) (*initResult, error) {
	// Initialize the labels file writer.
	labelsWriter, err := persistence.NewLabelsWriter(id, fileIndex, dir)
	if err != nil {
		return nil, err
	}

	// Initialize the labels merkle tree with the execution-phase zero challenge.
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(init.cfg.LowestLayerToCacheDuringProofGeneration), cache.MakeSliceReadWriterFactory())
	tree, err := merkle.NewTreeBuilder().
		WithHashFunc(shared.ZeroChallenge.GetSha256Parent).
		WithCacheWriter(cacheWriter).
		Build()
	if err != nil {
		return nil, err
	}

	init.logger.Info("initialization: start writing file %v, parallelism degree: %v",
		fileIndex, infileParallelism)

	numOfWorkers := infileParallelism
	workersOutputChans := make([]chan [][]byte, numOfWorkers)
	errChan := make(chan error, 0)
	finishedChan := make(chan struct{}, 0)
	batchSize := 100
	chanBufferSize := 100

	// CPU workers.
	fileOffset := uint64(fileIndex) * labelGroupsPerFile
	for i := 0; i < numOfWorkers; i++ {
		i := i
		workersOutputChans[i] = make(chan [][]byte, chanBufferSize)
		workerOffset := i
		go func() {
			// Calculate labels in groups and write them to channel in batches.
			position := uint64(workerOffset * batchSize)
			batch := make([][]byte, batchSize)
			for position < labelGroupsPerFile {
				batch[position%uint64(batchSize)] = CalcLabelGroup(id, position+fileOffset, Difficulty(init.cfg.Difficulty))

				if position%uint64(batchSize) == uint64(batchSize-1) {
					workersOutputChans[i] <- batch
					batch = make([][]byte, batchSize)
					position += uint64(numOfWorkers * batchSize)
				}

				position += 1
			}
			workersOutputChans[i] <- batch
		}()
	}

	// IO worker.
	go func() {
	batchesLoop:
		for i := 0; ; i++ {
			// Consume the next batch from the next worker.
			batch := <-workersOutputChans[i%numOfWorkers]
			for j, lg := range batch {
				if lg == nil {
					break batchesLoop
				}

				// Write label group to disk, and append it as leaf in the merkle tree.
				err := labelsWriter.Write(lg)
				if err != nil {
					errChan <- err
					return
				}
				err = tree.AddLeaf(lg)
				if err != nil {
					errChan <- err
					return
				}

				num := batchSize*i + j + 1
				if uint64(num)%init.cfg.LabelsLogRate == 0 {
					init.logger.Info("initialization: file %v completed %v label groups", fileIndex, num)
				}
			}
		}

		close(finishedChan)
	}()

	select {
	case <-finishedChan:
	case err := <-errChan:
		return nil, err
	}

	labelsReader, err := labelsWriter.GetReader()
	if err != nil {
		return nil, err
	}

	info, err := labelsWriter.Close()
	if err != nil {
		return nil, err
	}

	init.logger.Info("initialization: completed file %v, bytes written: %v", fileIndex, info.Size())

	cacheWriter.SetLayer(0, labelsReader)
	cacheReader, err := cacheWriter.GetReader()
	if err != nil {
		return nil, err
	}

	return &initResult{cacheReader, tree.Root()}, nil
}

func (init *Initializer) CalcParallelism(maxParallelism int) (files int, infile int) {
	maxParallelism = max(maxParallelism, 1)
	files = max(int(init.cfg.MaxFilesParallelism), 1)
	files = min(files, maxParallelism)
	infile = max(int(init.cfg.MaxInFileParallelism), 1)
	infile = min(infile, maxParallelism)

	// Potentially reduce files parallelism in favor of in-file parallelism.
	for i := files; i > 0; i-- {
		if i*infile <= maxParallelism {
			files = i
			break
		}
	}

	return
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

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
