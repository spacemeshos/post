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

	logger            Logger
	powDifficultyFunc func(uint64) []byte
}

func (o *option) validate() error {
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

// withDifficultyFunc sets the difficulty function for the initializer.
// NOTE: This is an internal option for tests and should not be used by external packages.
func withDifficultyFunc(powDifficultyFunc func(uint64) []byte) OptionFunc {
	return func(opts *option) error {
		if powDifficultyFunc == nil {
			return errors.New("difficulty function is nil")
		}
		opts.powDifficultyFunc = powDifficultyFunc
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

	nonce        atomic.Pointer[uint64]
	lastPosition atomic.Pointer[uint64]

	numLabelsWritten atomic.Uint64
	diskState        *DiskState
	mtx              sync.RWMutex

	logger            Logger
	powDifficultyFunc func(uint64) []byte
}

func NewInitializer(opts ...OptionFunc) (*Initializer, error) {
	options := &option{
		logger: shared.DisabledLogger{},

		powDifficultyFunc: shared.PowDifficulty,
	}

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	if err := options.validate(); err != nil {
		return nil, err
	}

	init := &Initializer{
		cfg:               *options.cfg,
		opts:              *options.initOpts,
		nodeId:            options.nodeId,
		commitmentAtxId:   options.commitmentAtxId,
		commitment:        options.commitment,
		diskState:         NewDiskState(options.initOpts.DataDir, uint(options.cfg.BitsPerLabel)),
		logger:            options.logger,
		powDifficultyFunc: options.powDifficultyFunc,
	}

	numLabelsWritten, err := init.diskState.NumLabelsWritten()
	if err != nil {
		return nil, err
	}

	if numLabelsWritten > 0 {
		m, err := init.loadMetadata()
		if err != nil {
			return nil, err
		}
		if err := init.verifyMetadata(m); err != nil {
			return nil, err
		}
		init.nonce.Store(m.Nonce)
		init.lastPosition.Store(m.LastPosition)
	}

	if err := init.saveMetadata(); err != nil {
		return nil, err
	}

	return init, nil
}

// Initialize is the process in which the prover commits to store some data, by having its storage filled with
// pseudo-random data with respect to a specific id. This data is the result of a computationally-expensive operation.
func (init *Initializer) Initialize(ctx context.Context) error {
	if !init.mtx.TryLock() {
		return ErrAlreadyInitializing
	}
	defer init.mtx.Unlock()

	init.logger.Info("initialization: datadir: %v, number of units: %v, max file size: %v, number of labels per unit: %v, number of bits per label: %v",
		init.opts.DataDir, init.opts.NumUnits, init.opts.MaxFileSize, init.cfg.LabelsPerUnit, init.cfg.BitsPerLabel)

	layout := deriveFilesLayout(init.cfg, init.opts)
	init.logger.Info("initialization: files layout: number of files: %v, number of labels per file: %v, last file number of labels: %v",
		layout.NumFiles, layout.FileNumLabels, layout.LastFileNumLabels)
	if err := init.removeRedundantFiles(layout); err != nil {
		return err
	}

	numLabels := uint64(init.opts.NumUnits) * init.cfg.LabelsPerUnit
	difficulty := init.powDifficultyFunc(numLabels)
	batchSize := uint64(config.DefaultComputeBatchSize)

	for i := 0; i < int(layout.NumFiles); i++ {
		fileOffset := uint64(i) * layout.FileNumLabels
		fileNumLabels := layout.FileNumLabels
		if i == int(layout.NumFiles)-1 {
			fileNumLabels = layout.LastFileNumLabels
		}

		if err := init.initFile(ctx, i, batchSize, fileOffset, fileNumLabels, difficulty); err != nil {
			return err
		}
	}

	if init.nonce.Load() != nil {
		return nil
	}

	init.logger.Info("initialization: no nonce found while computing leaves, continue searching")
	if init.lastPosition.Load() == nil || *init.lastPosition.Load() < numLabels {
		lastPos := numLabels
		init.lastPosition.Store(&lastPos)
	}

	// continue searching for a nonce
	defer init.saveMetadata()
	for i := *init.lastPosition.Load(); i < math.MaxUint64; i += batchSize {
		lastPos := i
		init.lastPosition.Store(&lastPos)

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

			init.nonce.Store(res.Nonce)
			return nil
		}
	}

	return fmt.Errorf("no nonce found")
}

func (init *Initializer) removeRedundantFiles(layout filesLayout) error {
	numFiles, err := init.diskState.NumFilesWritten()
	if err != nil {
		return err
	}

	for i := int(layout.NumFiles); i < numFiles; i++ {
		name := shared.InitFileName(i)
		init.logger.Info("initialization: removing redundant file: %v", name)
		if err := init.RemoveFile(name); err != nil {
			return err
		}
	}

	return nil
}

func (init *Initializer) NumLabelsWritten() uint64 {
	return init.numLabelsWritten.Load()
}

func (init *Initializer) Nonce() *uint64 {
	return init.nonce.Load()
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
		name := file.Name()
		if shared.IsInitFile(info) || name == metadataFileName {
			if err := init.RemoveFile(name); err != nil {
				return err
			}
		}
	}

	return nil
}

