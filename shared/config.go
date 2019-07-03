package shared

import (
	"github.com/spacemeshos/smutil"
	"path/filepath"
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
