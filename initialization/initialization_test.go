package initialization

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

var (
	nodeId          = make([]byte, 32)
	commitmentAtxId = make([]byte, 32)
)

func TestInitialize(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())
	opts.Scrypt.N = 16

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	require.NoError(t, err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		require.NoError(t, init.Initialize(ctx))
		cancel()
		eg.Wait()
	}
	require.Equal(t, uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	m := &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		LabelsPerUnit:   cfg.LabelsPerUnit,
	}
	require.NoError(t, verifying.VerifyVRFNonce(init.Nonce(), m, verifying.WithLabelScryptParams(opts.Scrypt)))
}

func TestInitialize_PowOutOfRange(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 8

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())
	opts.Scrypt.N = 16

	// nodeId where no label in the first uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit satisfies the PoW requirement.
	nodeId, err := hex.DecodeString("57fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649")
	r.NoError(err)

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
		// use a higher difficulty to make sure no Pow is found in the first `numLabels` labels.
		withDifficultyFunc(func(numLabels uint64) []byte {
			x := new(big.Int).Lsh(big.NewInt(1), 256)
			x.Div(x, big.NewInt(int64(numLabels)))

			difficulty := make([]byte, 32)
			return x.FillBytes(difficulty)
		}),
	)
	r.NoError(err)

	r.NoError(init.Initialize(context.Background()))
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	m := &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		LabelsPerUnit:   cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), m, verifying.WithLabelScryptParams(opts.Scrypt)))

	// check that the found nonce is outside of the range for calculating labels
	r.GreaterOrEqual(*init.Nonce(), uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit)
}

func TestInitialize_ContinueWithLastPos(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())
	opts.Scrypt.N = 16

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	r.NoError(init.Initialize(context.Background()))
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	meta := &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		LabelsPerUnit:   cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), meta, verifying.WithLabelScryptParams(opts.Scrypt)))

	// trying again returns same nonce
	origNonce := *init.Nonce()
	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	r.NoError(init.Initialize(context.Background()))
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	m, err := LoadMetadata(opts.DataDir)
	r.NoError(err)
	r.Equal(origNonce, *m.Nonce)
	r.Nil(m.LastPosition)

	// lastPos lower than numLabels is ignored
	m.LastPosition = new(uint64)
	*m.LastPosition = uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit - 10
	r.NoError(SaveMetadata(opts.DataDir, m))

	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	r.NoError(init.Initialize(context.Background()))
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	m, err = LoadMetadata(opts.DataDir)
	r.NoError(err)
	r.Equal(origNonce, *m.Nonce)

	// no nonce found and lastPos not set finds a higher nonce than numLabels
	m.Nonce = nil
	r.NoError(SaveMetadata(opts.DataDir, m))

	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	r.NoError(init.Initialize(context.Background()))
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	m, err = LoadMetadata(opts.DataDir)
	r.NoError(err)
	r.NotNil(m.Nonce)
	r.NotNil(m.LastPosition)
	r.LessOrEqual(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, *m.LastPosition)
	r.LessOrEqual(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, *m.Nonce)

	meta = &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		LabelsPerUnit:   cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), meta, verifying.WithLabelScryptParams(opts.Scrypt)))

	// lastPos sets lower bound for searching for nonce if none was found
	lastPos := *m.Nonce + 10
	*m.LastPosition = lastPos
	m.Nonce = nil
	r.NoError(SaveMetadata(opts.DataDir, m))

	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	r.NoError(init.Initialize(context.Background()))
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	m, err = LoadMetadata(opts.DataDir)
	r.NoError(err)
	r.NotNil(m.Nonce)
	r.NotNil(m.LastPosition)
	r.LessOrEqual(lastPos, *m.LastPosition)

	r.Less(lastPos, *m.Nonce)

	meta = &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		LabelsPerUnit:   cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), meta, verifying.WithLabelScryptParams(opts.Scrypt)))
}

func TestReset_WhileInitializing(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 15

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())
	opts.ComputeBatchSize = 1 << 14

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		var eg errgroup.Group
		eg.Go(func() error {
			r.Eventually(func() bool { return init.NumLabelsWritten() > 0 }, 5*time.Second, 5*time.Millisecond)
			r.ErrorIs(init.Reset(), ErrCannotResetWhileInitializing)
			return nil
		})
		eg.Go(func() error { return init.Initialize(context.Background()) })
		eg.Wait()

		r.NoError(init.Reset())
	}
}

func TestInitialize_Repeated(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()
	}

	// Initialize again using the same config & opts.
	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()
	}
}

func TestInitialize_NumUnits_Increase(t *testing.T) {
	t.Skip("not supported yet, see https://github.com/spacemeshos/go-spacemesh/issues/3759")

	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()
	}

	// Increase `opts.NumUnits`.
	opts.NumUnits++
	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()
	}
}

