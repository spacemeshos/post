package initialization

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

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

// Providers returns a list of available compute providers.
func Providers() []ComputeProvider {
	return gpu.Providers()
}

// CPUProviderID returns the ID of the CPU provider or nil if the CPU provider is not available.
func CPUProviderID() uint {
	return gpu.CPUProviderID()
}

type option struct {
	nodeId          []byte
	commitmentAtxId []byte

	commitment []byte

	cfg      *Config
	initOpts *config.InitOpts
	logger   Logger
}

func (o *option) verify() error {
	if o.nodeId == nil {
		return errors.New("`nodeId` is required")
	}

	if o.commitmentAtxId == nil {
		return errors.New("`commitmentAtxId` is required")
	}

	o.commitment = oracle.CommitmentBytes(o.nodeId, o.commitmentAtxId)

	if o.cfg == nil {
		return errors.New("no config provided")
	}

	if o.initOpts == nil {
		return errors.New("no init options provided")
	}

	return config.Validate(*o.cfg, *o.initOpts)
}

type OptionFunc func(*option) error

// WithNodeId sets the ID of the Node.
func WithNodeId(nodeId []byte) OptionFunc {
	return func(opts *option) error {
		if len(nodeId) != 32 {
			return fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(nodeId))
		}

		opts.nodeId = nodeId
		return nil
	}
}

// WithCommitmentAtxId sets the ID of the CommitmentATX.
func WithCommitmentAtxId(id []byte) OptionFunc {
	return func(opts *option) error {
		if len(id) != 32 {
			return fmt.Errorf("invalid `commitmentAtxId` length; expected: 32, given: %v", len(id))
		}

		opts.commitmentAtxId = id
		return nil
	}
}

// WithInitOpts sets the init options for the initializer.
func WithInitOpts(initOpts config.InitOpts) OptionFunc {
	return func(opts *option) error {
		opts.initOpts = &initOpts
		return nil
	}
}

// WithConfig sets the config for the initializer.
func WithConfig(cfg Config) OptionFunc {
	return func(opts *option) error {
		opts.cfg = &cfg
		return nil
	}
}

// WithLogger sets the logger for the initializer.
func WithLogger(logger Logger) OptionFunc {
	return func(opts *option) error {
		if logger == nil {
			return errors.New("logger is nil")
		}
		opts.logger = logger
		return nil
	}
}

// Initializer is responsible for initializing a new PoST commitment.
type Initializer struct {
	nodeId          []byte
	commitmentAtxId []byte

	commitment []byte

	cfg  Config
	opts InitOpts

	nonce            *uint64
	numLabelsWritten atomic.Uint64

	diskState *DiskState
	mtx       sync.RWMutex

	logger Logger
}

func NewInitializer(opts ...OptionFunc) (*Initializer, error) {
	options := &option{
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
		cfg:             *options.cfg,
		opts:            *options.initOpts,
		nodeId:          options.nodeId,
		commitmentAtxId: options.commitmentAtxId,
		commitment:      options.commitment,
		diskState:       NewDiskState(options.initOpts.DataDir, uint(options.cfg.BitsPerLabel)),
		logger:          options.logger,
	}, nil
}

// Initialize is the process in which the prover commits to store some data, by having its storage filled with
// pseudo-random data with respect to a specific id. This data is the result of a computationally-expensive operation.
func (init *Initializer) Initialize(ctx context.Context) error {
	if !init.mtx.TryLock() {
		return ErrAlreadyInitializing
	}
	defer init.mtx.Unlock()

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
		init.nonce = m.Nonce
	}

	if err := init.saveMetadata(); err != nil {
		return err
	}

	numLabels := uint64(init.opts.NumUnits) * uint64(init.cfg.LabelsPerUnit)
	fileNumLabels := numLabels / uint64(init.opts.NumFiles)
	difficulty := shared.PowDifficulty(numLabels)
	batchSize := uint64(config.DefaultComputeBatchSize)

	init.logger.Info("initialization: starting to write %v file(s); number of units: %v, number of labels per unit: %v, number of bits per label: %v, datadir: %v",
		init.opts.NumFiles, init.opts.NumUnits, init.cfg.LabelsPerUnit, init.cfg.BitsPerLabel, init.opts.DataDir)

	for i := 0; i < int(init.opts.NumFiles); i++ {
		if err := init.initFile(ctx, i, batchSize, numLabels, fileNumLabels, difficulty); err != nil {
			return err
		}
	}

	if init.nonce != nil {
		return nil
	}

	init.logger.Info("initialization: no nonce found while computing leaves, continue searching")

	// continue searching for a nonce
	// TODO(mafa): depending on the difficulty function this can take a VERY long time, with the current difficulty function
	// ~ 37% of all smeshers won't find a nonce while computing leaves
	// ~ 14% of all smeshers won't find a nonce even after checking 2x numLabels
	// ~  5% of all smeshers won't find a nonce even after checking 3x numLabels
	// ~  2% of all smeshers won't find a nonce even after checking 4x numLabels
	for i := numLabels; i < math.MaxUint64; i += batchSize {
		select {
		case <-ctx.Done():
			init.logger.Info("initialization: stopped")
			if res := gpu.Stop(); res != gpu.StopResultOk {
				return fmt.Errorf("gpu stop error: %s", res)
			}
			return ctx.Err()
		default:
			// continue looking for a nonce
		}

		init.logger.Debug("initialization: continue looking for a nonce: start position: %v, batch size: %v", i, batchSize)

		res, err := oracle.WorkOracle(
			oracle.WithComputeProviderID(uint(init.opts.ComputeProviderID)),
			oracle.WithCommitment(init.commitment),
			oracle.WithStartAndEndPosition(i, i+batchSize-1),
			oracle.WithComputePow(difficulty),
			oracle.WithComputeLeaves(false),
		)
		if err != nil {
			return err
		}
		if res.Nonce != nil {
			init.logger.Debug("initialization: found nonce: %d", *res.Nonce)

			init.nonce = new(uint64)
			*init.nonce = *res.Nonce

			init.saveMetadata()
			return nil
		}
	}

	return fmt.Errorf("no nonce found")
}

