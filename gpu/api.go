package gpu

import (
	"fmt"
	"github.com/pkg/errors"
	"time"
)

type ComputeProvider struct {
	ID         uint
	Model      string
	ComputeAPI ComputeAPIClass
}

func (p *ComputeProvider) Benchmark() (int, error) {
	id := make([]byte, 32)
	salt := make([]byte, 32)
	options := uint32(0)
	hashLenBits := uint8(8)
	startPosition := uint64(1)
	endPosition := uint64(1 << 20)
	if p.Model == "CPU" {
		endPosition = uint64(1 << 15)
	}

	_, hashesPerSec, err := ScryptPositions(p.ID, id, salt, startPosition, endPosition, hashLenBits, options)
	if err != nil {
		return 0, err
	}

	return hashesPerSec, nil
}

func Providers() []ComputeProvider {
	return cGetProviders()
}

func ScryptPositions(providerId uint, id, salt []byte, startPosition, endPosition uint64, hashLenBits uint8, options uint32) ([]byte, int, error) {
	if hashLenBits < 1 || hashLenBits > 8 {
		return nil, 0, fmt.Errorf("invalid `hashLenBits` value; expected: 1-8, given: %v", hashLenBits)
	}
	if len(id) != 32 {
		return nil, 0, fmt.Errorf("invalid `id` length; expected: 32, given: %v", len(id))
	}
	if len(salt) != 32 {
		return nil, 0, fmt.Errorf("invalid `salt` length; expected: 32, given: %v", len(salt))
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
			return nil, 0, errors.New("stop flag clearance timeout")
		}
		time.Sleep(100 * time.Millisecond)
	}

	const n, r, p = 512, 1, 1
	output, hashesPerSec, retVal := cScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options, n, r, p)
	switch retVal {
	case 0:
		return output, hashesPerSec, nil
	case -1:
		return nil, 0, fmt.Errorf("invalid provider id: %v", providerId)
	default:
		panic("unreachable")
	}
}

func Stop() StopResult {
	return cStop(20000)
}