func (init *Initializer) RemoveFile(name string) error {
	path := filepath.Join(init.opts.DataDir, name)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file (%v): %w", path, err)
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

func (init *Initializer) initFile(ctx context.Context, fileIndex int, batchSize, fileOffset, fileNumLabels uint64, difficulty []byte) error {
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

	switch {
	case numLabelsWritten == fileNumLabels:
		init.logger.Info("initialization: file #%v already initialized; number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileOffset)
		init.numLabelsWritten.Store(fileTargetPosition)
		return nil

	case numLabelsWritten > fileNumLabels:
		init.logger.Info("initialization: truncating file #%v; current number of labels: %v, target number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileNumLabels, fileOffset)
		if err := writer.Truncate(fileNumLabels); err != nil {
			return err
		}
		init.numLabelsWritten.Store(fileTargetPosition)
		return nil

	case numLabelsWritten > 0:
		init.logger.Info("initialization: continuing to write file #%v; current number of labels: %v, target number of labels: %v, start position: %v", fileIndex, numLabelsWritten, fileNumLabels, fileOffset)

	default:
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
		if init.nonce.Load() != nil {
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

			init.nonce.Store(res.Nonce)
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

func (init *Initializer) verifyMetadata(m *shared.PostMetadata) error {
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

	if init.opts.MaxFileSize != m.MaxFileSize {
		return ConfigMismatchError{
			Param:    "MaxFileSize",
			Expected: fmt.Sprintf("%d", init.opts.MaxFileSize),
			Found:    fmt.Sprintf("%d", m.MaxFileSize),
			DataDir:  init.opts.DataDir,
		}
	}

	if init.opts.NumUnits > m.NumUnits {
		return ConfigMismatchError{
			Param:    "NumUnits",
			Expected: fmt.Sprintf(">= %d", init.opts.NumUnits),
			Found:    fmt.Sprintf("%d", m.NumUnits),
			DataDir:  init.opts.DataDir,
		}
	}

	return nil
}

func (init *Initializer) saveMetadata() error {
	v := shared.PostMetadata{
		NodeId:          init.nodeId,
		CommitmentAtxId: init.commitmentAtxId,
		BitsPerLabel:    init.cfg.BitsPerLabel,
		LabelsPerUnit:   init.cfg.LabelsPerUnit,
		NumUnits:        init.opts.NumUnits,
		MaxFileSize:     init.opts.MaxFileSize,
		Nonce:           init.nonce.Load(),
		LastPosition:    init.lastPosition.Load(),
	}
	return SaveMetadata(init.opts.DataDir, &v)
}

func (init *Initializer) loadMetadata() (*shared.PostMetadata, error) {
	return LoadMetadata(init.opts.DataDir)
}
