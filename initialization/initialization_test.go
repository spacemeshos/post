package initialization

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type testLogger struct {
	shared.Logger

	t *testing.T
}

func (l testLogger) Info(msg string, args ...any)  { l.t.Logf("\tINFO\t"+msg, args...) }
func (l testLogger) Debug(msg string, args ...any) { l.t.Logf("\tDEBUG\t"+msg, args...) }

func TestInitialize(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	m := &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), m))
}

func TestInitialize_PowOutOfRange(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	// nodeId where no label in the first uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit satisfies the PoW requirement.
	nodeId, err := hex.DecodeString("52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649")
	r.NoError(err)

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), m))

	// check that the found nonce is outside of the range for calculating labels
	r.Less(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, *init.Nonce())
}

func TestInitialize_ContinueWithLastPos(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)

	r.NoError(init.Initialize(context.Background()))
	r.Equal(uint64(cfg.MinNumUnits)*cfg.LabelsPerUnit, init.NumLabelsWritten())

	meta := &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), meta))

	// trying again returns same nonce
	origNonce := *init.Nonce()
	init, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
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
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), meta))

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
		WithLogger(testLogger{t: t}),
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
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), meta))
}

func TestReset_WhileInitializing(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)

	{
		var eg errgroup.Group
		eg.Go(func() error {
			r.Eventually(func() bool { return init.NumLabelsWritten() > 0 }, 5*time.Second, 50*time.Millisecond)
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
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
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
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 1
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
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
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits + 1
	opts.NumFiles = 1
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
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

func TestInitialize_NumUnits_MultipleFiles(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits + 1
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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

	prevNumUnits := opts.NumUnits

	// Increase `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits + 1
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	var cfgMissErr shared.ConfigMismatchError
	r.ErrorAs(err, &cfgMissErr)
	r.Equal("NumUnits", cfgMissErr.Param)

	// Decrease `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits - 1
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.ErrorAs(err, &cfgMissErr)
	r.Equal("NumUnits", cfgMissErr.Param)
}

func TestInitialize_MultipleFiles(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	oneFileData, err := initData(opts.DataDir, uint(cfg.BitsPerLabel))
	r.NoError(err)

	m := &shared.VRFNonceMetadata{
		NodeId:          nodeId,
		CommitmentAtxId: commitmentAtxId,
		NumUnits:        opts.NumUnits,
		BitsPerLabel:    cfg.BitsPerLabel,
		LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
	}
	r.NoError(verifying.VerifyVRFNonce(init.Nonce(), m))

	// TODO(mafa): since we are not looking for the absolute lowest nonce, we can't guarantee that the nonce will be the same.
	// see also https://github.com/spacemeshos/post/issues/90
	// oneFileNonce := *m.Nonce

	for numFiles := uint32(2); numFiles <= 16; numFiles <<= 1 {
		t.Run(fmt.Sprintf("NumFiles=%d", numFiles), func(t *testing.T) {
			r := require.New(t)
			opts := opts
			opts.NumFiles = numFiles
			opts.DataDir = t.TempDir()

			init, err := NewInitializer(
				WithNodeId(nodeId),
				WithCommitmentAtxId(commitmentAtxId),
				WithConfig(cfg),
				WithInitOpts(opts),
				WithLogger(testLogger{t: t}),
			)
			r.NoError(err)
			r.NoError(init.Initialize(context.Background()))

			multipleFilesData, err := initData(opts.DataDir, uint(cfg.BitsPerLabel))
			r.NoError(err)

			r.Equal(multipleFilesData, oneFileData)

			m := &shared.VRFNonceMetadata{
				NodeId:          nodeId,
				CommitmentAtxId: commitmentAtxId,
				NumUnits:        opts.NumUnits,
				BitsPerLabel:    cfg.BitsPerLabel,
				LabelsPerUnit:   uint64(opts.NumUnits) * cfg.LabelsPerUnit,
			}
			r.NoError(verifying.VerifyVRFNonce(init.Nonce(), m))

			// TODO(mafa): see above
			// r.Equal(oneFileNonce, *init.Nonce())
		})
	}
}

func TestNumLabelsWritten(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
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
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
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
		WithLogger(testLogger{t: t}),
	)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("CommitmentAtxId", errConfigMismatch.Param)

	// Attempt to initialize with different `cfg.BitsPerLabel`.
	newCfg := cfg
	newCfg.BitsPerLabel = cfg.BitsPerLabel + 1
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(newCfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("BitsPerLabel", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumFiles`.
	newOpts := opts
	newOpts.NumFiles = 4
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(newOpts),
		WithLogger(testLogger{t: t}),
	)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("NumFiles", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumUnits` while `opts.NumFiles` > 1.
	newOpts = opts
	newOpts.NumUnits++
	_, err = NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(newOpts),
		WithLogger(testLogger{t: t}),
	)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("NumUnits", errConfigMismatch.Param)
}

func TestStop(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = 10
	opts.NumFiles = 5
	opts.ComputeProviderID = int(CPUProviderID())

	init, err := NewInitializer(
		WithNodeId(nodeId),
		WithCommitmentAtxId(commitmentAtxId),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.Equal(StatusNotStarted, init.Status())

	// Start initialization and stop it after it has written some labels
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(func() error {
			r.Eventually(func() bool { return init.NumLabelsWritten() > 0 }, 5*time.Second, 50*time.Millisecond)
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
			r.Eventually(func() bool { return init.Status() == StatusInitializing }, 5*time.Second, 50*time.Millisecond)
			return nil
		})
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()

		r.Equal(StatusCompleted, init.Status())
	}
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

func initData(datadir string, bitsPerLabel uint) ([]byte, error) {
	reader, err := persistence.NewLabelsReader(datadir, bitsPerLabel)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	gsReader := shared.NewGranSpecificReader(reader, bitsPerLabel)
	writer := bytes.NewBuffer(nil)
	for {
		b, err := gsReader.ReadNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		writer.Write(b)
	}

	return writer.Bytes(), nil
}
