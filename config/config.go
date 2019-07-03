package config

import (
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/smutil"
	"math"
	"path/filepath"
)

const (
	LabelGroupSize = merkle.NodeSize

	// In bytes. 1 peta-byte of storage.
	// This would protect against number of labels uint64 overflow as well,
	// since the number of labels per byte can be 8 at most (3 extra bit shifts).
	MaxSpace = 1 << 40 // 1099511627777

	MaxNumFiles = math.MaxUint8 // 255

	MinDifficulty = 5 // 1 byte per label
	MaxDifficulty = 8 // 1 bit per label
)

const (
	DefaultDataDirName                             = "data"
	DefaultLabelsLogRate                           = 5000000
	DefaultMaxFilesParallelism                     = 1
	DefaultMaxInFileParallelism                    = 6
	DefaultMaxReadParallelism                      = 6
	DefaultSpacePerUnit                            = 1 << 20 // 1MB. Temporary value.
	DefaultFileSize                                = 1 << 20 // 1MB. Temporary value.
	DefaultDifficulty                              = MinDifficulty
	DefaultNumProvenLabels                         = 100 // The recommended setting to ensure proof safety.
	DefaultLowestLayerToCacheDuringProofGeneration = 11
)

var (
	DefaultDataDir = filepath.Join(smutil.GetUserHomeDirectory(), "post", DefaultDataDirName)
)

type Config struct {
	DataDir                   string `mapstructure:"datadir"`
	LabelsLogRate             uint64 `mapstructure:"lograte"`
	MaxWriteFilesParallelism  uint   `mapstructure:"parallel-files"`
	MaxWriteInFileParallelism uint   `mapstructure:"parallel-infile"`
	MaxReadFilesParallelism   uint   `mapstructure:"parallel-read"`

	// Protocol params.
	SpacePerUnit                            uint64 `mapstructure:"space"`
	FileSize                                uint64 `mapstructure:"filesize"`
	Difficulty                              uint   `mapstructure:"difficulty"`
	NumProvenLabels                         uint   `mapstructure:"labels"`
	LowestLayerToCacheDuringProofGeneration uint   `mapstructure:"cachelayer"`
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:                                 DefaultDataDir,
		LabelsLogRate:                           DefaultLabelsLogRate,
		MaxWriteFilesParallelism:                DefaultMaxFilesParallelism,
		MaxWriteInFileParallelism:               DefaultMaxInFileParallelism,
		MaxReadFilesParallelism:                 DefaultMaxReadParallelism,
		SpacePerUnit:                            DefaultSpacePerUnit,
		FileSize:                                DefaultFileSize,
		Difficulty:                              DefaultDifficulty,
		NumProvenLabels:                         DefaultNumProvenLabels,
		LowestLayerToCacheDuringProofGeneration: DefaultLowestLayerToCacheDuringProofGeneration,
	}
}
