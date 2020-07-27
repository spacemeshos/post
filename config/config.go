package config

import (
	"fmt"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil"
	"math"
	"path/filepath"
)

const (
	// In bytes. 1 peta-byte of storage.
	// This would protect against number of labels uint64 overflow as well,
	// since the number of labels per byte can be 8 at most (3 extra bit shifts).
	MaxSpace = 1 << 50

	MaxNumFiles = 256
	MinFileSize = 32
)

const (
	DefaultDataDirName          = "data"
	DefaultLabelsLogRate        = 5000000
	DefaultMaxFilesParallelism  = 1
	DefaultMaxInFileParallelism = 6
	DefaultMaxReadParallelism   = 6

	DefaultNumFiles = 1

	// 1MB space. Temporary value.
	DefaultNumLabels = 1 << 17
	DefaultLabelSize = 8

	DefaultK1 = 1 << 15
	DefaultK2 = 100
)

var (
	DefaultDataDir = filepath.Join(smutil.GetUserHomeDirectory(), "post", DefaultDataDirName)
)

type Config struct {
	DataDir                        string `mapstructure:"post-datadir"`
	LabelsLogRate                  uint64 `mapstructure:"post-lograte"`
	MaxWriteFilesParallelism       uint   `mapstructure:"post-parallel-files"`
	MaxWriteInFileParallelism      uint   `mapstructure:"post-parallel-infile"`
	MaxReadFilesParallelism        uint   `mapstructure:"post-parallel-read"`
	DisableSpaceAvailabilityChecks bool   `mapstructure:"post-disable-space-checks"`

	// Protocol params.
	NumLabels uint64 `mapstructure:"post-numlabels"`
	LabelSize uint   `mapstructure:"post-labelsize"`
	K1        uint   `mapstructure:"post-k1"`
	K2        uint   `mapstructure:"post-k2"`

	NumFiles uint `mapstructure:"post-numfiles"`
}

func (cfg *Config) Validate() error {
	if cfg.NumLabels == 0 {
		return fmt.Errorf("invalid NumLabels; expected: > 0, given: 0")
	}

	if res := uint64MulOverflow(cfg.NumLabels, uint64(cfg.K1)); res {
		return fmt.Errorf("uint64 overflow: NumLabels (%v) multipled by K1 (%v) exceeds the range allowed by uint64",
			cfg.NumLabels, cfg.K1)
	}

	space := cfg.Space()
	if space > MaxSpace {
		return fmt.Errorf("invalid space; expected: < %d, actual: %d", MaxSpace, space)
	}

	if !shared.IsPowerOfTwo(uint64(cfg.NumFiles)) {
		return fmt.Errorf("invalid NumFiles; expected: a power of 2, given: %d", cfg.NumFiles)
	}

	if cfg.NumFiles > MaxNumFiles {
		return fmt.Errorf("invalid NumFiles; expected: < %d, given: %d", MaxNumFiles, cfg.NumFiles)
	}

	fileSize := space / uint64(cfg.NumFiles)
	if fileSize < MinFileSize {
		return fmt.Errorf("invalid file size; expected: > %d, actual: %d", MinFileSize, fileSize)
	}

	return nil
}

func (cfg *Config) Space() uint64 {
	return cfg.NumLabels * uint64(cfg.LabelSize)
}

func (cfg *Config) ProvingDifficulty() uint64 {
	maxTarget := uint64(math.MaxUint64)
	K1 := uint64(cfg.K1)

	x := maxTarget / cfg.NumLabels
	y := maxTarget % cfg.NumLabels
	return x*K1 + (y*K1)/cfg.NumLabels
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:                        DefaultDataDir,
		LabelsLogRate:                  DefaultLabelsLogRate,
		MaxWriteFilesParallelism:       DefaultMaxFilesParallelism,
		MaxWriteInFileParallelism:      DefaultMaxInFileParallelism,
		MaxReadFilesParallelism:        DefaultMaxReadParallelism,
		DisableSpaceAvailabilityChecks: true, // TODO: permanently remove the checks if they are not reliable.

		LabelSize: DefaultLabelSize,
		NumLabels: DefaultNumLabels,
		K1:        DefaultK1,
		K2:        DefaultK2,

		NumFiles: DefaultNumFiles,
	}
}

func uint64MulOverflow(a, b uint64) bool {
	if a == 0 || b == 0 {
		return false
	}
	c := a * b
	return c/b != a
}
