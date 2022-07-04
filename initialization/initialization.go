package initialization

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
)

type (
	Config              = config.Config
	InitOpts            = config.InitOpts
	Proof               = shared.Proof
	Logger              = shared.Logger
	ConfigMismatchError = shared.ConfigMismatchError
	ComputeProvider     = gpu.ComputeProvider
)

var (
	ErrNotInitializing              = errors.New("not initializing")
	ErrAlreadyInitializing          = errors.New("already initializing")
	ErrCannotResetWhileInitializing = errors.New("cannot reset while initializing")
	ErrStopped                      = errors.New("gpu-post: stopped")
	ErrStateMetadataFileMissing     = errors.New("metadata file is missing")
)

func Providers() []ComputeProvider {
	return gpu.Providers()
}

func CPUProviderID() int {
	return gpu.CPUProviderID()
}

type Initializer struct {
	// data per file
	// numLabelsWritten should be aligned by 8 bytes because it's accessed by atomics.
	numLabelsWritten     uint64
	numLabelsWrittenChan chan uint64

	processedFiles uint64

	cfg  Config
	opts InitOpts
	id   []byte

	diskState    *DiskState
	initializing bool
	mtx          sync.RWMutex

	stopChan chan struct{}
	doneChan chan struct{}

	logger Logger
}

func NewInitializer(cfg Config, opts config.InitOpts, id []byte) (*Initializer, error) {
	if len(id) != 32 {
		return nil, fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(id))
	}

	if err := config.Validate(cfg, opts); err != nil {
		return nil, err
	}

	return &Initializer{
		cfg:       cfg,
		opts:      opts,
		id:        id,
		diskState: NewDiskState(opts.DataDir, cfg.BitsPerLabel),
		logger:    shared.DisabledLogger{},
	}, nil
}

// Initialize is the process in which the prover commits to store some data, by having its storage filled with
// pseudo-random data with respect to a specific id. This data is the result of a computationally-expensive operation.
func (init *Initializer) Initialize() error {
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
		init.mtx.Lock()
		init.initializing = false
		init.mtx.Unlock()

		close(init.doneChan)

		if init.numLabelsWrittenChan != nil {
			close(init.numLabelsWrittenChan)
			init.numLabelsWrittenChan = nil
		}
	}()

	if numLabelsWritten, err := init.diskState.NumLabelsWritten(); err != nil {
		return err
	} else if numLabelsWritten > 0 {
		m, err := init.loadMetadata()
		if err != nil {
			return err
		}
		if err := init.verifyMetadata(m); err != nil {
			return err
		}
	}

	if err := init.saveMetadata(); err != nil {
		return err
	}

	fileNumLabels := init.getFileNumLabels()

	init.logger.Info("initialization: starting to write %v file(s); number of units: %v, number of labels per unit: %v, number of bits per label: %v, datadir: %v",
		init.opts.NumFiles, init.opts.NumUnits, init.cfg.LabelsPerUnit, init.cfg.BitsPerLabel, init.opts.DataDir)

	for i := 0; i < int(init.opts.NumFiles); i++ {
		if err := init.initFile(uint(init.opts.ComputeProviderID), i, fileNumLabels); err != nil {
			return err
		}
		atomic.AddUint64(&init.processedFiles, 1)
	}

	return nil
}

func (init *Initializer) getFileNumLabels() uint64 {
	numLabels := uint64(init.opts.NumUnits) * uint64(init.cfg.LabelsPerUnit)
	return numLabels / uint64(init.opts.NumFiles)
}

func (init *Initializer) isInitializing() bool {
	init.mtx.RLock()
	defer init.mtx.RUnlock()

	return init.initializing
}

