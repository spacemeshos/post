package config

import (
	//"fmt"
	//"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil"
	"path/filepath"
)

const (
	// BitsPerLabel value is determined by the protocol.
	MaxBitsPerLabel = 256
	MinBitsPerLabel = 1

	// LabelsPerUnit value is determined by the protocol.
	MaxLabelsPerUnit = 1 << 20
	MinLabelsPerUnit = 32

	// NumUnits value is seleted by the user.
	MaxNumUnits = 10
	MinNumUnits = 1

	// NumFiles value is seleted by the user.
	MaxNumFiles = 256
	MinNumFiles = 1

	//MaxNumLabels     = 1 << 50
	//MinFileNumLabels = 32
)

const (
	DefaultDataDirName      = "data"
	DefaultNumFiles         = 1
	DefaultComputeBatchSize = 1 << 14

	// 1MB per unit. Temporary value.
	DefaultBitsPerLabel  = 8
	DefaultLabelsPerUnit = 1 << 20

	DefaultK1 = 1 << 10
	DefaultK2 = 100
)

var (
	DefaultDataDir = filepath.Join(smutil.GetUserHomeDirectory(), "post", DefaultDataDirName)
)

type Config struct {
	DataDir          string `mapstructure:"post-datadir"`
	NumFiles         uint   `mapstructure:"post-numfiles"` // Remove? should be only in initoptions.
	ComputeBatchSize uint   `mapstructure:"post-compute-batch-size"`

	// Protocol params.
	NumUnits      uint // REMOVE!!
	BitsPerLabel  uint
	LabelsPerUnit uint
	MinNumUnits   uint
	MaxNumUnits   uint

	K1 uint `mapstructure:"post-k1"`
	K2 uint `mapstructure:"post-k2"`
}

// TODO(moshababo): add tests for all cases
func (cfg *Config) Validate() error {
	//if cfg.NumLabels > MaxNumLabels {
	//	return fmt.Errorf("invalid `NumLabels`; expected: <= %d, given: %d", MaxNumLabels, cfg.NumLabels)
	//}
	//
	//if !shared.IsPowerOfTwo(uint64(cfg.NumFiles)) {
	//	return fmt.Errorf("invalid `NumFiles`; expected: a power of 2, given: %d", cfg.NumFiles)
	//}
	//
	//if cfg.NumFiles > MaxNumFiles {
	//	return fmt.Errorf("invalid `NumFiles`; expected: <= %d, given: %d", MaxNumFiles, cfg.NumFiles)
	//}
	//
	//if cfg.NumFiles < MinNumFiles {
	//	return fmt.Errorf("invalid `NumFiles`; expected: >= %d, given: %d", MinNumFiles, cfg.NumFiles)
	//}
	//
	//if cfg.BitsPerLabel > MaxBitsPerLabel {
	//	return fmt.Errorf("invalid `BitsPerLabel`; expected: <= %d, given: %d", MaxBitsPerLabel, cfg.LabelSize)
	//}
	//
	//if cfg.BitsPerLabel < MinBitsPerLabel {
	//	return fmt.Errorf("invalid `BitsPerLabel`; expected: >= %d, given: %d", MinBitsPerLabel, cfg.LabelSize)
	//}
	//
	//if cfg.NumLabels%uint64(cfg.NumFiles) != 0 {
	//	return fmt.Errorf("invalid `NumLabels`; expected: evenly divisible by `NumFiles` (%v), given: %d", cfg.NumFiles, cfg.NumLabels)
	//}
	//
	//fileNumLabels := cfg.NumLabels / uint64(cfg.NumFiles)
	//if fileNumLabels < MinFileNumLabels {
	//	return fmt.Errorf("invalid file number of labels; expected: >= %d, given: %d", MinFileNumLabels, fileNumLabels)
	//}
	//
	//// Divisibility by 8 will guarantee that writing to disk, and in addition file truncating,
	//// is byte-granular, regardless of LabelSize.
	//if cfg.ComputeBatchSize%8 != 0 {
	//	return fmt.Errorf("invalid `ComputeBatchSize`; expected: evenly divisible by 8, given: %d", cfg.ComputeBatchSize)
	//}
	//lastComputeBatchSize := fileNumLabels % uint64(cfg.ComputeBatchSize)
	//if lastComputeBatchSize%8 != 0 {
	//	return fmt.Errorf("invalid last batch size; expected: evenly divisible by 8, given: %d", lastComputeBatchSize)
	//}
	//if fileNumLabels%8 != 0 {
	//	return fmt.Errorf("invalid file number of labels; expected: evenly divisible by 8, given: %d", fileNumLabels)
	//}
	//
	//if res := shared.Uint64MulOverflow(cfg.NumLabels, uint64(cfg.K1)); res {
	//	return fmt.Errorf("uint64 overflow: `NumLabels` (%v) multipled by K1 (%v) exceeds the range allowed by uint64",
	//		cfg.NumLabels, cfg.K1)
	//}

	return nil
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:          DefaultDataDir,
		NumFiles:         DefaultNumFiles,
		ComputeBatchSize: DefaultComputeBatchSize,

		BitsPerLabel:  DefaultBitsPerLabel,
		LabelsPerUnit: DefaultLabelsPerUnit,
		MaxNumUnits:   MaxNumUnits,
		MinNumUnits:   MinNumUnits,

		K1: DefaultK1,
		K2: DefaultK2,
	}
}
