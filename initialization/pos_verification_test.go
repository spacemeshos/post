package initialization

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/internal/postrs"
)

func TestVerifyPos(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 128

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = 5
	opts.MaxFileSize = 2 * cfg.UnitSize()
	opts.ProviderID = int(CPUProviderID())
	opts.Scrypt.N = 2

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	require.NoError(t, err)
	err = init.Initialize(context.Background())
	require.NoError(t, err)

	scryptParams := postrs.TranslateScryptParams(opts.Scrypt.N, opts.Scrypt.R, opts.Scrypt.P)

	t.Run("valid", func(t *testing.T) {
		err := postrs.VerifyPos(opts.DataDir, scryptParams, postrs.WithFraction(100.0))
		require.NoError(t, err)
	})
	t.Run("invalid N", func(t *testing.T) {
		wrongScrypt := postrs.TranslateScryptParams(4, opts.Scrypt.R, opts.Scrypt.P)
		err := postrs.VerifyPos(opts.DataDir, wrongScrypt, postrs.WithFraction(100.0))
		require.ErrorIs(t, err, postrs.ErrInvalidPos)
	})
	t.Run("corrupted data", func(t *testing.T) {
		file, err := os.OpenFile(opts.DataDir+"/postdata_0.bin", os.O_WRONLY, 0)
		require.NoError(t, err)
		defer file.Close()
		_, err = file.WriteAt([]byte("1234567890123456"), 0)
		require.NoError(t, err)

		err = postrs.VerifyPos(opts.DataDir, scryptParams, postrs.WithFraction(100.0))
		require.ErrorIs(t, err, postrs.ErrInvalidPos)
	})
}
