package gpu

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spacemeshos/post/config"
	"time"
)

type ComputeProvider struct {
	ID         uint
	Model      string
	ComputeAPI ComputeAPIClass
}

func Providers() []ComputeProvider {
	return cGetProviders()
}

func Benchmark(p ComputeProvider) (int, error) {
	id := make([]byte, 32)
	salt := make([]byte, 32)
	hashLenBits := uint32(8)
	startPosition := uint64(1)
	endPosition := uint64(1 << 17)
	if p.Model == "CPU" {
		endPosition = uint64(1 << 14)
	}

	res, err := ScryptPositions(p.ID, id, salt, startPosition, endPosition, hashLenBits)
	if err != nil {
		return 0, err
	}

	return res.HashesPerSec, nil
}

type ScryptPositionsResult struct {
	Output       []byte
	IdxSolution  uint64
	HashesPerSec int
	Stopped      bool
}

func ScryptPositions(providerId uint, id, salt []byte, startPosition, endPosition uint64, bitsPerLabel uint32) (*ScryptPositionsResult, error) {
	if len(id) != 32 {
		return nil, fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(id))
	}

	if len(salt) != 32 {
		return nil, fmt.Errorf("invalid `salt` length; expected: 32, given: %v", len(salt))
	}

	if bitsPerLabel < config.MinBitsPerLabel || bitsPerLabel > config.MaxBitsPerLabel {
		return nil, fmt.Errorf("invalid `bitsPerLabel`; expected: %d-%d, given: %v",
			config.MinBitsPerLabel, config.MaxBitsPerLabel, bitsPerLabel)
	}

	// Wait for the stop flag clearance for avoiding a race condition which can
	// occur if ScryptPositions is called immediately after a prior Stop call.
	var i int
	for {
		i++
		cleared := cStopCleared()
		if cleared {
			break
		}
		if i == 20 {
			return nil, errors.New("stop flag clearance timeout")
		}
		time.Sleep(100 * time.Millisecond)
	}

	const n, r, p = 512, 1, 1
	const options = 1 // COMPUTE_LEAFS on, COMPUTE_POW off.

	output, idxSolution, hashesPerSec, retVal := cScryptPositions(providerId, id, salt, startPosition, endPosition, bitsPerLabel, options, n, r, p)

	switch retVal {
	case 1:
		panic("pow solution found") // TODO: handle
	case 0:
		return &ScryptPositionsResult{output, idxSolution, hashesPerSec, false}, nil
	case -1:
		return nil, fmt.Errorf("gpu-post error")
	case -2:
		return nil, fmt.Errorf("gpu-post error: timeout")
	case -3:
		return nil, fmt.Errorf("gpu-post error: already stopped")
	case -4:
		return &ScryptPositionsResult{output, idxSolution, hashesPerSec, true}, nil
	case -5:
		return nil, fmt.Errorf("gpu-post error: no compute options")
	case -6:
		return nil, fmt.Errorf("gpu-post error: invalid param")
	default:
		panic(fmt.Sprintf("unreachable reVal %d",retVal))
	}
}

func Stop() StopResult {
	return cStop(20000)
}
