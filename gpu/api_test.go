package gpu

import (
	"flag"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

var (
	id   = make([]byte, 32)
	salt = make([]byte, 32)

	debug = flag.Bool("debug", false, "")
	long  = flag.Bool("long", false, "")
)

// TestScryptPositions is an output correctness sanity test. It doesn't cover many cases.
func TestScryptPositions(t *testing.T) {
	r := require.New(t)

	providers := Providers()
	var prevOutput []byte
	for _, p := range providers {
		providerId := uint(p.ID)
		startPosition := uint64(1)
		endPosition := uint64(1 << 8)
		hashLenBits := uint32(4)
		res, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.False(res.Stopped)

		if *debug {
			fmt.Printf("provider: %+v, res: %+v\n", p, res)
		}

		// Assert that output content is equal between different providers.
		if prevOutput == nil {
			prevOutput = res.Output
		} else {
			r.Equal(prevOutput, res.Output, fmt.Sprintf("not equal: provider: %+v, hashLenBits: %v", p, hashLenBits))
		}
	}
}

// TestScryptPositions_HashLenBits tests output correctness for the entire value range of HashLenBits for a specific batch size.
func TestScryptPositions_HashLenBits(t *testing.T) {
	r := require.New(t)
	if !*long {
		t.Skip("long test")
	}

	providers := Providers()
	for hashLenBits := uint32(1); hashLenBits <= 256; hashLenBits++ {
		var prevOutput []byte
		for _, p := range providers {
			providerId := uint(p.ID)
			startPosition := uint64(1)
			endPosition := uint64(1 << 12)
			res, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits)
			r.NoError(err)
			r.NotNil(res)
			r.NotNil(res.Output)
			r.False(res.Stopped)

			if *debug {
				fmt.Printf("provider: %+v, len: %v, hs: %v\n", p, hashLenBits, res.HashesPerSec)
			}

			// Assert that output content is equal between different providers.
			if prevOutput == nil {
				prevOutput = res.Output
			} else {
				r.Equal(prevOutput, res.Output, fmt.Sprintf("not equal: provider: %+v, hashLenBits: %v", p, hashLenBits))
			}
		}
	}
}

func TestScryptPositions_InvalidProviderId(t *testing.T) {
	req := require.New(t)

	invalidProviderId := uint(1 << 10)
	res, err := ScryptPositions(invalidProviderId, id, salt, 1, 1, 8)
	req.EqualError(err, fmt.Sprintf("gpu-post error: invalid param"))
	req.Nil(res)
}

func TestStop(t *testing.T) {
	r := require.New(t)

	providers := Providers()
	for _, p := range providers {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Second)
			res := cStop(10000)
			r.Equal(StopResultOk, res)
		}()
		go func() {
			defer wg.Done()
			providerId := uint(p.ID)
			startPosition := uint64(1)
			endPosition := uint64(1 << 18)
			hashLenBits := uint32(8)
			res, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits)
			r.NoError(err)
			r.NotNil(res)
			r.NotNil(res.Output)
			r.True(res.Stopped)

			// `res.Output` size is expected be smaller than expected due to `Stop` call.
			expectedOutputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
			r.True(len(res.Output) > 0)
			r.True(len(res.Output) < int(expectedOutputSize))
		}()
		c := make(chan struct{})
		go func() {
			defer close(c)
			wg.Wait()
		}()
		select {
		case <-c:
		case <-time.After(10 * time.Second):
			r.Fail(fmt.Sprintf("stop timed out; provider: %+v", p))
		}

		// Testing that a call to `ScryptPositions` after `Stop` is working properly.
		providerID := uint(p.ID)
		startPosition := uint64(1)
		endPosition := uint64(1 << 17)
		hashLenBits := uint32(8)
		res, err := ScryptPositions(providerID, id, salt, startPosition, endPosition, hashLenBits)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.False(res.Stopped)

		expectedOutputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
		r.Equal(int(expectedOutputSize), len(res.Output))
	}
}

func TestStop_SameThread(t *testing.T) {
	r := require.New(t)

	providers := Providers()
	for _, p := range providers {
		go func() {
			time.Sleep(1 * time.Second)
			res := cStop(10000)
			r.Equal(StopResultOk, res)
		}()
		providerID := uint(p.ID)
		startPosition := uint64(1)
		endPosition := uint64(1 << 18)
		hashLenBits := uint32(8)
		res, err := ScryptPositions(providerID, id, salt, startPosition, endPosition, hashLenBits)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.True(res.Stopped)

		// `res.Output` size is expected be smaller than expected due to `Stop` call.
		outputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
		r.True(len(res.Output) > 0)
		r.True(len(res.Output) < int(outputSize))

		// Testing that a call to `ScryptPositions` after `Stop` is working properly.
		startPosition = uint64(1)
		endPosition = uint64(1 << 17)
		res, err = ScryptPositions(providerID, id, salt, startPosition, endPosition, hashLenBits)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.False(res.Stopped)

		expectedOutputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
		r.Equal(int(expectedOutputSize), len(res.Output))
	}
}

func TestScryptPositions_PartialByte(t *testing.T) {
	req := require.New(t)

	providers := Providers()
	var prevOutput []byte
	for _, p := range providers {
		providerId := uint(p.ID)
		startPosition := uint64(1)
		endPosition := uint64(9)
		hashLenBits := uint32(4)
		res, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits)
		req.NoError(err)
		req.NotNil(res)
		req.NotNil(res.Output)

		// Assert that output content is equal between different providers.
		if prevOutput == nil {
			prevOutput = res.Output
		} else {
			req.Equal(prevOutput, res.Output, fmt.Sprintf("not equal: provider: %v, hashLenBits: %v", p.Model, hashLenBits))
		}
	}
}

func TestBenchmark(t *testing.T) {
	req := require.New(t)

	for _, p := range Providers() {
		b, err := Benchmark(p)
		req.NoError(err)
		req.True(b > 0)
	}
}
