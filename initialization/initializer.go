package initialization

import (
	"bytes"
	"code.cloudfoundry.org/bytefmt"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/nullstyle/go-xdr/xdr3"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

type (
	Config = config.Config
	Proof  = shared.Proof
	Logger = shared.Logger
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

func NewInitializer(cfg *Config, id []byte) (*Initializer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	dir := shared.GetInitDir(cfg.DataDir, id)
	return &Initializer{cfg, id, dir, shared.DisabledLogger{}}, nil
}

func (init *Initializer) SetLogger(logger Logger) {
	init.logger = logger
}

// Initialize perform the initialization procedure.
func (init *Initializer) Initialize() error {
	if err := init.VerifyInitAllowed(); err != nil {
		return err
	}

	if err := init.SaveMetadata(MetadataStateStarted); err != nil {
		return err
	}

	numLabelsPerFile := init.cfg.NumLabels / uint64(init.cfg.NumFiles)
	if err := init.initFiles(int(init.cfg.NumFiles), numLabelsPerFile); err != nil {
		return err
	}

	if err := init.SaveMetadata(MetadataStateCompleted); err != nil {
		return err
	}

	return nil
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

func (init *Initializer) VerifyInitAllowed() error {
	state, requiredSpace, err := init.State()
	if err != nil {
		return err
	}

	if state == StateCompleted {
		return shared.ErrInitCompleted
	}

	if !init.cfg.DisableSpaceAvailabilityChecks {
		availableSpace := shared.AvailableSpace(init.cfg.DataDir)
		if requiredSpace > availableSpace {
			return fmt.Errorf("not enough disk space. required: %v, available: %v",
				bytefmt.ByteSize(requiredSpace), bytefmt.ByteSize(availableSpace))
		}
	}

	return nil
}

func (init *Initializer) initFiles(numFiles int, numLabelsPerFile uint64) error {
	filesParallelism, infileParallelism := init.CalcParallelism()
	numWorkers := filesParallelism
	jobsChan := make(chan int, numFiles)
	okChan := make(chan struct{}, numFiles)
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

				if err := init.initFile(index, numLabelsPerFile, infileParallelism); err != nil {
					errChan <- err
					return
				}

				okChan <- struct{}{}
			}
		}()
	}

	for i := 0; i < numFiles; i++ {
		select {
		case <-okChan:
		case err := <-errChan:
			return err
		}
	}

	return nil
}

func (init *Initializer) initFile(fileIndex int, numLabels uint64, infileParallelism int) error {
	// Initialize the labels file writer.
	labelsWriter, err := persistence.NewLabelsWriter(init.cfg.DataDir, init.id, fileIndex, init.cfg.LabelSize)
	if err != nil {
		return err
	}

	// Potentially perform recovery procedure; continue to initialize from latest position.

	labelsReader, err := labelsWriter.GetReader()
	if err != nil {
		return err
	}

	existingWidth, err := labelsReader.Width()
	if err != nil {
		return err
	}

	if existingWidth > 0 {
		if existingWidth > numLabels {
			return ErrStateInconsistent
		}

		init.logger.Info("initialization recovery: starting file %v, position: %v, missing: %v", fileIndex, existingWidth, numLabels-existingWidth)

		if existingWidth == numLabels {
			return nil
		}
	} else {
		init.logger.Info("initialization: start writing file %v, parallelism degree: %v",
			fileIndex, infileParallelism)
	}

	numWorkers := infileParallelism
	workersChans := make([]chan [][]byte, numWorkers)
	okChan := make(chan struct{}, 0)
	errChan := make(chan error, 0)

	batchSize := 100 // CPU workers send data to IO worker in batches, to reduce channel ops overhead.
	chanBufferSize := 100

	// Start CPU workers.
	fileOffset := uint64(fileIndex) * numLabels
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
			for batchPosition+position < numLabels {
				// Calculate the label of the combined position and the global offset for this file.
				batch[batchPosition] = oracle.WorkOracle(init.id, batchPosition+position+fileOffset, init.cfg.LabelSize)

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
			for j, label := range batch {

				// The first empty label indicates an unfilled batch which indicates the end of work.
				if label == nil {
					break batchesLoop
				}

				// Write label to disk.
				if err := labelsWriter.Write(label); err != nil {
					errChan <- err
					return
				}

				num := batchSize*i + j + 1
				if uint64(num)%init.cfg.LabelsLogRate == 0 {
					init.logger.Info("initialization: file %v completed %v labels", fileIndex, num)
				}
			}
		}

		close(okChan)
	}()

	select {
	case <-okChan:
	case err := <-errChan:
		return err
	}

	info, err := labelsWriter.Close()
	if err != nil {
		return err
	}

	init.logger.Info("initialization: completed file %v, bytes written: %v", fileIndex, info.Size())

	return nil
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
	requiredSpace := init.cfg.Space()

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

	if !configMatch(&metadata.Cfg, init.cfg) {
		return 0, 0, ErrStateConfigMismatch
	}

	switch metadata.State {
	case MetadataStateCompleted:
		return StateCompleted, 0, nil
	case MetadataStateStarted:
		for _, file := range initFiles {
			if requiredSpace < uint64(file.Size()) {
				return 0, 0, ErrStateInconsistent
			}
			requiredSpace -= uint64(file.Size())
		}

		if requiredSpace%uint64(init.cfg.LabelSize) != 0 {
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
	return cfg1.NumLabels == cfg2.NumLabels &&
		cfg1.LabelSize == cfg2.LabelSize &&
		cfg1.NumFiles == cfg2.NumFiles
}
