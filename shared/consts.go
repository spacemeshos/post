package shared

import (
	"github.com/spacemeshos/merkle-tree"
	"math"
)

const (
	LabelGroupSize = merkle.NodeSize

	// NumOfProvenLabels is the recommended setting for this argument to ensure proof safety.
	NumOfProvenLabels = 100

	LowestLayerToCacheDuringProofGeneration = 11

	// In bytes. 1 peta-byte of storage.
	// This would protect against number of labels uint64 overflow as well,
	// since the number of labels per byte can be 8 at most (3 extra bit shifts).
	MaxSpace = 1 << 40 // 1099511627777

	MaxNumOfFiles = math.MaxUint8 // 255

	MinDifficulty = 5 // 1 byte per label
	MaxDifficulty = 8 // 1 bit per label
)
