package initialization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type (
	Config          = config.Config
	Proof           = shared.Proof
	Logger          = shared.Logger
	ComputeProvider = gpu.ComputeProvider
)

var (
	ErrNotInitializing              = errors.New("not initializing")
	ErrAlreadyInitializing          = errors.New("already initializing")
	ErrCannotResetWhileInitializing = errors.New("cannot reset while initializing")
	ErrStopped                      = errors.New("stopped")

	ErrStateMetadataFileMissing = errors.New("metadata file missing")
	ErrStateInconsistent        = errors.New("inconsistent state")
)

type configMismatchError struct {
	param    string
	expected string
	found    string
	datadir  string
}

func (err configMismatchError) Error() string {
	return fmt.Sprintf("`%v` config mismatch; expected: %v, found: %v, datadir: %v",
		err.param, err.expected, err.found, err.datadir)
}

type unexpectedFileSize struct {
	expected string
	found    string
	filename string
}

func (err unexpectedFileSize) Error() string {
	return fmt.Sprintf("unexpected file size; expected: %v, found: %v, filename: %v",
		err.expected, err.found, err.filename)
}

type DiskState struct {
	InitState    initState
	BytesWritten uint64
}

type initState int

const (
	InitStateNotStarted initState = 1 + iota
	InitStateCompleted
	InitStateStopped
	InitStateCrashed
)

func (s initState) String() string {
	switch s {
	case InitStateNotStarted:
		return "not started"
	case InitStateCompleted:
		return "completed"
	case InitStateStopped:
		return "stopped"
	case InitStateCrashed:
		return "crashed"
	default:
		panic("unreachable")
	}
}

var (
	providers     []ComputeProvider
	cpuProviderID uint
)

func init() {
	providers = gpu.Providers()
	cpuProviderID = cpuProvider(providers).ID
}

func Providers() []ComputeProvider {
	return providers
}

func CPUProviderID() uint {
	return cpuProviderID
}

func cpuProvider(providers []ComputeProvider) ComputeProvider {
	for _, p := range providers {
		if p.Model == "CPU" {
			return p
		}
	}
	panic("unreachable")
}

type Initializer struct {
	cfg *Config
	id  []byte

	initializing bool
	mtx          sync.Mutex

	progressChan chan float64
	stopChan     chan struct{}
	doneChan     chan struct{}

	logger Logger
}

func NewInitializer(cfg *Config, id []byte) (*Initializer, error) {
	if len(id) != 32 {
		return nil, fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(id))
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Initializer{
		cfg:    cfg,
		id:     id,
		logger: shared.DisabledLogger{},
	}, nil
}

// Initialize is the process in which the prover commits to store some data, by having its storage filled with
// pseudo-random data with respect to a specific id. This data is the result of a computationally-expensive operation.
func (init *Initializer) Initialize(computeProviderID uint) error {
	init.mtx.Lock()
	if init.initializing {
		init.mtx.Unlock()
		return ErrAlreadyInitializing
	}
	init.stopChan = make(chan struct{})
	init.doneChan = make(chan struct{})
	init.initializing = true
	init.mtx.Unlock()
	defer func() {
		init.initializing = false
		close(init.doneChan)

		if init.progressChan != nil {
			close(init.progressChan)
			init.progressChan = nil
		}
	}()

	if err := init.VerifyInitAllowed(); err != nil {
		return err
	}

	if err := init.SaveMetadata(MetadataInitStateStarted); err != nil {
		return err
	}

	fileNumLabels := init.cfg.NumLabels / uint64(init.cfg.NumFiles)

	init.logger.Info("initialization: starting to write %v file(s); numLabels: %v, fileNumLabels: %v, labelSize: %v, labelsCalcBatchSize: %v, datadir: %v",
		init.cfg.NumFiles, init.cfg.NumLabels, fileNumLabels, init.cfg.LabelSize, init.cfg.LabelsCalcBatchSize, init.cfg.DataDir)

	for i := 0; i < int(init.cfg.NumFiles); i++ {
		if err := init.initFile(computeProviderID, i, fileNumLabels); err != nil {
			if err == ErrStopped {
				if err := init.SaveMetadata(MetadataInitStateStopped); err != nil {
					return err
				}
			}
			return err
		}
	}

	if err := init.SaveMetadata(MetadataInitStateCompleted); err != nil {
		return err
	}

	return nil
}

