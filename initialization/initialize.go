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
	Config           = shared.Config
	Logger           = shared.Logger
	Difficulty       = shared.Difficulty
	MTreeOutput      = shared.MTreeOutput
	MTreeOutputEntry = shared.MTreeOutputEntry
	CacheReader      = cache.CacheReader
)

var (
	VerifyNotInitialized = shared.VerifyNotInitialized
	VerifyInitialized    = shared.VerifyInitialized
	ValidateSpace        = shared.ValidateSpace
	NumFiles             = shared.NumFiles
	NumLabelGroups       = shared.NumLabelGroups
)

type Initializer struct {
	cfg    *Config
	logger Logger
}

func NewInitializer(cfg *Config, logger Logger) *Initializer { return &Initializer{cfg, logger} }

// Initialize takes an id (public key), space (in bytes), numProvenLabels and difficulty.
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

	numFiles, err := NumFiles(init.cfg.SpacePerUnit, init.cfg.FileSize)
	if err != nil {
		return nil, err
	}
	labelGroupsPerFile := NumLabelGroups(init.cfg.FileSize)

	outputs, err := init.initFiles(id, numFiles, labelGroupsPerFile)
	if err != nil {
		return nil, err
	}

	output, err := shared.Merge(outputs)
	if err != nil {
		return nil, err
	}

	leafReader := output.Reader.GetLayerReader(0)
	width, err := leafReader.Width()
	if err != nil {
		err = fmt.Errorf("failed to get leaves reader width: %v", err)
		log.Error(err.Error())
		return nil, err
	}

	provenLeafIndices := proving.CalcProvenLeafIndices(output.Root, width<<difficulty, uint8(init.cfg.NumProvenLabels), difficulty)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, output.Reader)
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
		MerkleRoot:   output.Root,
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

func (init *Initializer) initFiles(id []byte, numFiles int, labelGroupsPerFile uint64) ([]*MTreeOutput, error) {
	filesParallelism, infileParallelism := init.CalcParallelism()
	numWorkers := filesParallelism
	jobsChan := make(chan int, numFiles)
	resultsChan := make(chan *MTreeOutputEntry, numFiles)
	errChan := make(chan error, 0)
	dir := shared.GetInitDir(init.cfg.DataDir, id)

	init.logger.Info("initialization: start writing %v files, parallelism degree: %v, dir: %v", numFiles, numWorkers, dir)

	for i := 0; i < numFiles; i++ {
		jobsChan <- i
	}
	close(jobsChan)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				index, more := <-jobsChan
				if !more {
					return
				}

				output, err := init.initFile(id, index, labelGroupsPerFile, dir, infileParallelism)
				if err != nil {
					errChan <- err
					return
				}

				resultsChan <- &MTreeOutputEntry{Index: index, MTreeOutput: output}
			}
		}()
	}

	outputs := make([]*MTreeOutput, numFiles)
	for i := 0; i < numFiles; i++ {
		select {
		case res := <-resultsChan:
			outputs[res.Index] = res.MTreeOutput
		case err := <-errChan:
			return nil, err
		}
	}

	return outputs, nil
}

func (init *Initializer) initFile(id []byte, fileIndex int, labelGroupsPerFile uint64, dir string, infileParallelism int) (*MTreeOutput, error) {
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

	numWorkers := infileParallelism
	workersChans := make([]chan [][]byte, numWorkers)
	errChan := make(chan error, 0)
	finishedChan := make(chan struct{}, 0)
	batchSize := 100
	chanBufferSize := 100

	// CPU workers.
	fileOffset := uint64(fileIndex) * labelGroupsPerFile
	for i := 0; i < numWorkers; i++ {
		i := i
		workersChans[i] = make(chan [][]byte, chanBufferSize)
		workerOffset := i
		go func() {
			// Calculate labels in groups and write them to channel in batches.
			position := uint64(workerOffset * batchSize)
			batch := make([][]byte, batchSize)
			for position < labelGroupsPerFile {
				batch[position%uint64(batchSize)] = CalcLabelGroup(id, position+fileOffset, Difficulty(init.cfg.Difficulty))

				if position%uint64(batchSize) == uint64(batchSize-1) {
					workersChans[i] <- batch
					batch = make([][]byte, batchSize)
					position += uint64((numWorkers - 1) * batchSize)
				}

				position += 1
			}
			workersChans[i] <- batch
		}()
	}

	// IO worker.
	go func() {
	batchesLoop:
		for i := 0; ; i++ {
			// Consume the next batch from the next worker.
			batch := <-workersChans[i%numWorkers]
			for j, lg := range batch {
				if lg == nil {
					break batchesLoop
				}

				// Write label group to disk.
				err := labelsWriter.Write(lg)
				if err != nil {
					errChan <- err
					return
				}

				// Append label group as leaf in the merkle tree. The tree cache
				// isn't suppose to handle writing of the leaf layer (0) to disk.
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

	return &MTreeOutput{Reader: cacheReader, Root: tree.Root()}, nil
}

func (init *Initializer) CalcParallelism() (files int, infile int) {
	return init.calcParallelism(runtime.NumCPU())
}

func (init *Initializer) calcParallelism(max int) (files int, infile int) {
	max = shared.Max(max, 1)
	files = shared.Max(int(init.cfg.MaxWriteFilesParallelism), 1)
	infile = shared.Max(int(init.cfg.MaxWriteInFileParallelism), 1)

	files = shared.Min(files, max)
	infile = shared.Min(infile, max)

	// Potentially reduce files parallelism in favor of in-file parallelism.
	for i := files; i > 0; i-- {
		if i*infile <= max {
			files = i
			break
		}
	}

	return
}
