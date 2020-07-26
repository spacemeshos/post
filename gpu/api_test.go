package gpu

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

var (
	id      = []byte("id")
	salt    = []byte("salt")
	options = uint32(0)
)

func TestScryptPositions(t *testing.T) {
	req := require.New(t)

	providers := GetProviders()
	for _, p := range providers {
		providerId := uint(p.Id)
		startPosition := uint64(1)
		endPosition := uint64(1 << 11)
		hashLenBits := uint8(4)
		output, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options)
		req.NoError(err)
		req.NotNil(output)
		req.Equal(1<<10, len(output))
	}
}

func TestScryptPositions_InvalidProviderId(t *testing.T) {
	req := require.New(t)

	invalidProviderId := uint(1 << 10)
	output, err := ScryptPositions(invalidProviderId, id, salt, 1, 1, 8, options)
	req.EqualError(err, fmt.Sprintf("invalid provider id: %d", invalidProviderId))
	req.Nil(output)
}

func TestStop(t *testing.T) {
	req := require.New(t)

	providers := GetProviders()
	for _, p := range providers {
		// CPU-mode stop currently not working.
		if p.ComputeAPI == ComputeAPIClassCPU {
			continue
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Second)
			res := cStop(2000)
			req.Equal(StopResultOk, res)
		}()
		go func() {
			defer wg.Done()
			providerId := uint(p.Id)
			startPosition := uint64(1)
			endPosition := uint64(1 << 20)
			hashLenBits := uint8(8)
			output, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options)
			req.NoError(err)
			req.NotNil(output)
			req.Equal(1<<20, len(output))
		}()
		c := make(chan struct{})
		go func() {
			defer close(c)
			wg.Wait()
		}()
		select {
		case <-c:
		case <-time.After(3 * time.Second):
			req.Fail(fmt.Sprintf("stop timed out; provider: %+v", p))
		}
	}
}

//func TestBenchmark(t *testing.T) {
//	req := require.New(t)
//
//	results := make(map[string]uint64)
//	providers := GetProviders()
//	for _, p := range providers {
//		providerId := uint(p.Id)
//		hps, err := Benchmark(providerId)
//		req.NoError(err)
//		results[p.Model] = hps
//	}
//
//	fmt.Printf("%v\n", results)
//}
