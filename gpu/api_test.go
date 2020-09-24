package gpu

import (
	"fmt"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

var (
	id      = make([]byte, 32)
	salt    = make([]byte, 32)
	options = uint32(0)
)

func TestScryptPositions(t *testing.T) {
	req := require.New(t)

	providers := Providers()
	var prevOutput []byte
	for _, p := range providers {
		providerId := uint(p.ID)
		startPosition := uint64(1)
		endPosition := uint64(1 << 11)
		hashLenBits := uint8(4)
		output, _, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options)
		req.NoError(err)
		req.NotNil(output)

		// Assert that output content is equal between different providers.
		if prevOutput == nil {
			prevOutput = output
		} else {
			req.Equal(prevOutput, output, fmt.Sprintf("not equal: provider: %v, hashLenBits: %v", p.Model, hashLenBits))
		}
	}
}

func TestScryptPositions_HashLenBits(t *testing.T) {
	req := require.New(t)

	providers := Providers()
	for hashLenBits := uint8(1); hashLenBits <= 8; hashLenBits++ {
		var prevOutput []byte
		for _, p := range providers {
			providerId := uint(p.ID)
			startPosition := uint64(1)
			endPosition := uint64(1 << 4)
			output, _, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options)
			req.NoError(err)
			req.NotNil(output)

			// Assert that output content is equal between different providers.
			if prevOutput == nil {
				prevOutput = output
			} else {
				req.Equal(prevOutput, output, fmt.Sprintf("not equal: provider: %v, hashLenBits: %v", p.Model, hashLenBits))
			}
		}
	}
}

func TestScryptPositions_InvalidProviderId(t *testing.T) {
	req := require.New(t)

	invalidProviderId := uint(1 << 10)
	output, _, err := ScryptPositions(invalidProviderId, id, salt, 1, 1, 8, options)
	req.EqualError(err, fmt.Sprintf("invalid provider id: %d", invalidProviderId))
	req.Nil(output)
}

func TestStop(t *testing.T) {
	req := require.New(t)

	providers := Providers()
	for _, p := range providers {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Second)
			res := cStop(10000)
			req.Equal(StopResultOk, res)
		}()
		go func() {
			defer wg.Done()
			providerId := uint(p.ID)
			startPosition := uint64(1)
			endPosition := uint64(1 << 18)
			hashLenBits := uint8(8)
			output, _, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options)
			req.NoError(err)
			req.NotNil(output)

			// `output` size is expected be smaller than expected due to `Stop` call.
			expectedOutputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
			req.True(len(output) > 0)
			req.True(len(output) < int(expectedOutputSize))
		}()
		c := make(chan struct{})
		go func() {
			defer close(c)
			wg.Wait()
		}()
		select {
		case <-c:
		case <-time.After(10 * time.Second):
			req.Fail(fmt.Sprintf("stop timed out; provider: %+v", p))
		}

		// Testing that a call to `ScryptPositions` after `Stop` is working properly.
		providerID := uint(p.ID)
		startPosition := uint64(1)
		endPosition := uint64(1 << 17)
		hashLenBits := uint8(8)
		output, _, err := ScryptPositions(providerID, id, salt, startPosition, endPosition, hashLenBits, options)
		req.NoError(err)
		req.NotNil(output)
		expectedOutputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
		req.Equal(int(expectedOutputSize), len(output))
	}
}

func TestStop_SameThread(t *testing.T) {
	req := require.New(t)

	providers := Providers()
	for _, p := range providers {
		go func() {
			time.Sleep(1 * time.Second)
			res := cStop(10000)
			req.Equal(StopResultOk, res)
		}()
		providerID := uint(p.ID)
		startPosition := uint64(1)
		endPosition := uint64(1 << 18)
		hashLenBits := uint8(8)
		output, _, err := ScryptPositions(providerID, id, salt, startPosition, endPosition, hashLenBits, options)
		req.NoError(err)
		req.NotNil(output)

		// `output` size is expected be smaller than expected due to `Stop` call.
		outputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
		req.True(len(output) > 0)
		req.True(len(output) < int(outputSize))

		// Testing that a call to `ScryptPositions` after `Stop` is working properly.
		startPosition = uint64(1)
		endPosition = uint64(1 << 17)
		output, _, err = ScryptPositions(providerID, id, salt, startPosition, endPosition, hashLenBits, options)
		req.NoError(err)
		req.NotNil(output)
		expectedOutputSize := shared.DataSize(uint64(endPosition-startPosition+1), uint(hashLenBits))
		req.Equal(int(expectedOutputSize), len(output))
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
		hashLenBits := uint8(4)
		output, _, err := ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options)
		req.NoError(err)
		req.NotNil(output)

		// Assert that output content is equal between different providers.
		if prevOutput == nil {
			prevOutput = output
		} else {
			req.Equal(prevOutput, output, fmt.Sprintf("not equal: provider: %v, hashLenBits: %v", p.Model, hashLenBits))
		}
	}
}

func TestBenchmark(t *testing.T) {
	req := require.New(t)

	for _, p := range Providers() {
		b, err := p.Benchmark()
		req.NoError(err)
		req.True(b > 0)
	}
}
