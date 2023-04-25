package initialization

import "github.com/spacemeshos/post/internal/gpu"

// Benchmark returns the hashes per second the selected compute provider achieves on the current machine.
func Benchmark(p ComputeProvider) (int, error) {
	endPosition := uint64(1 << 13)
	if p.Model == gpu.CPUProviderName {
		endPosition = uint64(1 << 10)
	}

	res, err := gpu.ScryptPositions(
		gpu.WithComputeProviderID(p.ID),
		gpu.WithCommitment(make([]byte, 32)),
		gpu.WithSalt(make([]byte, 32)),
		gpu.WithStartAndEndPosition(1, endPosition),
		gpu.WithBitsPerLabel(8),
		gpu.WithScryptParams(8192, 1, 1),
	)
	if err != nil {
		return 0, err
	}

	return res.HashesPerSec, nil
}
