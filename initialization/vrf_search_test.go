package initialization

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

func TestCheckLabel(t *testing.T) {
	cpuProviderID := postrs.CPUProviderID()
	woReference, err := oracle.New(
		oracle.WithProviderID(&cpuProviderID),
		oracle.WithCommitment(make([]byte, 32)),
		oracle.WithVRFDifficulty(make([]byte, 32)),
		oracle.WithScryptParams(shared.ScryptParams{
			N: 2,
			R: 1,
			P: 1,
		}),
	)
	require.NoError(t, err)

	res, err := woReference.Position(77)
	require.NoError(t, err)

	t.Run("label < difficulty", func(t *testing.T) {
		label := make([]byte, 32)
		difficulty := res.Output
		ok, err := checkLabel(77, label, difficulty, woReference)
		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("label > difficulty", func(t *testing.T) {
		label := res.Output
		difficulty := make([]byte, 32)
		ok, err := checkLabel(77, label, difficulty, woReference)
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("label == difficulty", func(t *testing.T) {
		label := res.Output
		difficulty := res.Output
		ok, err := checkLabel(77, label, difficulty, woReference)
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("label MSB == difficulty / LSB > difficulty", func(t *testing.T) {
		label := res.Output
		difficulty := res.Output
		copy(difficulty[16:], bytes.Repeat([]byte{0}, 16))
		ok, err := checkLabel(77, label, difficulty, woReference)
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("label MSB == difficulty / LSB < difficulty", func(t *testing.T) {
		label := res.Output
		difficulty := append(res.Output, bytes.Repeat([]byte{0xff}, 16)...)
		ok, err := checkLabel(77, label, difficulty, woReference)
		require.NoError(t, err)
		require.True(t, ok)
	})
}

func TestSearchForNonce(t *testing.T) {
	// Initialize some data first
	cfg, opts := getTestConfig(t)
	opts.NumUnits = 20
	opts.MaxFileSize = cfg.UnitSize() * 2

	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(logger),
	)
	require.NoError(t, err)
	err = init.Initialize(context.Background())
	require.NoError(t, err)

	metadata, err := LoadMetadata(opts.DataDir)
	require.NoError(t, err)

	expectedNonce := metadata.Nonce
	expectedNonceValue := metadata.NonceValue
	// remove Nonce and NonceValue from metadata
	metadata.Nonce = nil
	metadata.NonceValue = nil
	err = SaveMetadata(opts.DataDir, metadata)
	require.NoError(t, err)

	nonce, value, err := SearchForNonce(
		context.Background(),
		cfg,
		opts,
		SearchWithLogger(logger),
	)
	require.NoError(t, err)
	require.Equal(t, *expectedNonce, nonce)
	require.EqualValues(t, expectedNonceValue, value)

	// Verify that nonce was written to the metatada file
	metadata, err = LoadMetadata(opts.DataDir)
	require.NoError(t, err)
	require.Equal(t, expectedNonce, metadata.Nonce)
	require.EqualValues(t, expectedNonceValue, metadata.NonceValue)

	// Search only in 1 file - nonce not found
	opts.FromFileIdx = 2
	opts.ToFileIdx = &opts.FromFileIdx

	_, _, err = SearchForNonce(
		context.Background(),
		cfg,
		opts,
		SearchWithLogger(logger),
	)
	require.ErrorIs(t, err, ErrNonceNotFound)
}

func TestSearchForNonceNotFound(t *testing.T) {
	cfg, opts := getTestConfig(t)
	opts.NumUnits = 10
	opts.MaxFileSize = cfg.UnitSize() * 2

	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(logger),
	)
	require.NoError(t, err)
	err = init.Initialize(context.Background())
	require.NoError(t, err)

	_, _, err = SearchForNonce(
		context.Background(),
		cfg,
		opts,
		SearchWithLogger(logger),
		searchWithPowDifficultyFunc(func(uint64) []byte { return make([]byte, 32) }),
	)
	require.ErrorIs(t, err, ErrNonceNotFound)
}
