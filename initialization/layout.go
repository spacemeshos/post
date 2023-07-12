package initialization

import (
	"fmt"

	"github.com/spacemeshos/post/config"
)

type filesLayout struct {
	From uint64
	To   uint64

	NumFiles          uint
	FileNumLabels     uint64
	LastFileNumLabels uint64
}

func deriveFilesLayout(cfg config.Config, opts config.InitOpts) (filesLayout, error) {
	maxFileSizeBits := opts.MaxFileSize * 8
	maxFileNumLabels := maxFileSizeBits / uint64(config.BitsPerLabel)
	totalLabels := uint64(opts.NumUnits) * uint64(cfg.LabelsPerUnit)

	start := opts.From
	end := totalLabels

	if opts.To != nil {
		end = *opts.To
	}

	if start >= end {
		return filesLayout{}, fmt.Errorf("invalid range: start (%v) must be less then end (%v)", start, end)
	}
	// Avoid starting in the middle of a file
	if start%maxFileNumLabels != 0 {
		return filesLayout{}, fmt.Errorf("invalid range: start (%v) must be a multiple of: %v", start, maxFileNumLabels)
	}

	numLabels := end - start
	numFiles := numLabels / maxFileNumLabels

	lastFileNumLabels := maxFileNumLabels
	remainder := numLabels % maxFileNumLabels
	if remainder > 0 {
		numFiles++
		lastFileNumLabels = remainder
	}

	return filesLayout{
		From:              start,
		To:                end,
		NumFiles:          uint(numFiles),
		FileNumLabels:     maxFileNumLabels,
		LastFileNumLabels: lastFileNumLabels,
	}, nil
}
