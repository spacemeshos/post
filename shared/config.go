package shared

import (
	"github.com/spacemeshos/smutil"
	"path/filepath"
)

const (
	DefaultDataDirName                             = "data"
	DefaultLabelsLogRate                           = 5000000
	DefaultEnableParallelism                       = true
	DefaultSpacePerUnit                            = 1 << 20 // 1MB. Temporary value.
	DefaultFileSize                                = 1 << 20 // 1MB. Temporary value.
	DefaultDifficulty                              = MinDifficulty
	DefaultNumOfProvenLabels                       = 100 // The recommended setting to ensure proof safety.
	DefaultLowestLayerToCacheDuringProofGeneration = 11
)

var (
	defaultDataDir = filepath.Join(smutil.GetUserHomeDirectory(), "post", DefaultDataDirName)
)

type Config struct {
	DataDir           string `mapstructure:"datadir"`
	LabelsLogRate     uint64 `mapstructure:"lograte"`
	EnableParallelism bool   `mapstructure:"parallel"`

	// Protocol params.
	SpacePerUnit                            uint64 `mapstructure:"space"`
	FileSize                                uint64 `mapstructure:"filesize"`
	Difficulty                              uint   `mapstructure:"difficulty"`
	NumOfProvenLabels                       uint   `mapstructure:"t"`
	LowestLayerToCacheDuringProofGeneration uint   `mapstructure:"cachelayer"`
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:                                 defaultDataDir,
		LabelsLogRate:                           DefaultLabelsLogRate,
		EnableParallelism:                       DefaultEnableParallelism,
		SpacePerUnit:                            DefaultSpacePerUnit,
		FileSize:                                DefaultFileSize,
		Difficulty:                              DefaultDifficulty,
		NumOfProvenLabels:                       DefaultNumOfProvenLabels,
		LowestLayerToCacheDuringProofGeneration: DefaultLowestLayerToCacheDuringProofGeneration,
	}
}
