package initialization

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
)

func getTestOptions(t *testing.T) []initializeOptionFunc {
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = CPUProviderID()

	return []initializeOptionFunc{
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	}
}

type testLogger struct {
	shared.Logger

	t *testing.T
}

func (l testLogger) Info(msg string, args ...interface{})  { l.t.Logf("\tINFO\t"+msg, args...) }
func (l testLogger) Debug(msg string, args ...interface{}) { l.t.Logf("\tDEBUG\t"+msg, args...) }

func TestInitialize(t *testing.T) {
	r := require.New(t)
	init, err := NewInitializer(getTestOptions(t)...)
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

func TestInitialize_Repeated(t *testing.T) {
	r := require.New(t)
	opts := getTestOptions(t)
	init, err := NewInitializer(opts...)
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
	init, err = NewInitializer(opts...)
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
	opts.ComputeProviderID = CPUProviderID()

	init, err := NewInitializer(
		WithCommitment(make([]byte, 32)),
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
		WithCommitment(make([]byte, 32)),
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
	opts.ComputeProviderID = CPUProviderID()

	init, err := NewInitializer(
		WithCommitment(make([]byte, 32)),
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
		WithCommitment(make([]byte, 32)),
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
	opts.ComputeProviderID = CPUProviderID()

	init, err := NewInitializer(
		WithCommitment(make([]byte, 32)),
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
	init, err = NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	cfgMissErr := &shared.ConfigMismatchError{}
	r.ErrorAs(init.Initialize(context.Background()), cfgMissErr)
	r.Equal("NumUnits", cfgMissErr.Param)

	// Decrease `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits - 1
	init, err = NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.ErrorAs(init.Initialize(context.Background()), cfgMissErr)
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
	opts.ComputeProviderID = CPUProviderID()

	init, err := NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	oneFileData, err := initData(opts.DataDir, uint(cfg.BitsPerLabel))
	r.NoError(err)

	for numFiles := uint32(2); numFiles <= 16; numFiles <<= 1 {
		t.Run(fmt.Sprintf("NumFiles=%d", numFiles), func(t *testing.T) {
			opts := opts
			opts.NumFiles = numFiles
			opts.DataDir = t.TempDir()

			init, err := NewInitializer(
				WithCommitment(make([]byte, 32)),
				WithConfig(cfg),
				WithInitOpts(opts),
				WithLogger(testLogger{t: t}),
			)
			r.NoError(err)
			r.NoError(init.Initialize(context.Background()))

			multipleFilesData, err := initData(opts.DataDir, uint(cfg.BitsPerLabel))
			r.NoError(err)

			r.Equal(multipleFilesData, oneFileData)
		})
	}
}

func TestNumLabelsWritten(t *testing.T) {
	r := require.New(t)

	opts := getTestOptions(t)
	init, err := NewInitializer(opts...)
	r.NoError(err)

	// Check initial state.
	numLabelsWritten, err := init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(0), numLabelsWritten)

	// Initialize.
	r.NoError(init.Initialize(context.Background()))
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(init.opts.NumUnits)*init.cfg.LabelsPerUnit, numLabelsWritten)

	// Initialize repeated.
	r.NoError(init.Initialize(context.Background()))
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(init.opts.NumUnits)*init.cfg.LabelsPerUnit, numLabelsWritten)

	// Initialize repeated, using a new instance.
	init, err = NewInitializer(opts...)
	r.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(init.opts.NumUnits)*init.cfg.LabelsPerUnit, numLabelsWritten)
	r.NoError(init.Initialize(context.Background()))
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	r.NoError(err)
	r.Equal(uint64(init.opts.NumUnits)*init.cfg.LabelsPerUnit, numLabelsWritten)
}

func TestValidateMetadata(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = CPUProviderID()

	init, err := NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)

	m, err := init.loadMetadata()
	r.Equal(ErrStateMetadataFileMissing, err)
	r.Nil(m)

	r.NoError(init.Initialize(context.Background()))
	m, err = init.loadMetadata()
	r.NoError(err)
	r.NoError(init.verifyMetadata(m))

	// Attempt to initialize with different `Commitment`.
	newCommitment := make([]byte, 32)
	newCommitment[0] = newCommitment[0] + 1
	init, err = NewInitializer(
		WithCommitment(newCommitment),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	var errConfigMismatch ConfigMismatchError
	r.ErrorAs(init.Initialize(context.Background()), &errConfigMismatch)
	r.Equal("Commitment", errConfigMismatch.Param)

	// Attempt to initialize with different `cfg.BitsPerLabel`.
	newCfg := cfg
	newCfg.BitsPerLabel = cfg.BitsPerLabel + 1
	init, err = NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(newCfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.ErrorAs(init.Initialize(context.Background()), &errConfigMismatch)
	r.Equal("BitsPerLabel", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumFiles`.
	newOpts := opts
	newOpts.NumFiles = 4
	init, err = NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(newOpts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.ErrorAs(init.Initialize(context.Background()), &errConfigMismatch)
	r.Equal("NumFiles", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumUnits` while `opts.NumFiles` > 1.
	newOpts = opts
	newOpts.NumUnits++
	init, err = NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(newOpts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.ErrorAs(init.Initialize(context.Background()), &errConfigMismatch)
	r.Equal("NumUnits", errConfigMismatch.Param)
}

func TestStop(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = 10
	opts.NumFiles = 2
	opts.ComputeProviderID = CPUProviderID()

	init, err := NewInitializer(
		WithCommitment(make([]byte, 32)),
		WithConfig(cfg),
		WithInitOpts(opts),
		WithLogger(testLogger{t: t}),
	)
	r.NoError(err)

	// Start initialization and stop it after a short while.
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(func() error {
			time.Sleep(2 * time.Second)
			cancel()
			return nil
		})
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.ErrorIs(init.Initialize(ctx), context.Canceled)
		eg.Wait()

		status := init.Status()
		assert.Equal(t, StatusStarted, status)
	}

	// Continue the initialization to completion.
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize(ctx))
		cancel()
		eg.Wait()

		status := init.Status()
		assert.Equal(t, StatusCompleted, status)
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
				num := init.SessionNumLabelsWritten()
				t.Logf("num labels written: %v\n", num)
				assert.LessOrEqual(t, prev, num)
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
