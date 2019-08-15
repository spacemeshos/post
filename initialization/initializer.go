package initialization

import (
	"bytes"
	"code.cloudfoundry.org/bytefmt"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/nullstyle/go-xdr/xdr3"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

const (
	LabelGroupSize = config.LabelGroupSize
)

type (
	Config           = config.Config
	Proof            = shared.Proof
	Difficulty       = shared.Difficulty
	Logger           = shared.Logger
	MTreeOutput      = shared.MTreeOutput
	MTreeOutputEntry = shared.MTreeOutputEntry
	CacheReader      = cache.CacheReader
)

var (
	ValidateSpace  = shared.ValidateSpace
	NumFiles       = shared.NumFiles
	NumLabelGroups = shared.NumLabelGroups
)

type state int

var states = []string{
	"NOT_STARTED",
	"COMPLETED",
	"CRASHED",
}

const (
	StateNotStarted state = 1 + iota
	StateCompleted
	StateCrashed
)

func (s state) String() string {
	return states[s-1]
}

var (
	ErrStateMetadataFileMissing = errors.New("metadata file missing")
	ErrStateConfigMismatch      = errors.New("config mismatch")
	ErrStateInconsistent        = errors.New("inconsistent state")
)

var metadataFileName = ".init"

type metadataState int

const (
	MetadataStateStarted metadataState = 1 + iota
	MetadataStateCompleted
)

type metadata struct {
	State metadataState
	Cfg   config.Config
}

type Initializer struct {
	cfg    *Config
	id     []byte
	dir    string
	logger Logger
}

func NewInitializer(cfg *Config, id []byte) *Initializer {
	dir := shared.GetInitDir(cfg.DataDir, id)
	return &Initializer{cfg, id, dir, shared.DisabledLogger{}}
}

func (init *Initializer) SetLogger(logger Logger) {
	init.logger = logger
}

// Initialize perform the initialization procedure and returns a proof of the
// initialized data with an empty challenge. The data and the proof are applied
// to the configuration of the Initializer instance: id, space, numProvenLabels and difficulty.
// Difficulty determines the number of bits per label that are stored. Each leaf in the tree is 32 bytes = 256 bits.
// The number of bits per label is 256 / LabelsPerGroup. LabelsPerGroup = 1 << difficulty.
// Supported values range from 5 (8 bits per label) to 8 (1 bit per label).
func (init *Initializer) Initialize() (*Proof, error) {
	if err := ValidateSpace(init.cfg.SpacePerUnit); err != nil {
		return nil, err
	}

	difficulty := Difficulty(init.cfg.Difficulty)
	if err := difficulty.Validate(); err != nil {
		return nil, err
	}

	state, requiredSpace, err := init.State()
	if err != nil {
		return nil, err
	}

	if state == StateCompleted {
		return nil, shared.ErrInitCompleted
	}

	if !init.cfg.DisableSpaceAvailabilityChecks {
		availableSpace := shared.AvailableSpace(init.cfg.DataDir)
		if requiredSpace > availableSpace {
			return nil, fmt.Errorf("not enough disk space. required: %v, available: %v",
				bytefmt.ByteSize(requiredSpace), bytefmt.ByteSize(availableSpace))
		}
	}

	if err := init.SaveMetadata(MetadataStateStarted); err != nil {
		return nil, err
	}

	numFiles, err := NumFiles(init.cfg.SpacePerUnit, init.cfg.FileSize)
	if err != nil {
		return nil, err
	}
	labelGroupsPerFile := NumLabelGroups(init.cfg.FileSize)

	outputs, err := init.initFiles(numFiles, labelGroupsPerFile)
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

	provenLeafIndices := shared.CalcProvenLeafIndices(output.Root, width<<difficulty, uint8(init.cfg.NumProvenLabels), difficulty)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, output.Reader)
	if err != nil {
		return nil, err
	}

	err = leafReader.Close()
	if err != nil {
		return nil, err
	}

	if err := init.SaveMetadata(MetadataStateCompleted); err != nil {
		return nil, err
	}

	proof := &Proof{
		Challenge:    shared.ZeroChallenge,
		Identity:     init.id,
		MerkleRoot:   output.Root,
		ProvenLeaves: provenLeaves,
		ProofNodes:   proofNodes,
	}

	return proof, err
}

