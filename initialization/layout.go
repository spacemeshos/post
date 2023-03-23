package initialization

import "github.com/spacemeshos/post/config"

type filesLayout struct {
	NumFiles          uint
	FileNumLabels     uint64
	LastFileNumLabels uint64
}

func deriveFilesLayout(cfg config.Config, opts config.InitOpts) filesLayout {
	maxFileSizeBits := opts.MaxFileSize * 8
	maxFileNumLabels := maxFileSizeBits / uint64(config.BitsPerLabel)
	numLabels := cfg.LabelsPerUnit * uint64(opts.NumUnits)
	numFiles := numLabels / maxFileNumLabels

	lastFileNumLabels := maxFileNumLabels
	remainder := numLabels % maxFileNumLabels
	if remainder > 0 {
		numFiles++
		lastFileNumLabels = remainder
	}

	return filesLayout{
		NumFiles:          uint(numFiles),
		FileNumLabels:     maxFileNumLabels,
		LastFileNumLabels: lastFileNumLabels,
	}
}