func TestInitialize_NumUnits_Decrease(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits + 1
	opts.ProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()
	}

	// Decrease `opts.NumUnits`.
	opts.NumUnits--
	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()
	}
}

func TestInitialize_RedundantFiles(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits + 1
	opts.MaxFileSize = 1 << 12
	opts.ProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()
	}

	// Decrease `opts.NumUnits`.
	newOpts := opts
	newOpts.NumUnits = opts.NumUnits - 1
	newInit, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(newOpts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))

		numFiles, err := init.diskState.NumFilesWritten()
		r.NoError(err)
		layout, err := deriveFilesLayout(cfg, opts)
		r.NoError(err)
		r.Equal(int(layout.NumFiles), numFiles)

		r.NoError(newInit.Initialize(ctx))

		numFiles, err = newInit.diskState.NumFilesWritten()
		r.NoError(err)
		newLayout, err := deriveFilesLayout(cfg, newOpts)
		r.NoError(err)
		r.Equal(int(newLayout.NumFiles), numFiles)
		r.Less(newLayout.NumFiles, layout.NumFiles)

		cancel()
		eg.Wait()
	}
}

func TestInitialize_MultipleFiles(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 14

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.MaxFileSize = uint64(opts.NumUnits) * cfg.LabelsPerUnit * config.BitsPerLabel / 8
	opts.ProviderID = int(CPUProviderID())

	var oneFileData []byte
	var oneFileNonce uint64

	t.Run("NumFiles: 1", func(t *testing.T) {
		init, err := NewInitializer(
			WithNodeId(nodeId),
			WithCommitmentAtxId(commitmentAtxId),
			WithConfig(cfg),
			WithInitOpts(opts),
			WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
		)
		require.NoError(t, err)
		require.NoError(t, init.Initialize(context.Background()))

		oneFileData, err = initData(opts.DataDir)
		require.NoError(t, err)

		m := &shared.VRFNonceMetadata{
			NodeId:          nodeId,
			CommitmentAtxId: commitmentAtxId,
			NumUnits:        opts.NumUnits,
			LabelsPerUnit:   cfg.LabelsPerUnit,
		}
		require.NoError(t, verifying.VerifyVRFNonce(init.Nonce(), m, verifying.WithLabelScryptParams(opts.Scrypt)))
		oneFileNonce = *init.Nonce()
	})

	for numFiles := 2; numFiles <= 16; numFiles *= 2 {
		t.Run(fmt.Sprintf("NumFiles: %d", numFiles), func(t *testing.T) {
			opts := opts
			opts.MaxFileSize /= uint64(numFiles)
			opts.DataDir = t.TempDir()

			layout, err := deriveFilesLayout(cfg, opts)
			require.NoError(t, err)
			require.Equal(t, numFiles, int(layout.NumFiles))

			init, err := NewInitializer(
				WithNodeId(nodeId),
				WithCommitmentAtxId(commitmentAtxId),
				WithConfig(cfg),
				WithInitOpts(opts),
				WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
			)
			require.NoError(t, err)
			require.NoError(t, init.Initialize(context.Background()))

			multipleFilesData, err := initData(opts.DataDir)
			require.NoError(t, err)

			require.Equal(t, oneFileData, multipleFilesData)

			m := &shared.VRFNonceMetadata{
				NodeId:          nodeId,
				CommitmentAtxId: commitmentAtxId,
				NumUnits:        opts.NumUnits,
				LabelsPerUnit:   cfg.LabelsPerUnit,
			}
			require.NoError(t, verifying.VerifyVRFNonce(init.Nonce(), m, verifying.WithLabelScryptParams(opts.Scrypt)))
			require.Equal(t, oneFileNonce, *init.Nonce())
		})
	}
}

func TestNumLabelsWritten(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	// Check initial state.
	numLabelsWritten, err := init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(0), numLabelsWritten)

	// Initialize.
	r.NoError(init.Initialize(context.Background()))
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)

	// Initialize repeated.
	r.NoError(init.Initialize(context.Background()))
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)

	// Initialize repeated, using a new instance.
	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)
	r.NoError(init.Initialize(context.Background()))
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)
}