func (init *Initializer) Stop() error {
	if !init.initializing {
		return ErrNotInitializing
	}

	close(init.stopChan)
	if res := gpu.Stop(); res != gpu.StopResultOk {
		return fmt.Errorf("gpu stop error: %s", res)
	}

	select {
	case <-init.doneChan:
	case <-time.After(5 * time.Second):
		return errors.New("stop timeout")
	}

	return nil
}

func (init *Initializer) Progress() <-chan float64 {
	if init.progressChan == nil {
		init.progressChan = make(chan float64, 1024)
	}
	return init.progressChan
}

func (init *Initializer) Reset() error {
	if init.initializing {
		return ErrCannotResetWhileInitializing
	}

	if err := init.VerifyStarted(); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(init.cfg.DataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if init.isInitFile(file) || file.Name() == metadataFileName {
			path := filepath.Join(init.cfg.DataDir, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file (%v): %v", path, err)
			}
		}
	}

	return nil
}

func (init *Initializer) VerifyStarted() error {
	state, err := init.DiskState()
	if err != nil {
		return err
	}

	if state.InitState == InitStateNotStarted {
		return shared.ErrInitNotStarted
	}

	return nil
}

func (init *Initializer) VerifyNotCompleted() error {
	diskState, err := init.DiskState()
	if err != nil {
		return err
	}

	if diskState.InitState == InitStateCompleted {
		return shared.ErrInitCompleted

	}

	return nil
}

func (init *Initializer) VerifyCompleted() error {
	diskState, err := init.DiskState()
	if err != nil {
		return err
	}

	if diskState.InitState != InitStateCompleted {
		return fmt.Errorf("initialization not completed; state: %s, datadir: %v", diskState.InitState, init.cfg.DataDir)
	}

	return nil
}

func (init *Initializer) VerifyInitAllowed() error {
	diskState, err := init.DiskState()
	if err != nil {
		return err
	}

	if diskState.InitState == InitStateCompleted {
		return shared.ErrInitCompleted
	}

	return nil
}

func (init *Initializer) initFile(computeProviderID uint, fileIndex int, numLabelsPerFile uint64) error {
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
		if existingWidth > numLabelsPerFile {
			return ErrStateInconsistent
		}

		if existingWidth == numLabelsPerFile {
			return nil
		}

		init.logger.Debug("initialization recovery: start writing file %v, position: %v, number of missing labels: %v", fileIndex, existingWidth, numLabelsPerFile-existingWidth)
	} else {
		init.logger.Debug("initialization: starting to write file #%v; numLabels: %v", fileIndex, numLabelsPerFile)
	}

	fileOffset := uint64(fileIndex) * numLabelsPerFile
	currentPosition := existingWidth
	batchSize := init.cfg.LabelsCalcBatchSize
	outputChan := make(chan []byte, 1024)
	computeErr := make(chan error, 1)
	ioError := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	// Start compute worker.
	go func() {
		defer func() {
			close(outputChan)
			wg.Done()
		}()
		for currentPosition < numLabelsPerFile {
			select {
			case <-init.stopChan:
				init.logger.Info("initialization: stopped")
				computeErr <- ErrStopped
				return
			case <-ioError:
				return
			default:
			}

			// The last batch might need to be smaller.
			remaining := uint(numLabelsPerFile - currentPosition)
			if remaining < batchSize {
				batchSize = remaining
			}

			init.logger.Debug("initialization: file #%v current position: %v, remaining: %v", fileIndex, currentPosition, remaining)

			// Calculate labels of the batch position range.
			startPosition := fileOffset + currentPosition
			endPosition := startPosition + uint64(batchSize) - 1
			output, err := oracle.WorkOracle(computeProviderID, init.id, startPosition, endPosition, uint8(init.cfg.LabelSize))
			if err != nil {
				computeErr <- err
				return
			}
			outputChan <- output
			currentPosition += uint64(batchSize)

			if init.progressChan != nil {
				init.progressChan <- float64(fileOffset+currentPosition) / float64(init.cfg.NumLabels)
			}
		}
	}()

	// Start IO worker.
	go func() {
		defer wg.Done()
		for {
			batch, more := <-outputChan
			if !more {
				_ = labelsWriter.Flush()
				return
			}

			// Write labels batch to disk.
			if err := labelsWriter.Write(batch); err != nil {
				ioError <- err
				return
			}
		}
	}()

	wg.Wait()

	select {
	case err := <-computeErr:
		return err
	case err := <-ioError:
		return err
	default:
	}

	info, err := labelsWriter.Close()
	if err != nil {
		return err
	}

	init.logger.Info("initialization: file #%v completed; bytes written: %v", fileIndex, info.Size())

	return nil
}