func (init *Initializer) Stop() error {
	if !init.isInitializing() {
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

func (init *Initializer) SessionNumLabelsWrittenChan() <-chan uint64 {
	init.mtx.Lock()
	defer init.mtx.Unlock()

	if init.numLabelsWrittenChan == nil {
		init.numLabelsWrittenChan = make(chan uint64, 1024)
	}

	return init.numLabelsWrittenChan
}

func (init *Initializer) SessionNumLabelsWritten() uint64 {
	return atomic.LoadUint64(&init.numLabelsWritten)
}

func (init *Initializer) Reset() error {
	if init.isInitializing() {
		return ErrCannotResetWhileInitializing
	}

	if err := init.VerifyStarted(); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(init.opts.DataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if shared.IsInitFile(file) || file.Name() == metadataFileName {
			path := filepath.Join(init.opts.DataDir, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file (%v): %v", path, err)
			}
		}
	}

	return nil
}

func (init *Initializer) Started() (bool, error) {
	numLabelsWritten, err := init.diskState.NumLabelsWritten()
	if err != nil {
		return false, err
	}

	return numLabelsWritten > 0, nil
}

func (init *Initializer) Completed() (bool, error) {
	numLabelsWritten, err := init.diskState.NumLabelsWritten()
	if err != nil {
		return false, err
	}

	target := uint64(init.opts.NumUnits) * uint64(init.cfg.LabelsPerUnit)
	return numLabelsWritten == target, nil
}

func (init *Initializer) VerifyStarted() error {
	ok, err := init.Started()
	if err != nil {
		return err
	}
	if ok == false {
		return shared.ErrInitNotStarted
	}

	return nil
}

func (init *Initializer) initFile(computeProviderID uint, fileIndex int, fileNumLabels uint64) error {
	fileOffset := uint64(fileIndex) * fileNumLabels
	fileTargetPosition := fileOffset + fileNumLabels
	batchSize := uint64(config.DefaultComputeBatchSize)

	// Initialize the labels file writer.
	writer, err := persistence.NewLabelsWriter(init.opts.DataDir, fileIndex, init.cfg.BitsPerLabel)
	if err != nil {
		return err
	}
	defer writer.Close()

	numLabelsWritten, err := writer.NumLabelsWritten()
	if err != nil {
		return err
	}

	if numLabelsWritten > 0 {
		if numLabelsWritten == fileNumLabels {
			init.logger.Info("initialization: file #%v already initialized; number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileOffset)
			init.updateSessionNumLabelsWritten(fileTargetPosition)
			return nil
		}

		if numLabelsWritten > fileNumLabels {
			init.logger.Info("initialization: truncating file #%v; current number of labels: %v, target number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileNumLabels, fileOffset)
			if err := writer.Truncate(fileNumLabels); err != nil {
				return err
			}
			init.updateSessionNumLabelsWritten(fileTargetPosition)
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
			remaining := fileNumLabels - currentPosition
			if remaining < batchSize {
				batchSize = remaining
			}

			init.logger.Debug("initialization: file #%v current position: %v, remaining: %v", fileIndex, currentPosition, remaining)

			// Calculate labels of the batch position range.
			startPosition := fileOffset + currentPosition
			endPosition := startPosition + uint64(batchSize) - 1
			output, err := oracle.WorkOracle(computeProviderID, init.id, startPosition, endPosition, uint32(init.cfg.BitsPerLabel))
			if err != nil {
				computeErr <- err
				return
			}
			outputChan <- output
			currentPosition += uint64(batchSize)

			init.updateSessionNumLabelsWritten(fileOffset + currentPosition)
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

func (init *Initializer) updateSessionNumLabelsWritten(numLabelsWritten uint64) {
	atomic.StoreUint64(&init.numLabelsWritten, numLabelsWritten)

	init.mtx.RLock()
	ch := init.numLabelsWrittenChan
	init.mtx.RUnlock()

	if ch != nil {
		ch <- numLabelsWritten
	}
}

func (init *Initializer) verifyMetadata(m *Metadata) error {
	if bytes.Compare(init.id, m.ID) != 0 {
		return ConfigMismatchError{
			Param:    "ID",
			Expected: fmt.Sprintf("%x", init.id),
			Found:    fmt.Sprintf("%x", m.ID),
			DataDir:  init.opts.DataDir,
		}
	}

	if init.cfg.BitsPerLabel != m.BitsPerLabel {
		return ConfigMismatchError{
			Param:    "BitsPerLabel",
			Expected: fmt.Sprintf("%d", init.cfg.BitsPerLabel),
			Found:    fmt.Sprintf("%d", m.BitsPerLabel),
			DataDir:  init.opts.DataDir,
		}
	}

	if init.cfg.LabelsPerUnit != m.LabelsPerUnit {
		return ConfigMismatchError{
			Param:    "LabelsPerUnit",
			Expected: fmt.Sprintf("%d", init.cfg.LabelsPerUnit),
			Found:    fmt.Sprintf("%d", m.LabelsPerUnit),
			DataDir:  init.opts.DataDir,
		}
	}

	if init.opts.NumFiles != m.NumFiles {
		return ConfigMismatchError{
			Param:    "NumFiles",
			Expected: fmt.Sprintf("%d", init.opts.NumFiles),
			Found:    fmt.Sprintf("%d", m.NumFiles),
			DataDir:  init.opts.DataDir,
		}
	}

	// `opts.NumUnits` alternation isn't supported (yet) while `opts.NumFiles` > 1.
	if init.opts.NumUnits != m.NumUnits && init.opts.NumFiles > 1 {
		return ConfigMismatchError{
			Param:    "NumUnits",
			Expected: fmt.Sprintf("%d", init.opts.NumUnits),
			Found:    fmt.Sprintf("%d", m.NumUnits),
			DataDir:  init.opts.DataDir,
		}
	}

	return nil
}

func (init *Initializer) saveMetadata() error {
	v := Metadata{
		ID:            init.id,
		BitsPerLabel:  init.cfg.BitsPerLabel,
		LabelsPerUnit: init.cfg.LabelsPerUnit,
		NumUnits:      init.opts.NumUnits,
		NumFiles:      init.opts.NumFiles,
	}
	return SaveMetadata(init.opts.DataDir, &v)
}

func (init *Initializer) loadMetadata() (*Metadata, error) {
	return LoadMetadata(init.opts.DataDir)
}

func (init *Initializer) SetLogger(logger Logger) {
	init.logger = logger
}
