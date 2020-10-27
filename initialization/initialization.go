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
	ErrStateMetadataFileMissing     = errors.New("metadata file is missing")
)

type ConfigMismatchError struct {
	Param    string
	Expected string
	Found    string
	Datadir  string
}

func (err ConfigMismatchError) Error() string {
	return fmt.Sprintf("`%v` config mismatch; expected: %v, found: %v, datadir: %v",
		err.Param, err.Expected, err.Found, err.Datadir)
}

type DiskState struct {
	NumLabelsWritten uint64
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

	if numLabelsWritten, err := init.NumLabelsWritten(); err != nil {
		return err
	} else if numLabelsWritten > 0 {
		if err := init.VerifyMetadata(); err != nil {
			return err
		}
	}

	if err := init.SaveMetadata(); err != nil {
		return err
	}

	fileNumLabels := init.cfg.NumLabels / uint64(init.cfg.NumFiles)

	init.logger.Info("initialization: starting to write %v file(s); number of labels: %v, number of labels per file: %v, label bit size: %v, compute batch size: %v, datadir: %v",
		init.cfg.NumFiles, init.cfg.NumLabels, fileNumLabels, init.cfg.LabelSize, init.cfg.ComputeBatchSize, init.cfg.DataDir)

	for i := 0; i < int(init.cfg.NumFiles); i++ {
		if err := init.initFile(computeProviderID, i, fileNumLabels); err != nil {
			return err
		}
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
		if shared.IsInitFile(file) || file.Name() == metadataFileName {
			path := filepath.Join(init.cfg.DataDir, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file (%v): %v", path, err)
			}
		}
	}

	return nil
}

func (init *Initializer) VerifyStarted() error {
	numLabelsWritten, err := init.NumLabelsWritten()
	if err != nil {
		return err
	}

	if numLabelsWritten == 0 {
		return shared.ErrInitNotStarted
	}

	return nil
}

func (init *Initializer) VerifyNotCompleted() error {
	numLabelsWritten, err := init.NumLabelsWritten()
	if err != nil {
		return err
	}

	if numLabelsWritten == init.cfg.NumLabels {
		return shared.ErrInitCompleted
	}

	return nil
}

func (init *Initializer) VerifyCompleted() error {
	numLabelsWritten, err := init.NumLabelsWritten()
	if err != nil {
		return err
	}

	if numLabelsWritten != init.cfg.NumLabels {
		return shared.ErrInitNotCompleted
	}

	return nil
}