func (init *Initializer) DiskState() (*DiskState, error) {
	files, err := ioutil.ReadDir(init.cfg.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &DiskState{InitStateNotStarted, 0}, nil
		}
		return nil, err
	}

	initFiles := make([]os.FileInfo, 0)
	for _, file := range files {
		if init.isInitFile(file) {
			initFiles = append(initFiles, file)
		}
	}

	metadata, err := init.LoadMetadata()
	if err != nil {
		if err == ErrStateMetadataFileMissing && len(initFiles) == 0 {
			return &DiskState{InitStateNotStarted, 0}, nil
		}
		return nil, err
	}

	if bytes.Compare(init.id, metadata.ID) != 0 {
		return nil, configMismatchError{
			param:    "id",
			expected: fmt.Sprintf("%x", init.id),
			found:    fmt.Sprintf("%x", metadata.ID),
			datadir:  init.cfg.DataDir,
		}
	}

	if init.cfg.NumFiles != metadata.Cfg.NumFiles {
		return nil, configMismatchError{
			param:    "NumFiles",
			expected: fmt.Sprintf("%d", init.cfg.NumFiles),
			found:    fmt.Sprintf("%d", metadata.Cfg.NumFiles),
			datadir:  init.cfg.DataDir,
		}
	}

	if init.cfg.LabelSize != metadata.Cfg.LabelSize {
		return nil, configMismatchError{
			param:    "LabelSize",
			expected: fmt.Sprintf("%d", init.cfg.LabelSize),
			found:    fmt.Sprintf("%d", metadata.Cfg.LabelSize),
			datadir:  init.cfg.DataDir,
		}
	}

	fileNumLabels := init.cfg.NumLabels / uint64(init.cfg.NumFiles)
	fileDataSize := shared.DataSize(fileNumLabels, init.cfg.LabelSize)
	var bytesWritten uint64
	for _, file := range initFiles {
		fileSize := uint64(file.Size())
		if fileSize > fileDataSize {
			return nil, unexpectedFileSize{
				expected: fmt.Sprintf("<= %d", fileDataSize),
				found:    fmt.Sprintf("%d", fileSize),
				filename: path.Join(init.cfg.DataDir, file.Name()),
			}
		}
		if metadata.State == MetadataInitStateCompleted && fileSize < fileDataSize {
			return nil, unexpectedFileSize{
				expected: fmt.Sprintf("%d", fileDataSize),
				found:    fmt.Sprintf("%d", fileSize),
				filename: path.Join(init.cfg.DataDir, file.Name()),
			}
		}
		bytesWritten += uint64(fileSize)
	}

	if metadata.State == MetadataInitStateCompleted {
		return &DiskState{InitStateCompleted, bytesWritten}, nil
	}

	switch metadata.State {
	case MetadataInitStateStopped:
		return &DiskState{InitStateStopped, bytesWritten}, nil
	case MetadataInitStateStarted:
		if bytesWritten > 0 {
			return &DiskState{InitStateCrashed, bytesWritten}, nil
		} else {
			return &DiskState{InitStateNotStarted, 0}, nil
		}
	default:
		return nil, ErrStateInconsistent
	}
}

func (init *Initializer) SaveMetadata(state metadataInitState) error {
	err := os.MkdirAll(init.cfg.DataDir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %v", err)
	}

	data, err := json.Marshal(metadata{*init.cfg, init.id, state})
	if err != nil {
		return fmt.Errorf("serialization failure: %v", err)
	}

	err = ioutil.WriteFile(filepath.Join(init.cfg.DataDir, metadataFileName), data, shared.OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil
}

func (init *Initializer) LoadMetadata() (*metadata, error) {
	filename := filepath.Join(init.cfg.DataDir, metadataFileName)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrStateMetadataFileMissing
		}
		return nil, fmt.Errorf("read file failure: %v", err)
	}

	metadata := metadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (init *Initializer) SetLogger(logger Logger) {
	init.logger = logger
}

func (init *Initializer) isInitFile(file os.FileInfo) bool {
	return shared.IsInitFile(init.id, file)
}
