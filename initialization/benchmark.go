package initialization

import (
	"time"

	"github.com/spacemeshos/post/internal/postrs"
)

// Benchmark returns the hashes per second the selected compute provider achieves on the current machine.
func Benchmark(p Provider) (int, error) {
	endPosition := uint64(1 << 14)
	if p.DeviceType == postrs.ClassCPU {
		endPosition = uint64(1 << 12)
	}

	start := time.Now()
	_, err := postrs.ScryptPositions(
		postrs.WithProviderID(p.ID),
		postrs.WithCommitment(make([]byte, 32)),
		postrs.WithStartAndEndPosition(1, endPosition),
		postrs.WithScryptN(8192),
	)
	elapsed := time.Since(start)
	if err != nil {
		return 0, err
	}

	hashesPerSecond := float64(endPosition) / elapsed.Seconds()
	return int(hashesPerSecond), nil
}