func TestValidateMetadata(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 16
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)

	m, err := init.loadMetadata()
	r.NoError(err)
	r.NoError(init.verifyMetadata(m))

	r.NoError(init.Initialize(context.Background()))
	m, err = init.loadMetadata()
	r.NoError(err)
	r.NoError(init.verifyMetadata(m))

	// Attempt to initialize with different `NodeId`.
	newNodeId := make([]byte, 32)
	newNodeId[0] = newNodeId[0] + 1
	_, err = NewInitializer(
		WithNodeId(newNodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	var errConfigMismatch ConfigMismatchError
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("NodeId", errConfigMismatch.Param)

	// Attempt to initialize with different `AtxId`.
	newAtxId := make([]byte, 32)
	newAtxId[0] = newAtxId[0] + 1
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(newAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("CommitmentAtxId", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.MaxFileSize`.
	newOpts := opts
	newOpts.MaxFileSize = opts.MaxFileSize + 1
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(newOpts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("MaxFileSize", errConfigMismatch.Param)

	// Attempt to initialize with a higher `opts.NumUnits`.
	newOpts = opts
	newOpts.NumUnits++
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(newOpts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("NumUnits", errConfigMismatch.Param)
}

func TestStop(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.Scrypt.N = 64
	opts.DataDir = t.TempDir()
	opts.NumUnits = 10
	opts.ProviderID = int(CPUProviderID())
	opts.ComputeBatchSize = 1 << 10

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)
	r.Equal(StatusNotStarted, init.Status())

	// Start initialization and stop it after it has written some labels
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(func() error {
			r.Eventually(func() bool { return init.NumLabelsWritten() > 0 }, 5*time.Second, 5*time.Millisecond)
			cancel()
			return nil
		})
		r.ErrorIs(init.Initialize(ctx), context.Canceled)
		eg.Wait()

		r.Equal(StatusStarted, init.Status())
	}

	// Continue the initialization to completion.
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		eg.Go(func() error {
			r.Eventually(func() bool { return init.Status() == StatusInitializing }, 5*time.Second, 5*time.Millisecond)
			return nil
		})
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()

		r.Equal(StatusCompleted, init.Status())
	}
}

func TestValidateComputeBatchSize(t *testing.T) {
	cfg := config.DefaultConfig()
	opts := config.DefaultInitOpts()

	// Set invalid value of 0
	opts.ComputeBatchSize = 0

	_, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	assert.Error(t, err)
}

func assertNumLabelsWritten(ctx context.Context, t *testing.T, init *Initializer) func() error {
	return func() error {
		timer := time.NewTimer(50 * time.Millisecond)
		defer timer.Stop()
		var prev uint64

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				num := init.NumLabelsWritten()
				t.Logf("num labels written: %v\n", num)
				assert.GreaterOrEqual(t, num, prev)
				prev = num
			}
		}
	}
}

func initData(datadir string) ([]byte, error) {
	reader, err := persistence.NewLabelsReader(datadir, config.BitsPerLabel)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func TestInitializeSubset(t *testing.T) {
	r := require.New(t)
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 128

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = 20
	opts.MaxFileSize = cfg.LabelsPerUnit * 2 * uint64(config.BytesPerLabel()) // 2 units per file
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

	optsSubset := opts
	optsSubset.DataDir = t.TempDir()
	optsSubset.FromFileIdx = 3
	optsSubset.ToFileIdx = new(int)
	*optsSubset.ToFileIdx = 4

	initSubset, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(optsSubset),
		WithLogger(zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))),
	)
	r.NoError(err)
	err = initSubset.Initialize(context.Background())
	r.NoError(err)

	// Verify that the subset is a subset of the full set
	fullData, err := initData(opts.DataDir)
	r.NoError(err)
	subsetData, err := initData(optsSubset.DataDir)
	r.NoError(err)
	r.True(bytes.Contains(fullData, subsetData))

	// Verify that the subset contains files 3 and 4, but not 0-2 and 5
	_, err = os.Stat(filepath.Join(optsSubset.DataDir, "postdata_0.bin"))
	r.ErrorIs(err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(optsSubset.DataDir, "postdata_1.bin"))
	r.ErrorIs(err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(optsSubset.DataDir, "postdata_2.bin"))
	r.ErrorIs(err, os.ErrNotExist)

	_, err = os.Stat(filepath.Join(optsSubset.DataDir, "postdata_3.bin"))
	r.NoError(err)
	_, err = os.Stat(filepath.Join(optsSubset.DataDir, "postdata_4.bin"))
	r.NoError(err)

	_, err = os.Stat(filepath.Join(optsSubset.DataDir, "postdata_5.bin"))
	r.ErrorIs(err, os.ErrNotExist)

	// Verify that postdata_3.bin from both initializations contain the same data
	fullPostdata3, err := os.ReadFile(filepath.Join(opts.DataDir, "postdata_3.bin"))
	r.NoError(err)
	subsetPostdata3, err := os.ReadFile(filepath.Join(optsSubset.DataDir, "postdata_3.bin"))
	r.NoError(err)
	r.Equal(fullPostdata3, subsetPostdata3)

	// Verify that postdata_4.bin from both initializations contain the same data
	fullPostdata4, err := os.ReadFile(filepath.Join(opts.DataDir, "postdata_4.bin"))
	r.NoError(err)
	subsetPostdata4, err := os.ReadFile(filepath.Join(optsSubset.DataDir, "postdata_4.bin"))
	r.NoError(err)
	r.Equal(fullPostdata4, subsetPostdata4)
}
