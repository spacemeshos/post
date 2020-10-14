package config

import (
	"fmt"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil"
	"path/filepath"
)

const (
	// In bytes. 1 peta-byte of storage.
	// This would protect against number of labels uint64 overflow as well,
	// since the number of labels per byte can be 8 at most (3 extra left bit shifts).
	MaxDataSize      = 1 << 50
	MaxNumLabels     = 1<<54 - 1 // TODO: FIX (after API changes)
	MinFileNumLabels = 32

	MinFileDataSize = 32

	MaxNumFiles = 256
	MinNumFiles = 1

	MaxLabelSize = 256
	MinLabelSize = 1
)

const (
	DefaultDataDirName      = "data"
	DefaultNumFiles         = 1
	DefaultComputeBatchSize = 1 << 14

	// 1MB space. Temporary value.
	DefaultNumLabels = 1 << 20
	DefaultLabelSize = 8

	DefaultK1 = 1 << 10
	DefaultK2 = 100
)

var (
	DefaultDataDir = filepath.Join(smutil.GetUserHomeDirectory(), "post", DefaultDataDirName)
)

type Config struct {
	DataDir          string `mapstructure:"post-datadir"`
	NumFiles         uint   `mapstructure:"post-numfiles"`
	ComputeBatchSize uint   `mapstructure:"post-compute-batch-size"`

	// Protocol params.
	NumLabels uint64 `mapstructure:"post-numlabels"`
	LabelSize uint   `mapstructure:"post-labelsize"`
	K1        uint   `mapstructure:"post-k1"`
	K2        uint   `mapstructure:"post-k2"`
}

// TODO(moshababo): add tests for all cases
func (cfg *Config) Validate() error {
	dataSize := shared.DataSize(cfg.NumLabels, cfg.LabelSize)
	if dataSize > MaxDataSize {
		return fmt.Errorf("invalid data size; expected: <= %d, given: %d", MaxDataSize, dataSize)
	}

	if !shared.IsPowerOfTwo(uint64(cfg.NumFiles)) {
		return fmt.Errorf("invalid `NumFiles`; expected: a power of 2, given: %d", cfg.NumFiles)
	}

	if cfg.NumFiles > MaxNumFiles {
		return fmt.Errorf("invalid `NumFiles`; expected: <= %d, given: %d", MaxNumFiles, cfg.NumFiles)
	}

	if cfg.NumFiles < MinNumFiles {
		return fmt.Errorf("invalid `NumFiles`; expected: >= %d, given: %d", MinNumFiles, cfg.NumFiles)
	}

	if cfg.LabelSize > MaxLabelSize {
		return fmt.Errorf("invalid `LabelSize`; expected: <= %d, given: %d", MaxLabelSize, cfg.LabelSize)
	}

	if cfg.LabelSize < MinLabelSize {
		return fmt.Errorf("invalid `LabelSize`; expected: >= %d, given: %d", MinLabelSize, cfg.LabelSize)
	}

	if cfg.NumLabels%uint64(cfg.NumFiles) != 0 {
		return fmt.Errorf("invalid `NumLabels`; expected: evenly divisible by `NumFiles` (%v), given: %d", cfg.NumFiles, cfg.NumLabels)
	}

	// (ComputeBatchSize%8 == 0) will guarantee that labels writing is in byte-granularity, regardless of LabelSize.
	if cfg.ComputeBatchSize%8 != 0 {
		return fmt.Errorf("invalid `ComputeBatchSize`; expected: evenly divisible by 8, given: %d", cfg.ComputeBatchSize)
	}

	fileNumLabels := cfg.NumLabels / uint64(cfg.NumFiles)
	fileDataSize := shared.DataSize(fileNumLabels, cfg.LabelSize)
	if fileDataSize < MinFileDataSize {
		return fmt.Errorf("invalid file data size; expected: >= %d, given: %d", MinFileDataSize, fileDataSize)
	}

	if res := shared.Uint64MulOverflow(cfg.NumLabels, uint64(cfg.K1)); res {
		return fmt.Errorf("uint64 overflow: `NumLabels` (%v) multipled by K1 (%v) exceeds the range allowed by uint64",
			cfg.NumLabels, cfg.K1)
	}

	return nil
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:          DefaultDataDir,
		NumFiles:         DefaultNumFiles,
		ComputeBatchSize: DefaultComputeBatchSize,

		LabelSize: DefaultLabelSize,
		NumLabels: DefaultNumLabels,
		K1:        DefaultK1,
		K2:        DefaultK2,
	}
}
