package shared

import (
	"github.com/spacemeshos/post/config"
	"os"
)

var (
	MaxSpace       = uint64(config.MaxSpace)
	LabelGroupSize = uint64(config.LabelGroupSize)
	MinDifficulty  = Difficulty(config.MinDifficulty)
	MaxDifficulty  = Difficulty(config.MaxDifficulty)
	MaxNumFiles    = config.MaxNumFiles

	// OwnerReadWriteExec is a standard owner read / write / exec file permission.
	OwnerReadWriteExec = os.FileMode(0700)

	// OwnerReadWrite is a standard owner read / write file permission.
	OwnerReadWrite = os.FileMode(0600)
)

func ValidateConfig(cfg *config.Config) error {
	if err := ValidateSpace(cfg.SpacePerUnit); err != nil {
		return err
	}

	if err := ValidateNumFiles(cfg.SpacePerUnit, uint64(cfg.NumFiles)); err != nil {
		return err
	}

	difficulty := Difficulty(cfg.Difficulty)
	if err := difficulty.Validate(); err != nil {
		return err
	}

	return nil
}
