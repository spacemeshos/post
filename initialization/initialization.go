package initialization

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

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

type Status int

const (
	StatusNotStarted Status = iota
	StatusStarted
	StatusInitializing
	StatusCompleted
	StatusError
)

var (
	ErrAlreadyInitializing          = errors.New("already initializing")
	ErrCannotResetWhileInitializing = errors.New("cannot reset while initializing")
	ErrStateMetadataFileMissing     = errors.New("metadata file is missing")
)

func Providers() []ComputeProvider {
	return gpu.Providers()
}

func CPUProviderID() int {
	return gpu.CPUProviderID()
}

func RemoveDataFiles(dataDir string) error {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		if shared.IsInitFile(info) || file.Name() == metadataFileName {
			path := filepath.Join(dataDir, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file (%v): %w", path, err)
			}
		}
	}

	return nil
}

type initializeOption struct {
	commitment []byte
	cfg        *Config
	initOpts   *config.InitOpts
	logger     Logger
}

func (opts *initializeOption) verify() error {
	if opts.cfg == nil {
		return errors.New("no config provided")
	}

	if opts.initOpts == nil {
		return errors.New("no init options provided")
	}

	if err := config.Validate(*opts.cfg, *opts.initOpts); err != nil {
		return err
	}
	return nil
}

type initializeOptionFunc func(*initializeOption) error

func WithCommitment(commitment []byte) initializeOptionFunc {
	return func(opts *initializeOption) error {
		if len(commitment) != 32 {
			return fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(commitment))
		}
		opts.commitment = commitment
		return nil
	}
}

func WithInitOpts(initOpts config.InitOpts) initializeOptionFunc {
	return func(opts *initializeOption) error {
		opts.initOpts = &initOpts
		return nil
	}
}

func WithConfig(cfg Config) initializeOptionFunc {
	return func(opts *initializeOption) error {
		opts.cfg = &cfg
		return nil
	}
}

func WithLogger(logger Logger) initializeOptionFunc {
	return func(opts *initializeOption) error {
		if logger == nil {
			return errors.New("logger is nil")
		}
		opts.logger = logger
		return nil
	}
}

type Initializer struct {
	numLabelsWritten atomic.Uint64

	cfg        Config
	opts       InitOpts
	commitment []byte

	diskState    *DiskState
	initializing bool
	mtx          sync.RWMutex

	logger Logger
}

func NewInitializer(opts ...initializeOptionFunc) (*Initializer, error) {
	options := &initializeOption{
		logger: shared.DisabledLogger{},
	}

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	if err := options.verify(); err != nil {
		return nil, err
	}

	return &Initializer{
		cfg:        *options.cfg,
		opts:       *options.initOpts,
		commitment: options.commitment,
		diskState:  NewDiskState(options.initOpts.DataDir, uint(options.cfg.BitsPerLabel)),
		logger:     options.logger,
	}, nil
}

