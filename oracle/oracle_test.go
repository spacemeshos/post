package oracle

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/internal/postrs/mocks"
	"github.com/stretchr/testify/require"
)

func TestOracleRetryPositions(t *testing.T) {
	t.Parallel()
	commitment := make([]byte, 32)
	vrfDifficulty := make([]byte, 32)
	mockScrypter := mocks.NewMockScrypter(gomock.NewController(t))
	o, err := New(WithCommitment(commitment), WithVRFDifficulty(vrfDifficulty), WithMaxRetries(2), WithRetryDelay(0), WithScrypter(mockScrypter))

	t.Run("retries max time and quits", func(t *testing.T) {
		mockScrypter.EXPECT().Positions(uint64(0), uint64(10)).Return(postrs.ScryptPositionsResult{}, postrs.ErrInitializationFailed).Times(3)
		require.NoError(t, err)
		_, err = o.Positions(0, 10)
		require.Error(t, err)
	})
	t.Run("eventually succeeds", func(t *testing.T) {
		mockScrypter.EXPECT().Positions(uint64(0), uint64(10)).Return(postrs.ScryptPositionsResult{}, postrs.ErrInitializationFailed).Times(2)
		mockScrypter.EXPECT().Positions(uint64(0), uint64(10)).Return(postrs.ScryptPositionsResult{}, nil).Times(1)
		_, err = o.Positions(0, 10)
		require.NoError(t, err)
	})
	t.Run("does not retry on unknown error", func(t *testing.T) {
		mockScrypter.EXPECT().Positions(uint64(0), uint64(10)).Return(postrs.ScryptPositionsResult{}, errors.New("unknown error")).Times(1)
		_, err = o.Positions(0, 10)
		require.Error(t, err)
	})
}

func TestOracleFailsOnInvalidIndices(t *testing.T) {
	t.Parallel()
	commitment := make([]byte, 32)
	vrfDifficulty := make([]byte, 32)
	mockScrypter := mocks.NewMockScrypter(gomock.NewController(t))
	o, err := New(WithCommitment(commitment), WithVRFDifficulty(vrfDifficulty), WithMaxRetries(2), WithRetryDelay(0), WithScrypter(mockScrypter))
	require.NoError(t, err)

	_, err = o.Positions(10, 0)
	require.Error(t, err)
}

func TestOracleCantInitializeAfterClose(t *testing.T) {
	t.Parallel()
	commitment := make([]byte, 32)
	vrfDifficulty := make([]byte, 32)
	mockScrypter := mocks.NewMockScrypter(gomock.NewController(t))
	o, err := New(WithCommitment(commitment), WithVRFDifficulty(vrfDifficulty), WithMaxRetries(2), WithRetryDelay(0), WithScrypter(mockScrypter))
	require.NoError(t, err)

	mockScrypter.EXPECT().Close().Return(nil).Times(1)
	require.NoError(t, o.Close())

	_, err = o.Positions(0, 10)
	require.Error(t, err)
}
