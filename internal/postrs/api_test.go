package postrs

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	commitment        = make([]byte, 32)
	defaultDifficulty = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).Bytes()
)

func TestCPUProviderExists(t *testing.T) {
	id := CPUProviderID()
	require.NotZero(t, id, "CPU provider not found")

	providers, err := OpenCLProviders()
	require.NoError(t, err)

	for _, p := range providers {
		if p.ID == id {
			require.NotEmpty(t, p.Model)
			require.Equal(t, ClassCPU, p.DeviceType)
			return
		}
	}

	require.Fail(t, "CPU provider doesn't exist")
}

// TestScryptPositions is an output correctness sanity test. It doesn't cover many cases.
func TestScryptPositions(t *testing.T) {
	providers, err := OpenCLProviders()
	require.NoError(t, err)

	vrfDifficulty := make([]byte, 32)
	copy(vrfDifficulty, defaultDifficulty)
	vrfDifficulty[0] = 0

	start := uint64(1)
	end := uint64(1 << 8)

	var prevOutput []byte
	var nonce *uint64
	for _, p := range providers {
		scrypt, err := NewScrypt(
			WithProviderID(p.ID),
			WithCommitment(commitment),
			WithVRFDifficulty(vrfDifficulty),
			WithScryptN(32),
		)
		require.NoError(t, err)
		require.NotNil(t, scrypt)
		defer scrypt.Close()

		res, err := scrypt.Positions(start, end)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Output)

		// assert that output content is equal between different providers.
		if prevOutput == nil {
			prevOutput = res.Output
		} else {
			require.Equal(t, prevOutput, res.Output, fmt.Sprintf("not equal: provider: %+v", p))
		}

		if nonce == nil {
			nonce = res.IdxSolution
		} else {
			require.Equal(t, *nonce, *res.IdxSolution)
		}
	}

	require.NotNil(t, prevOutput)
	require.Len(t, prevOutput, 16*int(end-start+1))
	require.NotNil(t, nonce)
}

func TestScrypt_Close(t *testing.T) {
	providers, err := OpenCLProviders()
	require.NoError(t, err)

	start := uint64(1)
	end := uint64(1 << 8)

	for _, p := range providers {
		scrypt, err := NewScrypt(
			WithProviderID(p.ID),
			WithCommitment(commitment),
			WithScryptN(32),
		)
		require.NoError(t, err)
		require.NotNil(t, scrypt)
		defer scrypt.Close()

		res, err := scrypt.Positions(start, end)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Output)
		require.NoError(t, scrypt.Close())

		reference := res.Output

		scrypt, err = NewScrypt(
			WithProviderID(p.ID),
			WithCommitment(commitment),
			WithScryptN(32),
		)
		require.NoError(t, err)
		require.NotNil(t, scrypt)
		defer scrypt.Close()

		res, err = scrypt.Positions(start, end)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Output)
		require.NoError(t, scrypt.Close())

		require.Equal(t, reference, res.Output)
	}
}

func TestScryptPositions_InvalidProviderId(t *testing.T) {
	invalidProviderId := uint(1 << 10)
	_, err := NewScrypt(
		WithProviderID(invalidProviderId),
		WithCommitment(commitment),
		WithVRFDifficulty(defaultDifficulty),
		WithScryptN(32),
	)
	require.ErrorIs(t, err, ErrInvalidProviderID)
}

func Test_ScryptPositions_Pow(t *testing.T) {
	providers, err := OpenCLProviders()
	require.NoError(t, err)

	commitment, err := hex.DecodeString("e26b543725490682675f6f84ea7689601adeaf14caa7024ec1140c82754ca339")
	require.NoError(t, err)

	vrfDifficulty := make([]byte, 32)
	copy(vrfDifficulty, defaultDifficulty)
	vrfDifficulty[0] = 0
	vrfDifficulty[1] = 0
	vrfDifficulty[2] = 0x3f
	require.NoError(t, err)

	start := uint64(1 << 10)
	end := uint64(1 << 18)
	nonce := uint64(165545)

	for _, p := range providers {
		scrypt, err := NewScrypt(
			WithProviderID(p.ID),
			WithCommitment(commitment),
			WithVRFDifficulty(vrfDifficulty),
			WithScryptN(32),
		)
		require.NoError(t, err)
		require.NotNil(t, scrypt)
		defer scrypt.Close()

		res, err := scrypt.Positions(start, end)
		require.NoError(t, err)
		require.NotNil(t, res.IdxSolution)
		require.Equal(t, nonce, *res.IdxSolution)
	}
}

func Test_ScryptPositions_NoPow(t *testing.T) {
	providers, err := OpenCLProviders()
	require.NoError(t, err)

	commitment, err := hex.DecodeString("e26b543725490682675f6f84ea7689601adeaf14caa7024ec1140c82754ca339")
	require.NoError(t, err)

	vrfDifficulty := make([]byte, 32)
	copy(vrfDifficulty, defaultDifficulty)
	vrfDifficulty[0] = 0
	vrfDifficulty[1] = 0
	vrfDifficulty[2] = 0
	require.NoError(t, err)

	start := uint64(1 << 10)
	end := uint64(1 << 18)

	for _, p := range providers {
		scrypt, err := NewScrypt(
			WithProviderID(p.ID),
			WithCommitment(commitment),
			WithVRFDifficulty(vrfDifficulty),
			WithScryptN(32),
		)
		require.NoError(t, err)
		require.NotNil(t, scrypt)
		defer scrypt.Close()

		res, err := scrypt.Positions(start, end)
		require.NoError(t, err)
		require.Nil(t, res.IdxSolution)
	}
}
