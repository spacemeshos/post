package gpu

import (
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/shared"
)

var (
	commitment = make([]byte, 32)
	salt       = make([]byte, 32)
)

func TestCPUProviderExists(t *testing.T) {
	r := require.New(t)

	id := CPUProviderID()
	r.NotNil(id, "CPU provider not found")

	for _, p := range Providers() {
		if p.ID == id {
			r.Equal("CPU", p.Model)
			r.Equal(ComputeAPIClassCPU, p.ComputeAPI)
			return
		}
	}

	r.Fail("CPU provider doesn't exist")
}

// TestScryptPositions is an output correctness sanity test. It doesn't cover many cases.
func TestScryptPositions(t *testing.T) {
	r := require.New(t)

	providers := Providers()
	var prevOutput []byte
	for _, p := range providers {
		hashLenBits := uint32(4)
		res, err := ScryptPositions(
			WithComputeProviderID(p.ID),
			WithCommitment(commitment),
			WithSalt(salt),
			WithStartAndEndPosition(1, 1<<8),
			WithBitsPerLabel(hashLenBits),
			WithScryptParams(32, 1, 1),
		)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.False(res.Stopped)

		t.Logf("provider: %+v, res: %+v\n", p, res)

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
	if testing.Short() {
		t.Skip("long test")
	}

	providers := Providers()
	for hashLenBits := uint32(1); hashLenBits <= 256; hashLenBits++ {
		var prevOutput []byte
		for _, p := range providers {
			res, err := ScryptPositions(
				WithComputeProviderID(p.ID),
				WithCommitment(commitment),
				WithSalt(salt),
				WithStartAndEndPosition(1, 1<<12),
				WithBitsPerLabel(hashLenBits),
				WithScryptParams(32, 1, 1),
			)
			r.NoError(err)
			r.NotNil(res)
			r.NotNil(res.Output)
			r.False(res.Stopped)

			t.Logf("provider: %+v, len: %v, hs: %v\n", p, hashLenBits, res.HashesPerSec)

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
	res, err := ScryptPositions(
		WithComputeProviderID(invalidProviderId),
		WithCommitment(commitment),
		WithSalt(salt),
		WithStartAndEndPosition(1, 1),
		WithBitsPerLabel(8),
		WithScryptParams(32, 1, 1),
	)
	req.EqualError(err, "gpu-post error: invalid provider")
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
			res, err := ScryptPositions(
				WithComputeProviderID(providerId),
				WithCommitment(commitment),
				WithSalt(salt),
				WithStartAndEndPosition(startPosition, endPosition),
				WithBitsPerLabel(hashLenBits),
				WithScryptParams(512, 1, 1),
			)
			r.NoError(err)
			r.NotNil(res)
			r.NotNil(res.Output)
			r.True(res.Stopped)

			// `res.Output` size is expected be smaller than expected due to `Stop` call.
			expectedOutputSize := shared.DataSize(endPosition-startPosition+1, uint(hashLenBits))
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
		startPosition := uint64(1)
		endPosition := uint64(1 << 17)
		hashLenBits := uint32(8)
		res, err := ScryptPositions(
			WithComputeProviderID(p.ID),
			WithCommitment(commitment),
			WithSalt(salt),
			WithStartAndEndPosition(startPosition, endPosition),
			WithBitsPerLabel(hashLenBits),
			WithScryptParams(2, 1, 1),
		)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.False(res.Stopped)

		expectedOutputSize := shared.DataSize(endPosition-startPosition+1, uint(hashLenBits))
		r.Equal(int(expectedOutputSize), len(res.Output))
	}
}

func TestStop_SameThread(t *testing.T) {
	r := require.New(t)

	providers := Providers()
	for _, p := range providers {
		go func() {
			time.Sleep(100 * time.Millisecond)
			res := cStop(10000)
			r.Equal(StopResultOk, res)
		}()
		startPosition := uint64(1)
		endPosition := uint64(1 << 18)
		hashLenBits := uint32(8)
		res, err := ScryptPositions(
			WithComputeProviderID(p.ID),
			WithCommitment(commitment),
			WithSalt(salt),
			WithStartAndEndPosition(startPosition, endPosition),
			WithBitsPerLabel(hashLenBits),
			WithScryptParams(512, 1, 1),
		)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.True(res.Stopped, "provider %v", p)

		// `res.Output` size is expected be smaller than expected due to `Stop` call.
		outputSize := shared.DataSize(endPosition-startPosition+1, uint(hashLenBits))
		r.True(len(res.Output) > 0)
		r.True(len(res.Output) < int(outputSize))

		// Testing that a call to `ScryptPositions` after `Stop` is working properly.
		startPosition = uint64(1)
		endPosition = uint64(1 << 17)
		res, err = ScryptPositions(
			WithComputeProviderID(p.ID),
			WithCommitment(commitment),
			WithSalt(salt),
			WithStartAndEndPosition(startPosition, endPosition),
			WithBitsPerLabel(hashLenBits),
			WithScryptParams(32, 1, 1),
		)
		r.NoError(err)
		r.NotNil(res)
		r.NotNil(res.Output)
		r.False(res.Stopped)

		expectedOutputSize := shared.DataSize(endPosition-startPosition+1, uint(hashLenBits))
		r.Equal(int(expectedOutputSize), len(res.Output))
	}
}

func TestScryptPositions_PartialByte(t *testing.T) {
	req := require.New(t)

	providers := Providers()
	var prevOutput []byte
	for _, p := range providers {
		hashLenBits := uint32(4)
		res, err := ScryptPositions(
			WithComputeProviderID(p.ID),
			WithCommitment(commitment),
			WithSalt(salt),
			WithStartAndEndPosition(1, 9),
			WithBitsPerLabel(hashLenBits),
			WithScryptParams(32, 1, 1),
		)
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

func Test_ScryptPositions_Pow(t *testing.T) {
	commitment, err := hex.DecodeString("e26b543725490682675f6f84ea7689601adeaf14caa7024ec1140c82754ca339")
	require.NoError(t, err)

	salt, err := hex.DecodeString("165310acce39719148915c356f25c5cb78e82203222cccdf3c15a9c3684e08cb")
	require.NoError(t, err)

	d, err := hex.DecodeString("00003fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	require.NoError(t, err)

	nonce := uint64(0x10f1d)

	for _, p := range Providers() {
		t.Run(fmt.Sprintf("Only PoW, Provider %s", p.Model), func(t *testing.T) {
			res, err := ScryptPositions(
				WithComputeProviderID(p.ID),
				WithCommitment(commitment),
				WithSalt(salt),
				WithStartAndEndPosition(0, 256*1024),
				WithBitsPerLabel(8*16),
				WithComputePow(d),
				WithComputeLeaves(false),
				WithScryptParams(128, 1, 1),
			)

			require.NoError(t, err)
			assert.NotNil(t, res.IdxSolution)
			assert.Equal(t, nonce, *res.IdxSolution)
		})

		t.Run(fmt.Sprintf("PoW + Leafs, Provider %s", p.Model), func(t *testing.T) {
			res, err := ScryptPositions(
				WithComputeProviderID(p.ID),
				WithCommitment(commitment),
				WithSalt(salt),
				WithStartAndEndPosition(0, 256*1024),
				WithBitsPerLabel(8*16),
				WithComputePow(d),
				WithComputeLeaves(true),
				WithScryptParams(128, 1, 1),
			)

			require.NoError(t, err)
			assert.NotNil(t, res.Output)
			assert.NotNil(t, res.IdxSolution)
			assert.Equal(t, nonce, *res.IdxSolution)
		})
	}
}
