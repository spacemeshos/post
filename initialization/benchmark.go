package initialization

import (
	"time"

	"github.com/spacemeshos/post/internal/postrs"
)

// Benchmark returns the hashes per second the selected compute provider achieves on the current machine.
func Benchmark(p ComputeProvider) (int, error) {
	endPosition := uint64(1 << 13)
	if p.DeviceType == postrs.ClassCPU {
		endPosition = uint64(1 << 10)
	}

	start := time.Now()
	res, err := postrs.ScryptPositions(
		postrs.WithProviderID(p.ID),
		postrs.WithCommitment(make([]byte, 32)),
		postrs.WithStartAndEndPosition(1, endPosition),
		postrs.WithScryptN(8192),
		postrs.WithVRFDifficulty(make([]byte, 32)),
	)
	elapsed := time.Since(start)
	_ = res
	if err != nil {
		return 0, err
	}

	hashesPerSecond := float64(endPosition) / elapsed.Seconds()
	return int(hashesPerSecond), nil
}