func (init *Initializer) initFile(computeProviderID uint, fileIndex int, fileNumLabels uint64) error {
	fileOffset := uint64(fileIndex) * fileNumLabels
	fileTargetPosition := fileOffset + fileNumLabels
	batchSize := init.cfg.ComputeBatchSize

	// Initialize the labels file writer.
	writer, err := persistence.NewLabelsWriter(init.cfg.DataDir, fileIndex, init.cfg.LabelSize)
	if err != nil {
		return err
	}

	numLabelsWritten, err := writer.NumLabelsWritten()
	if err != nil {
		return err
	}

	if numLabelsWritten > 0 {
		if numLabelsWritten == fileNumLabels {
			init.logger.Info("initialization: file #%v already initialized; number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileOffset)
			init.updateProgress(float64(fileTargetPosition) / float64(init.cfg.NumLabels))
			return nil
		}

		if numLabelsWritten > fileNumLabels {
			init.logger.Info("initialization: truncating file #%v; current number of labels: %v, target number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileNumLabels, fileOffset)
			if err := writer.Truncate(fileNumLabels); err != nil {
				return err
			}
			init.updateProgress(float64(fileTargetPosition) / float64(init.cfg.NumLabels))
			return nil
		}

		init.logger.Info("initialization: continuing to write file #%v; current number of labels: %v, target number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileNumLabels, fileOffset)
	} else {
		init.logger.Info("initialization: starting to write file #%v; target number of labels: %v, start position: %v", fileIndex, fileNumLabels, fileOffset)
	}

	currentPosition := numLabelsWritten
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
		for currentPosition < fileNumLabels {
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
			remaining := uint(fileNumLabels - currentPosition)
			if remaining < batchSize {
				batchSize = remaining
			}

			init.logger.Debug("initialization: file #%v current position: %v, remaining: %v", fileIndex, currentPosition, remaining)

			// Calculate labels of the batch position range.
			startPosition := fileOffset + currentPosition
			endPosition := startPosition + uint64(batchSize) - 1
			output, err := oracle.WorkOracle(computeProviderID, init.id, startPosition, endPosition, uint32(init.cfg.LabelSize))
			if err != nil {
				computeErr <- err
				return
			}
			outputChan <- output
			currentPosition += uint64(batchSize)

			init.updateProgress(float64(fileOffset+currentPosition) / float64(init.cfg.NumLabels))
		}
	}()

	// Start IO worker.
	go func() {
		defer wg.Done()
		for {
			batch, more := <-outputChan
			if !more {
				_ = writer.Flush()
				return
			}

			// Write labels batch to disk.
			if err := writer.Write(batch); err != nil {
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

	numLabelsWritten, err = writer.NumLabelsWritten()
	if err != nil {
		return err
	}

	init.logger.Info("initialization: file #%v completed; number of labels written: %v", fileIndex, numLabelsWritten)

	return nil
}

func (init *Initializer) updateProgress(pct float64) {
	if init.progressChan != nil {
		init.progressChan <- pct
	}
}

func (init *Initializer) VerifyMetadata() error {
	metadata, err := init.LoadMetadata()
	if err != nil {
		return err
	}

	if bytes.Compare(init.id, metadata.ID) != 0 {
		return ConfigMismatchError{
			Param:    "ID",
			Expected: fmt.Sprintf("%x", init.id),
			Found:    fmt.Sprintf("%x", metadata.ID),
			Datadir:  init.cfg.DataDir,
		}
	}

	if init.cfg.LabelSize != metadata.LabelSize {
		return ConfigMismatchError{
			Param:    "LabelSize",
			Expected: fmt.Sprintf("%d", init.cfg.LabelSize),
			Found:    fmt.Sprintf("%d", metadata.LabelSize),
			Datadir:  init.cfg.DataDir,
		}
	}

	if init.cfg.NumFiles != metadata.NumFiles {
		return ConfigMismatchError{
			Param:    "NumFiles",
			Expected: fmt.Sprintf("%d", init.cfg.NumFiles),
			Found:    fmt.Sprintf("%d", metadata.NumFiles),
			Datadir:  init.cfg.DataDir,
		}
	}

	// `NumLabels` alternation isn't supported (yet) for `NumFiles` > 1.
	if init.cfg.NumLabels != metadata.NumLabels && init.cfg.NumFiles > 1 {
		return ConfigMismatchError{
			Param:    "NumLabels",
			Expected: fmt.Sprintf("%d", init.cfg.NumLabels),
			Found:    fmt.Sprintf("%d", metadata.NumLabels),
			Datadir:  init.cfg.DataDir,
		}
	}

	return nil
}

func (init *Initializer) NumLabelsWritten() (uint64, error) {
	files, err := ioutil.ReadDir(init.cfg.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	initFiles := make([]os.FileInfo, 0)
	for _, file := range files {
		if shared.IsInitFile(file) {
			initFiles = append(initFiles, file)
		}
	}

	if len(initFiles) == 0 {
		return 0, nil
	}

	var bytesWritten uint64
	for _, file := range initFiles {
		bytesWritten += uint64(file.Size())
	}

	return shared.NumLabels(bytesWritten, init.cfg.LabelSize), nil
}

func (init *Initializer) SaveMetadata() error {
	err := os.MkdirAll(init.cfg.DataDir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %v", err)
	}

	data, err := json.Marshal(metadata{ID: init.id, NumLabels: init.cfg.NumLabels, NumFiles: init.cfg.NumFiles, LabelSize: init.cfg.LabelSize})
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
