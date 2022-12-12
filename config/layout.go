package config

type FilesLayout struct {
	NumFiles          uint
	FileNumLabels     uint64
	LastFileNumLabels uint64
}

func DeriveFilesLayout(cfg Config, opts InitOpts) FilesLayout {
	maxFileSizeBits := opts.MaxFileSize * 8
	maxFileNumLabels := maxFileSizeBits / uint64(cfg.BitsPerLabel)
	numLabels := cfg.LabelsPerUnit * uint64(opts.NumUnits)
	numFiles := numLabels / maxFileNumLabels

	lastFileNumLabels := maxFileNumLabels
	remainder := numLabels % maxFileNumLabels
	if remainder > 0 {
		numFiles++
		lastFileNumLabels = remainder
	}

	return FilesLayout{
		NumFiles:          uint(numFiles),
		FileNumLabels:     maxFileNumLabels,
		LastFileNumLabels: lastFileNumLabels,
	}
}