func (init *Initializer) Reset() error {
	if err := init.VerifyStarted(); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(init.dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if init.isInitFile(file) || file.Name() == metadataFileName {
			path := filepath.Join(init.dir, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file (%v): %v", path, err)
			}
		}
	}

	// Remove the dir if it's empty.
	_ = os.Remove(init.dir)

	init.logger.Info("id %v reset, directory %v files deleted", hex.EncodeToString(init.id), init.dir)

	return nil
}

func (init *Initializer) VerifyStarted() error {
	state, _, err := init.State()
	if err != nil {
		return err
	}

	if state == StateNotStarted {
		return shared.ErrInitNotStarted
	}

	return nil
}

func (init *Initializer) VerifyNotCompleted() error {
	state, _, err := init.State()
	if err != nil {
		return err
	}

	if state == StateCompleted {
		return shared.ErrInitCompleted

	}

	return nil
}

func (init *Initializer) VerifyCompleted() error {
	state, _, err := init.State()
	if err != nil {
		return fmt.Errorf("initialization state error: %v", err)
	}

	if state != StateCompleted {
		return fmt.Errorf("initialization not completed, state: %v, dir: %v", state.String(), init.dir)
	}

	return nil
}

func (init *Initializer) initFiles(numFiles int, labelGroupsPerFile uint64) ([]*MTreeOutput, error) {
	filesParallelism, infileParallelism := init.CalcParallelism()
	numWorkers := filesParallelism
	jobsChan := make(chan int, numFiles)
	resultsChan := make(chan *MTreeOutputEntry, numFiles)
	errChan := make(chan error, 0)

	init.logger.Info("initialization: start writing %v files, parallelism degree: %v, dir: %v", numFiles, numWorkers, init.dir)

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

				output, err := init.initFile(index, labelGroupsPerFile, infileParallelism)
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

func (init *Initializer) initFile(fileIndex int, labelGroupsPerFile uint64, infileParallelism int) (*MTreeOutput, error) {
	// Initialize the labels file writer.
	labelsWriter, err := persistence.NewLabelsWriter(init.cfg.DataDir, init.id, fileIndex)
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

	// Potentially perform recovery procedure.
	// if file already exists, read it from start, and re-construct the merkle tree state.
	// If file initialization was complete, return the merkle tree output.
	// Otherwise continue to initialize from latest position.

	labelsReader, err := labelsWriter.GetReader()
	if err != nil {
		return nil, err
	}

	existingWidth, err := labelsReader.Width()
	if err != nil {
		return nil, err
	}

	if existingWidth > 0 {
		if existingWidth > labelGroupsPerFile {
			return nil, ErrStateInconsistent
		}

		init.logger.Info("initialization recovery: starting file %v, position: %v, missing: %v", fileIndex, existingWidth, labelGroupsPerFile-existingWidth)

		for {
			lg, err := labelsReader.ReadNext()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}

			err = tree.AddLeaf(lg)
			if err != nil {
				return nil, err
			}
		}

		if existingWidth == labelGroupsPerFile {
			cacheWriter.SetLayer(0, labelsReader)
			cacheReader, err := cacheWriter.GetReader()
			if err != nil {
				return nil, err
			}

			return &MTreeOutput{Reader: cacheReader, Root: tree.Root()}, nil
		}
	}

	init.logger.Info("initialization: start writing file %v, parallelism degree: %v",
		fileIndex, infileParallelism)

	numWorkers := infileParallelism
	workersChans := make([]chan [][]byte, numWorkers)
	errChan := make(chan error, 0)
	finishedChan := make(chan struct{}, 0)

	batchSize := 100 // CPU workers send data to IO worker in batches, to reduce channel ops overhead.
	chanBufferSize := 100

	// Start CPU workers.
	fileOffset := uint64(fileIndex) * labelGroupsPerFile
	for i := 0; i < numWorkers; i++ {
		i := i
		workersChans[i] = make(chan [][]byte, chanBufferSize)
		workerOffset := i

		// Calculate labels in groups and write them to channel in batches.
		go func() {
			position := uint64(workerOffset*batchSize) + existingWidth // the starting point of this specific worker.
			batchPosition := uint64(0)                                 // the starting point of the current batch.
			batch := make([][]byte, batchSize)

			// Continue as long as the combined position didn't reach the file capacity.
			for batchPosition+position < labelGroupsPerFile {
				// Calculate the label group of the combined position and the global offset for this file.
				batch[batchPosition] = CalcLabelGroup(init.id, batchPosition+position+fileOffset, Difficulty(init.cfg.Difficulty))

				// If batch was filled, send it over the channel, and instantiate a new empty batch.
				// In addition, adjust position to the next location for this worker, after its own and all other workers batches.
				if batchPosition == uint64(batchSize-1) {
					workersChans[i] <- batch
					batch = make([][]byte, batchSize)
					batchPosition = 0

					position += uint64(numWorkers * batchSize)

					continue
				}

				batchPosition += 1
			}
			workersChans[i] <- batch
		}()
	}

	// Start IO worker.
	go func() {
	batchesLoop:
		for i := 0; ; i++ {
			// Consume the next batch from the next worker.
			batch := <-workersChans[i%numWorkers]
			for j, lg := range batch {

				// The first empty label group indicates an unfilled batch which indicates the end of work.
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

	labelsReader, err = labelsWriter.GetReader()
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

func (init *Initializer) State() (state, uint64, error) {
	requiredSpace := init.cfg.SpacePerUnit

	files, err := ioutil.ReadDir(init.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return StateNotStarted, requiredSpace, nil
		}
		return 0, 0, err
	}

	initFiles := make([]os.FileInfo, 0)
	for _, file := range files {
		if init.isInitFile(file) {
			initFiles = append(initFiles, file)
		}
	}
	if len(initFiles) == 0 {
		return StateNotStarted, requiredSpace, nil
	}

	metadata, err := init.LoadMetadata()
	if err != nil {
		return 0, 0, err
	}

	switch metadata.State {
	case MetadataStateCompleted:
		return StateCompleted, 0, nil
	case MetadataStateStarted:
		if !configMatch(&metadata.Cfg, init.cfg) {
			return 0, 0, ErrStateConfigMismatch
		}

		for _, file := range initFiles {
			if requiredSpace < uint64(file.Size()) {
				return 0, 0, ErrStateInconsistent
			}
			requiredSpace -= uint64(file.Size())
		}

		if requiredSpace%LabelGroupSize != 0 {
			return 0, 0, ErrStateInconsistent
		}

		return StateCrashed, requiredSpace, nil
	default:
		return 0, 0, ErrStateInconsistent
	}
}

func (init *Initializer) SaveMetadata(state metadataState) error {
	err := os.MkdirAll(init.dir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %v", err)
	}

	var w bytes.Buffer
	a := metadata{state, *init.cfg}
	_, err = xdr.Marshal(&w, a)
	if err != nil {
		return fmt.Errorf("serialization failure: %v", err)
	}

	err = ioutil.WriteFile(filepath.Join(init.dir, metadataFileName), w.Bytes(), shared.OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil
}

func (init *Initializer) LoadMetadata() (*metadata, error) {
	filename := filepath.Join(init.dir, metadataFileName)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrStateMetadataFileMissing
		}
		return nil, fmt.Errorf("read file failure: %v", err)
	}

	metadata := &metadata{}
	_, err = xdr.Unmarshal(bytes.NewReader(data), metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (init *Initializer) isInitFile(file os.FileInfo) bool {
	return shared.IsInitFile(init.id, file)
}

func configMatch(cfg1 *config.Config, cfg2 *config.Config) bool {
	return cfg1.SpacePerUnit == cfg2.SpacePerUnit &&
		cfg1.FileSize == cfg2.FileSize &&
		cfg1.Difficulty == cfg2.Difficulty &&
		cfg1.NumProvenLabels == cfg2.NumProvenLabels
}
