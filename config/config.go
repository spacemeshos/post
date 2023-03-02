package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spacemeshos/post/shared"
)

const (
	DefaultDataDirName = "data"

	DefaultMaxFileSize = uint64(4294967296) // 4 GB

	// DefaultComputeBatchSize value must be divisible by 8, to guarantee that writing to disk
	// and file truncating is byte-granular regardless of `BitsPerLabel` value.
	DefaultComputeBatchSize = 1 << 14

	DefaultBitsPerLabel = 8
	DefaultMaxNumUnits  = 10
	DefaultMinNumUnits  = 1

	// These values have been derived from https://colab.research.google.com/github/spacemeshos/notebooks/blob/main/post-proof-params.ipynb
	// The values here are only intended to be used for tests and are not optimized for performance or security!

	DefaultLabelsPerUnit = 1 << 11 // 2KB per unit.

	DefaultK1             = 200
	DefaultK2             = 212
	DefaultNonceBatchSize = 32
	DefaultAESBatchSize   = 16
)

const (
	MaxBitsPerLabel = 256
	MinBitsPerLabel = 1

	MinFileSize = 1024
)

var DefaultDataDir string

func init() {
	home, _ := os.UserHomeDir()
	DefaultDataDir = filepath.Join(home, "post", DefaultDataDirName)
}

type Config struct {
	MinNumUnits   uint32
	MaxNumUnits   uint32
	BitsPerLabel  uint8
	LabelsPerUnit uint64

	K1 uint32 // K1 specifies the difficulty for a label to be a candidate for a proof.
	K2 uint32 // K2 is the number of labels below the required difficulty required for a proof.
	B  uint32 // B is the number of labels used per AES invocation when generating a proof. Lower values speed up verification, higher values proof generation.
	N  uint32 // N is the number of nonces tried at the same time when generating a proof.
	// TODO(mafa): N should probably either be derived from the other parameters or be a configuration option of the node.
}

func DefaultConfig() Config {
	return Config{
		BitsPerLabel:  DefaultBitsPerLabel,
		LabelsPerUnit: DefaultLabelsPerUnit,
		MaxNumUnits:   DefaultMaxNumUnits,
		MinNumUnits:   DefaultMinNumUnits,
		K1:            DefaultK1,
		K2:            DefaultK2,
		B:             DefaultAESBatchSize,
		N:             DefaultNonceBatchSize,
	}
}

type InitOpts struct {
	DataDir           string
	NumUnits          uint32
	MaxFileSize       uint64
	ComputeProviderID int
	Throttle          bool
}

// BestProviderID can be used for selecting the most performant provider
// based on a short benchmarking session.
const BestProviderID = -1

func DefaultInitOpts() InitOpts {
	return InitOpts{
		DataDir:           DefaultDataDir,
		NumUnits:          DefaultMinNumUnits + 1,
		MaxFileSize:       DefaultMaxFileSize,
		ComputeProviderID: BestProviderID,
		Throttle:          false,
	}
}

func Validate(cfg Config, opts InitOpts) error {
	if opts.NumUnits < cfg.MinNumUnits {
		return fmt.Errorf("invalid `opts.NumUnits`; expected: >= %d, given: %d", cfg.MinNumUnits, opts.NumUnits)
	}

	if opts.NumUnits > cfg.MaxNumUnits {
		return fmt.Errorf("invalid `opts.NumUnits`; expected: <= %d, given: %d", cfg.MaxNumUnits, opts.NumUnits)
	}

	if opts.MaxFileSize < MinFileSize {
		return fmt.Errorf("invalid `opts.MaxFileSize`; expected: >= %d, given: %d", MinFileSize, opts.MaxFileSize)
	}

	if int(cfg.BitsPerLabel) > MaxBitsPerLabel {
		return fmt.Errorf("invalid `cfg.BitsPerLabel`; expected: <= %d, given: %d", MaxBitsPerLabel, cfg.BitsPerLabel)
	}

	if cfg.BitsPerLabel < MinBitsPerLabel {
		return fmt.Errorf("invalid `cfg.BitsPerLabel`; expected: >= %d, given: %d", MinBitsPerLabel, cfg.BitsPerLabel)
	}

	if res := shared.Uint64MulOverflow(cfg.LabelsPerUnit, uint64(opts.NumUnits)); res {
		return fmt.Errorf("uint64 overflow: `cfg.LabelsPerUnit` (%v) * `opts.NumUnits` (%v) exceeds the range allowed by uint64",
			cfg.LabelsPerUnit, opts.NumUnits)
	}

	numLabels := cfg.LabelsPerUnit * uint64(opts.NumUnits)
	if res := shared.Uint64MulOverflow(numLabels, uint64(cfg.K1)); res {
		return fmt.Errorf("uint64 overflow: `cfg.LabelsPerUnit` * `opts.NumUnits` (%v) * `cfg.K1` (%v) exceeds the range allowed by uint64",
			numLabels, cfg.K1)
	}

	return nil
}
