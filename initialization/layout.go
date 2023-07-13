package initialization

import (
	"fmt"
	"math"

	"github.com/spacemeshos/post/config"
)

type filesLayout struct {
	FirstFileIdx      int
	NumFiles          uint
	FileNumLabels     uint64
	LastFileNumLabels uint64
}

func deriveFilesLayout(cfg config.Config, opts config.InitOpts) (filesLayout, error) {
	maxFileNumLabels := opts.MaxFileNumLabels()
	totalLabels := uint64(opts.NumUnits) * uint64(cfg.LabelsPerUnit)

	firstFileIdx := opts.FromFileIdx
	lastFileIdx := TotalFiles(cfg, opts) - 1

	if opts.ToFileIdx != nil {
		if *opts.ToFileIdx < 0 {
			return filesLayout{}, fmt.Errorf("invalid range: opts.ToFileIdx (%v) must be greater then 0", *opts.ToFileIdx)
		}
		if *opts.ToFileIdx > lastFileIdx {
			return filesLayout{}, fmt.Errorf("invalid range: opts.ToFileIdx (%v) cannot be greater then %v", *opts.ToFileIdx, lastFileIdx)
		}
		lastFileIdx = *opts.ToFileIdx
	}

	lastFileNumLabels := maxFileNumLabels
	labelsLeft := totalLabels - firstLabelInFile(lastFileIdx, opts)
	if labelsLeft < maxFileNumLabels {
		lastFileNumLabels = labelsLeft
	}

	if firstFileIdx > lastFileIdx {
		return filesLayout{}, fmt.Errorf("invalid range: last file index (%v) must be greater then first (%v)", lastFileIdx, firstFileIdx)
	}

	numFiles := lastFileIdx - firstFileIdx + 1

	return filesLayout{
		FirstFileIdx:      firstFileIdx,
		NumFiles:          uint(numFiles),
		FileNumLabels:     maxFileNumLabels,
		LastFileNumLabels: lastFileNumLabels,
	}, nil
}

func firstLabelInFile(fileIdx int, opts config.InitOpts) uint64 {
	return uint64(fileIdx) * opts.MaxFileNumLabels()
}

func TotalFiles(cfg config.Config, opts config.InitOpts) int {
	totalLabels := uint64(opts.NumUnits) * uint64(cfg.LabelsPerUnit)
	return int(math.Ceil(float64(totalLabels) / float64(opts.MaxFileNumLabels())))
}