func (init *Initializer) SessionNumLabelsWritten() uint64 {
	return init.numLabelsWritten.Load()
}

func (init *Initializer) Reset() error {
	if !init.mtx.TryLock() {
		return ErrCannotResetWhileInitializing
	}
	defer init.mtx.Unlock()

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
	if !init.mtx.TryLock() {
		return StatusInitializing
	}
	defer init.mtx.Unlock()

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

func (init *Initializer) initFile(ctx context.Context, fileIndex int, batchSize, numLabels, fileNumLabels uint64, difficulty []byte) error {
	fileOffset := uint64(fileIndex) * fileNumLabels
	fileTargetPosition := fileOffset + fileNumLabels

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

	for currentPosition := numLabelsWritten; currentPosition < fileNumLabels; currentPosition += batchSize {
		select {
		case <-ctx.Done():
			init.logger.Info("initialization: stopped")
			if res := gpu.Stop(); res != gpu.StopResultOk {
				return fmt.Errorf("gpu stop error: %s", res)
			}
			if err := writer.Flush(); err != nil {
				return err
			}
			return ctx.Err()
		default:
			// continue initialization
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
		if init.nonce != nil {
			// don't look for a nonce, when we already have one
			difficulty = nil
		}

		res, err := oracle.WorkOracle(
			oracle.WithComputeProviderID(uint(init.opts.ComputeProviderID)),
			oracle.WithCommitment(init.commitment),
			oracle.WithStartAndEndPosition(startPosition, endPosition),
			oracle.WithBitsPerLabel(uint32(init.cfg.BitsPerLabel)),
			oracle.WithComputePow(difficulty),
		)
		if err != nil {
			return err
		}

		if res.Nonce != nil {
			init.logger.Info("initialization: file #%v, found nonce: %d", fileIndex, *res.Nonce)
			init.nonce = new(uint64)
			*init.nonce = *res.Nonce

			init.saveMetadata()
		}

		// Write labels batch to disk.
		if err := writer.Write(res.Output); err != nil {
			return err
		}

		init.numLabelsWritten.Store(fileOffset + currentPosition + uint64(batchSize))
	}

	if err := writer.Flush(); err != nil {
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
	if !bytes.Equal(init.nodeId, m.NodeId) {
		return ConfigMismatchError{
			Param:    "NodeId",
			Expected: fmt.Sprintf("%x", init.nodeId),
			Found:    fmt.Sprintf("%x", m.NodeId),
			DataDir:  init.opts.DataDir,
		}
	}

	if !bytes.Equal(init.commitmentAtxId, m.CommitmentAtxId) {
		return ConfigMismatchError{
			Param:    "CommitmentAtxId",
			Expected: fmt.Sprintf("%x", init.commitmentAtxId),
			Found:    fmt.Sprintf("%x", m.CommitmentAtxId),
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
		NodeId:          init.nodeId,
		CommitmentAtxId: init.commitmentAtxId,
		BitsPerLabel:    init.cfg.BitsPerLabel,
		LabelsPerUnit:   init.cfg.LabelsPerUnit,
		NumUnits:        init.opts.NumUnits,
		NumFiles:        init.opts.NumFiles,
		Nonce:           init.nonce,
	}
	return SaveMetadata(init.opts.DataDir, &v)
}

func (init *Initializer) loadMetadata() (*Metadata, error) {
	return LoadMetadata(init.opts.DataDir)
}
