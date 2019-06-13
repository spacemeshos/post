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
	DefaultSpacePerUnit                            = 1 << 20 // 1MB. Temporary value.
	DefaultFileSize                                = 1 << 20 // 1MB. Temporary value.
	DefaultDifficulty                              = MinDifficulty
	DefaultNumOfProvenLabels                       = 100 // The recommended setting to ensure proof safety.
	DefaultLowestLayerToCacheDuringProofGeneration = 11
)

var (
	DefaultDataDir = filepath.Join(smutil.GetUserHomeDirectory(), "post", DefaultDataDirName)
)

type Config struct {
	DataDir              string `mapstructure:"datadir"`
	LabelsLogRate        uint64 `mapstructure:"lograte"`
	MaxFilesParallelism  uint   `mapstructure:"parallel-files"`
	MaxInFileParallelism uint   `mapstructure:"parallel-infile"`

	// Protocol params.
	SpacePerUnit                            uint64 `mapstructure:"space"`
	FileSize                                uint64 `mapstructure:"filesize"`
	Difficulty                              uint   `mapstructure:"difficulty"`
	NumOfProvenLabels                       uint   `mapstructure:"labels"`
	LowestLayerToCacheDuringProofGeneration uint   `mapstructure:"cachelayer"`
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:                                 DefaultDataDir,
		LabelsLogRate:                           DefaultLabelsLogRate,
		MaxFilesParallelism:                     DefaultMaxFilesParallelism,
		MaxInFileParallelism:                    DefaultMaxInFileParallelism,
		SpacePerUnit:                            DefaultSpacePerUnit,
		FileSize:                                DefaultFileSize,
		Difficulty:                              DefaultDifficulty,
		NumOfProvenLabels:                       DefaultNumOfProvenLabels,
		LowestLayerToCacheDuringProofGeneration: DefaultLowestLayerToCacheDuringProofGeneration,
	}
}