// Initialize is the process in which the prover commits to store some data, by having its storage filled with
// pseudo-random data with respect to a specific id. This data is the result of a computationally-expensive operation.
func (init *Initializer) Initialize(ctx context.Context) error {
	init.mtx.Lock()

	if init.initializing {
		init.mtx.Unlock()
		return ErrAlreadyInitializing
	}

	init.initializing = true
	init.mtx.Unlock()

	defer func() {
		init.mtx.Lock()
		defer init.mtx.Unlock()
		init.initializing = false
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

	numLabels := uint64(init.opts.NumUnits) * uint64(init.cfg.LabelsPerUnit)
	fileNumLabels := numLabels / uint64(init.opts.NumFiles)

	init.logger.Info("initialization: starting to write %v file(s); number of units: %v, number of labels per unit: %v, number of bits per label: %v, datadir: %v",
		init.opts.NumFiles, init.opts.NumUnits, init.cfg.LabelsPerUnit, init.cfg.BitsPerLabel, init.opts.DataDir)

	for i := 0; i < int(init.opts.NumFiles); i++ {
		if err := init.initFile(ctx, uint(init.opts.ComputeProviderID), i, numLabels, fileNumLabels); err != nil {
			return err
		}
	}

	return nil
}

func (init *Initializer) isInitializing() bool {
	init.mtx.RLock()
	defer init.mtx.RUnlock()
	return init.initializing
}

func (init *Initializer) SessionNumLabelsWritten() uint64 {
	return init.numLabelsWritten.Load()
}

func (init *Initializer) Reset() error {
	switch init.Status() {
	case StatusInitializing:
		return ErrCannotResetWhileInitializing
	case StatusError:
		return fmt.Errorf("cannot determine status of initialization")
	}

	files, err := os.ReadDir(init.opts.DataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		if shared.IsInitFile(info) || file.Name() == metadataFileName {
			path := filepath.Join(init.opts.DataDir, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file (%v): %w", path, err)
			}
		}
	}

	return nil
}

func (init *Initializer) Status() Status {
	if init.isInitializing() {
		return StatusInitializing
	}

	numLabelsWritten, err := init.diskState.NumLabelsWritten()
	if err != nil {
		return StatusError
	}

	target := uint64(init.opts.NumUnits) * uint64(init.cfg.LabelsPerUnit)
	if numLabelsWritten == target {
		return StatusCompleted
	}

	if numLabelsWritten > 0 {
		return StatusStarted
	}

	return StatusNotStarted
}

func (init *Initializer) initFile(ctx context.Context, computeProviderID uint, fileIndex int, numLabels uint64, fileNumLabels uint64) error {
	fileOffset := uint64(fileIndex) * fileNumLabels
	fileTargetPosition := fileOffset + fileNumLabels
	batchSize := uint64(config.DefaultComputeBatchSize)

	// Initialize the labels file writer.
	writer, err := persistence.NewLabelsWriter(init.opts.DataDir, fileIndex, uint(init.cfg.BitsPerLabel))
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
			init.numLabelsWritten.Store(fileTargetPosition)
			return nil
		}

		if numLabelsWritten > fileNumLabels {
			init.logger.Info("initialization: truncating file #%v; current number of labels: %v, target number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileNumLabels, fileOffset)
			if err := writer.Truncate(fileNumLabels); err != nil {
				return err
			}
			init.numLabelsWritten.Store(fileTargetPosition)
			return nil
		}

		init.logger.Info("initialization: continuing to write file #%v; current number of labels: %v, target number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileNumLabels, fileOffset)
	} else {
		init.logger.Info("initialization: starting to write file #%v; target number of labels: %v, start position: %v", fileIndex, fileNumLabels, fileOffset)
	}

	currentPosition := numLabelsWritten
	outputChan := make(chan []byte, 1024)

	errGroup, ctx := errgroup.WithContext(ctx)

	// Start compute worker.
	errGroup.Go(func() error {
		defer close(outputChan)

		for currentPosition < fileNumLabels {
			select {
			case <-ctx.Done():
				init.logger.Info("initialization: stopped")

				if res := gpu.Stop(); res != gpu.StopResultOk {
					return fmt.Errorf("gpu stop error: %s", res)
				}

				return ctx.Err()
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
			output, err := oracle.WorkOracle(computeProviderID, init.commitment, startPosition, endPosition, uint32(init.cfg.BitsPerLabel))
			if err != nil {
				return err
			}
			outputChan <- output
			currentPosition += uint64(batchSize)

			init.numLabelsWritten.Store(fileOffset + currentPosition)
		}
		return nil
	})

	// Start IO worker.
	errGroup.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return writer.Flush()
			case batch, ok := <-outputChan:
				if !ok {
					return writer.Flush()
				}

				// Write labels batch to disk.
				if err := writer.Write(batch); err != nil {
					return err
				}
			}
		}
	})

	if err := errGroup.Wait(); err != nil {
		return err
	}

	numLabelsWritten, err = writer.NumLabelsWritten()
	if err != nil {
		return err
	}

	init.logger.Info("initialization: file #%v completed; number of labels written: %v", fileIndex, numLabelsWritten)
	return nil
}

func (init *Initializer) verifyMetadata(m *Metadata) error {
	if !bytes.Equal(init.commitment, m.Commitment) {
		return ConfigMismatchError{
			Param:    "Commitment",
			Expected: fmt.Sprintf("%x", init.commitment),
			Found:    fmt.Sprintf("%x", m.Commitment),
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
		Commitment:    init.commitment,
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
