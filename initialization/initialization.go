package initialization

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
)

type (
	Config              = config.Config
	InitOpts            = config.InitOpts
	Logger              = zap.Logger
	ConfigMismatchError = shared.ConfigMismatchError
	Provider            = postrs.Provider
)

type Status int

const (
	StatusNotStarted Status = iota
	StatusStarted
	StatusInitializing
	StatusCompleted
	StatusError
)

// Providers returns a list of available compute providers.
func OpenCLProviders() ([]Provider, error) {
	return postrs.OpenCLProviders()
}

// CPUProviderID returns the ID of the CPU provider or nil if the CPU provider is not available.
func CPUProviderID() uint32 {
	return postrs.CPUProviderID()
}

type option struct {
	nodeId          []byte
	commitmentAtxId []byte

	commitment []byte

	cfg      *Config
	initOpts *config.InitOpts

	logger            *Logger
	powDifficultyFunc func(uint64) []byte
	referenceOracle   *oracle.WorkOracle
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
func WithLogger(logger *zap.Logger) OptionFunc {
	return func(opts *option) error {
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

// withReferenceOracle sets the reference oracle for the initializer.
// NOTE: This is an internal option for tests and should not be used by external packages.
func withReferenceOracle(referenceOracle *oracle.WorkOracle) OptionFunc {
	return func(opts *option) error {
		if referenceOracle == nil {
			return errors.New("reference oracle is nil")
		}
		opts.referenceOracle = referenceOracle
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

	// these values are atomics so they can be read from multiple other goroutines safely
	// write is protected by mtx
	nonce            atomic.Pointer[uint64]
	nonceValue       atomic.Pointer[[]byte]
	lastPosition     atomic.Pointer[uint64]
	numLabelsWritten atomic.Uint64

	diskState *DiskState
	mtx       sync.RWMutex // TODO(mafa): instead of a RWMutex we should lock with a lock file to prevent other processes from initializing/modifying the data

	logger            *Logger
	referenceOracle   *oracle.WorkOracle
	powDifficultyFunc func(uint64) []byte
}

func NewInitializer(opts ...OptionFunc) (*Initializer, error) {
	options := &option{
		logger: zap.NewNop(),

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
		diskState:         NewDiskState(options.initOpts.DataDir, uint(config.BitsPerLabel)),
		logger:            options.logger,
		powDifficultyFunc: options.powDifficultyFunc,
		referenceOracle:   options.referenceOracle,
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

		switch {
		case m.NonceValue != nil:
			// there is already a nonce value in the metadata
			nonceValue := make([]byte, postrs.LabelLength)
			copy(nonceValue, m.NonceValue)
			init.nonceValue.Store(&nonceValue)
		case m.Nonce != nil:
			// there is a nonce in the metadata but no nonce value
			cpuProviderID := CPUProviderID()
			wo, err := oracle.New(
				oracle.WithProviderID(&cpuProviderID),
				oracle.WithCommitment(init.commitment),
				oracle.WithVRFDifficulty(make([]byte, 32)), // we are not looking for it, so set difficulty to 0
				oracle.WithScryptParams(init.opts.Scrypt),
				oracle.WithLogger(init.logger),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create work oracle: %w", err)
			}
			defer wo.Close()

			result, err := wo.Position(*m.Nonce)
			if err != nil {
				return nil, fmt.Errorf("failed to compute nonce value: %w", err)
			}
			nonceValue := make([]byte, postrs.LabelLength)
			copy(nonceValue, result.Output)
			init.nonceValue.Store(&nonceValue)
		default:
			// no nonce in the metadata
		}
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

	layout, err := deriveFilesLayout(init.cfg, init.opts)
	if err != nil {
		return err
	}

	init.logger.Info("initialization started",
		zap.String("datadir", init.opts.DataDir),
		zap.Uint32("numUnits", init.opts.NumUnits),
		zap.Uint64("maxFileSize", init.opts.MaxFileSize),
		zap.Uint64("labelsPerUnit", init.cfg.LabelsPerUnit),
	)

	init.logger.Info("initialization file layout",
		zap.Uint64("labelsPerFile", layout.FileNumLabels),
		zap.Uint64("labelsLastFile", layout.LastFileNumLabels),
		zap.Int("firstFileIndex", layout.FirstFileIdx),
		zap.Int("lastFileIndex", layout.LastFileIdx),
	)
	if err := removeRedundantFiles(init.cfg, init.opts, init.logger); err != nil {
		return err
	}

	numLabels := uint64(init.opts.NumUnits) * init.cfg.LabelsPerUnit
	difficulty := init.powDifficultyFunc(numLabels)
	batchSize := init.opts.ComputeBatchSize

	wo, err := oracle.New(
		oracle.WithProviderID(init.opts.ProviderID),
		oracle.WithCommitment(init.commitment),
		oracle.WithVRFDifficulty(difficulty),
		oracle.WithScryptParams(init.opts.Scrypt),
		oracle.WithLogger(init.logger),
	)
	if err != nil {
		return err
	}
	defer wo.Close()

	woReference := init.referenceOracle
	if woReference == nil {
		cpuProvider := CPUProviderID()
		woReference, err = oracle.New(
			oracle.WithProviderID(&cpuProvider),
			oracle.WithCommitment(init.commitment),
			oracle.WithVRFDifficulty(difficulty),
			oracle.WithScryptParams(init.opts.Scrypt),
			oracle.WithLogger(init.logger),
		)
		if err != nil {
			return err
		}
		defer woReference.Close()
	}

	for i := layout.FirstFileIdx; i <= layout.LastFileIdx; i++ {
		fileOffset := uint64(i) * layout.FileNumLabels
		fileNumLabels := layout.FileNumLabels
		if i == layout.LastFileIdx {
			fileNumLabels = layout.LastFileNumLabels
		}

		if err := init.initFile(ctx, wo, woReference, i, batchSize, fileOffset, fileNumLabels, difficulty); err != nil {
			return err
		}
	}

	if init.nonce.Load() != nil {
		init.logger.Info("initialization: completed, found nonce", zap.Uint64("nonce", *init.nonce.Load()))
		return nil
	}
	if layout.NumFiles() < init.opts.TotalFiles(init.cfg.LabelsPerUnit) {
		init.logger.Info("initialization: no nonce found while computing labels")
		return nil
	}

	init.logger.Info("initialization: no nonce found while computing labels, continue initializing")
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
			return ctx.Err()
		default:
			// continue looking for a nonce
		}

		init.logger.Debug("initialization: continue looking for a nonce",
			zap.Uint64("startPosition", i),
			zap.Uint64("batchSize", batchSize),
		)

		res, err := wo.Positions(i, i+batchSize-1)
		if err != nil {
			return err
		}
		if res.Nonce != nil {
			init.logger.Debug("initialization: found nonce",
				zap.Uint64("nonce", *res.Nonce),
			)

			init.nonce.Store(res.Nonce)
			return nil
		}
	}

	return fmt.Errorf("no nonce found")
}

func removeRedundantFiles(cfg config.Config, opts config.InitOpts, logger *zap.Logger) error {
	// Go over all postdata_N.bin files in the data directory and remove the ones that are not needed.
	// The files with indices from 0 to init.opts.TotalFiles(init.cfg.LabelsPerUnit) - 1 are preserved.
	// The rest are redundant and can be removed.
	maxFileIndex := opts.TotalFiles(cfg.LabelsPerUnit) - 1
	logger.Debug("attempting to remove redundant files above index", zap.Int("maxFileIndex", maxFileIndex))

	files, err := os.ReadDir(opts.DataDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := file.Name()
		fileIndex, err := shared.ParseFileIndex(name)
		if err != nil && name != MetadataFileName {
			// TODO(mafa): revert back to warning, see https://github.com/spacemeshos/go-spacemesh/issues/4789
			logger.Debug("found unrecognized file", zap.String("fileName", name))
			continue
		}
		if fileIndex > maxFileIndex {
			logger.Info("removing redundant file", zap.String("fileName", name))
			path := filepath.Join(opts.DataDir, name)
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file (%v): %w", path, err)
			}
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

func (init *Initializer) NonceValue() []byte {
	if init.nonceValue.Load() == nil {
		return nil
	}

	nonceValue := make([]byte, postrs.LabelLength)
	copy(nonceValue, *init.nonceValue.Load())
	return nonceValue
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
		if shared.IsInitFile(info) || name == MetadataFileName {
			path := filepath.Join(init.opts.DataDir, name)
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

func (init *Initializer) initFile(ctx context.Context, wo, woReference *oracle.WorkOracle, fileIndex int, batchSize, fileOffset, fileNumLabels uint64, difficulty []byte) error {
	fileTargetPosition := fileOffset + fileNumLabels

	// Initialize the labels file writer.
	writer, err := persistence.NewLabelsWriter(init.opts.DataDir, fileIndex, config.BitsPerLabel)
	if err != nil {
		return err
	}
	defer writer.Close()

	numLabelsWritten, err := writer.NumLabelsWritten()
	if err != nil {
		return err
	}

	fields := []zap.Field{
		zap.Int("fileIndex", fileIndex),
		zap.Uint64("currentNumLabels", numLabelsWritten),
		zap.Uint64("targetNumLabels", fileNumLabels),
		zap.Uint64("startPosition", fileOffset),
	}

	switch {
	case numLabelsWritten == fileNumLabels:
		init.logger.Info("initialization: file already initialized", fields...)
		init.numLabelsWritten.Store(fileTargetPosition)
		return nil

	case numLabelsWritten > fileNumLabels:
		init.logger.Info("initialization: truncating file")
		if err := writer.Truncate(fileNumLabels); err != nil {
			return err
		}
		init.numLabelsWritten.Store(fileTargetPosition)
		return nil

	case numLabelsWritten > 0:
		init.logger.Info("initialization: continuing to write file", fields...)

	default:
		init.logger.Info("initialization: starting to write file", fields...)
	}

	for currentPosition := numLabelsWritten; currentPosition < fileNumLabels; currentPosition += batchSize {
		select {
		case <-ctx.Done():
			init.logger.Info("initialization: stopped")
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

		init.logger.Debug("initialization: status",
			zap.Int("fileIndex", fileIndex),
			zap.Uint64("currentPosition", currentPosition),
			zap.Uint64("remaining", remaining),
		)

		// Calculate labels of the batch position range.
		startPosition := fileOffset + currentPosition
		endPosition := startPosition + uint64(batchSize) - 1

		res, err := wo.Positions(startPosition, endPosition)
		if err != nil {
			return fmt.Errorf("failed to compute labels: %w", err)
		}

		// sanity check with reference oracle
		reference, err := woReference.Position(endPosition)
		if err != nil {
			return fmt.Errorf("failed to compute reference label: %w", err)
		}
		if !bytes.Equal(res.Output[(batchSize-1)*postrs.LabelLength:], reference.Output) {
			return ErrReferenceLabelMismatch{
				Index:      endPosition,
				Commitment: init.commitment,
				Expected:   reference.Output,
				Actual:     res.Output[(batchSize-1)*postrs.LabelLength:],
			}
		}

		if res.Nonce != nil {
			candidate := res.Output[(*res.Nonce-startPosition)*postrs.LabelLength:]
			candidate = candidate[:postrs.LabelLength]

			fields := []zap.Field{
				zap.Int("fileIndex", fileIndex),
				zap.Uint64("nonce", *res.Nonce),
				zap.String("value", hex.EncodeToString(candidate)),
			}
			init.logger.Debug("initialization: found nonce", fields...)

			if init.nonceValue.Load() == nil || bytes.Compare(candidate, *init.nonceValue.Load()) < 0 {
				nonceValue := make([]byte, postrs.LabelLength)
				copy(nonceValue, candidate)

				init.logger.Info("initialization: found new best nonce", fields...)
				init.nonce.Store(res.Nonce)
				init.nonceValue.Store(&nonceValue)
				init.saveMetadata()
			}
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

	init.logger.Info("initialization: completed",
		zap.Int("fileIndex", fileIndex),
		zap.Uint64("numLabelsWritten", numLabelsWritten),
	)
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
		Version: 1,

		NodeId:          init.nodeId,
		CommitmentAtxId: init.commitmentAtxId,

		LabelsPerUnit: init.cfg.LabelsPerUnit,
		NumUnits:      init.opts.NumUnits,
		MaxFileSize:   init.opts.MaxFileSize,
		Scrypt:        init.opts.Scrypt,

		Nonce:        init.nonce.Load(),
		LastPosition: init.lastPosition.Load(),
	}
	if init.nonceValue.Load() != nil {
		v.NonceValue = *init.nonceValue.Load()
	}
	return SaveMetadata(init.opts.DataDir, &v)
}

func (init *Initializer) loadMetadata() (*shared.PostMetadata, error) {
	// TODO(mafa): migrate metadata if needed before loading it
	return LoadMetadata(init.opts.DataDir)
}
