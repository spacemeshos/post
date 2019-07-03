package shared

import "github.com/spacemeshos/post/config"

var (
	MaxSpace       = uint64(config.MaxSpace)
	LabelGroupSize = uint64(config.LabelGroupSize)
	MinDifficulty  = Difficulty(config.MinDifficulty)
	MaxDifficulty  = Difficulty(config.MaxDifficulty)
	MaxNumFiles    = config.MaxNumFiles
)
