package initialization

import "sync/atomic"

type LabelInfo struct {
	Label        uint64
	LabelInBytes uint64
}

func NewLabelInfo(label, bitsPerLabel uint64) LabelInfo {
	return LabelInfo{
		Label:        label,
		LabelInBytes: label * bitsPerLabel / 8,
	}
}

type InitializerStat struct {
	TotalFilesToWrite uint64    // total number of files to write
	CompletedFiles    uint64    // number of files that were actually written
	LabelsPerFile     LabelInfo // number of labels in each file
	WrittenAtLastFile LabelInfo // number of labels written to the last file
}

func (init *Initializer) SessionStat() InitializerStat {
	labelsPerFile := init.getFileNumLabels()
	writtenAtLast := atomic.LoadUint64(&init.numLabelsWritten)
	return InitializerStat{
		TotalFilesToWrite: uint64(init.opts.NumFiles),
		CompletedFiles:    atomic.LoadUint64(&init.processedFiles),
		WrittenAtLastFile: NewLabelInfo(writtenAtLast, uint64(init.cfg.BitsPerLabel)),
		LabelsPerFile:     NewLabelInfo(labelsPerFile, uint64(init.cfg.BitsPerLabel)),
	}
}
