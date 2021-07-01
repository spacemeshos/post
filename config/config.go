package config

import (
	"fmt"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil"
	"path/filepath"
)

const (
	DefaultDataDirName = "data"

	DefaultNumFiles = 1

	// DefaultComputeBatchSize value must be divisible by 8, to guarantee that writing to disk
	// and file truncating is byte-granular regardless of `BitsPerLabel` value.
	DefaultComputeBatchSize = 1 << 14

	// 1KB per unit. Temporary value.
	DefaultBitsPerLabel  = 8
	DefaultLabelsPerUnit = 1 << 10

	DefaultMaxNumUnits = 10
	DefaultMinNumUnits = 1

	DefaultK1 = 2000
	DefaultK2 = 1800
)

const (
	MaxBitsPerLabel = 256
	MinBitsPerLabel = 1

	MaxNumFiles = 32
	MinNumFiles = 1
)

var (
	DefaultDataDir = filepath.Join(smutil.GetUserHomeDirectory(), "post", DefaultDataDirName)
)

type Config struct {
	BitsPerLabel  uint
	LabelsPerUnit uint
	MinNumUnits   uint
	MaxNumUnits   uint
	K1            uint
	K2            uint
}

func DefaultConfig() Config {
	return Config{
		BitsPerLabel:  DefaultBitsPerLabel,
		LabelsPerUnit: DefaultLabelsPerUnit,
		MaxNumUnits:   DefaultMaxNumUnits,
		MinNumUnits:   DefaultMinNumUnits,
		K1:            DefaultK1,
		K2:            DefaultK2,
	}
}

type InitOpts struct {
	DataDir           string
	NumUnits          uint
	NumFiles          uint
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
		NumFiles:          DefaultNumFiles,
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

	if opts.NumFiles > MaxNumFiles {
		return fmt.Errorf("invalid `opts.NumFiles`; expected: <= %d, given: %d", MaxNumFiles, opts.NumFiles)
	}

	if opts.NumFiles < MinNumFiles {
		return fmt.Errorf("invalid `opts.NumFiles`; expected: >= %d, given: %d", MinNumFiles, opts.NumFiles)
	}

	if cfg.BitsPerLabel > MaxBitsPerLabel {
		return fmt.Errorf("invalid `cfg.BitsPerLabel`; expected: <= %d, given: %d", MaxBitsPerLabel, cfg.BitsPerLabel)
	}

	if cfg.BitsPerLabel < MinBitsPerLabel {
		return fmt.Errorf("invalid `cfg.BitsPerLabel`; expected: >= %d, given: %d", MinBitsPerLabel, cfg.BitsPerLabel)
	}

	if res := shared.Uint64MulOverflow(uint64(cfg.LabelsPerUnit), uint64(opts.NumUnits)); res {
		return fmt.Errorf("uint64 overflow: `cfg.LabelsPerUnit` (%v) * `opts.NumUnits` (%v) exceeds the range allowed by uint64",
			cfg.LabelsPerUnit, opts.NumUnits)
	}

	numLabels := cfg.LabelsPerUnit * opts.NumUnits

	if numLabels%opts.NumFiles != 0 {
		return fmt.Errorf("invalid `cfg.LabelsPerUnit` & `opts.NumUnits`; expected: `cfg.LabelsPerUnit` * `opts.NumUnits` to be evenly divisible by `opts.NumFiles` (%v), given: %d", opts.NumFiles, numLabels)
	}

	if res := shared.Uint64MulOverflow(uint64(numLabels), uint64(cfg.K1)); res {
		return fmt.Errorf("uint64 overflow: `cfg.LabelsPerUnit` * `opts.NumUnits` (%v) * `cfg.K1` (%v) exceeds the range allowed by uint64",
			numLabels, cfg.K1)
	}

	return nil
}
