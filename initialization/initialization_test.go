package initialization

import (
	"bytes"
	"context"
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

func getTestConfig(t *testing.T) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = CPUProviderID()

	return cfg, opts
}

type testLogger struct {
	shared.Logger

	t *testing.T
}

func (l testLogger) Info(msg string, args ...interface{})  { l.t.Logf("\tINFO\t"+msg, args...) }
func (l testLogger) Debug(msg string, args ...interface{}) { l.t.Logf("\tDEBUG\t"+msg, args...) }

func TestInitialize(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	commitment := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	// Cleanup.
	r.NoError(init.Reset())
}

func TestInitialize_Repeated(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	commitment := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	// Initialize again using the same config & opts.
	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	// Cleanup.
	r.NoError(init.Reset())
}

func TestInitialize_NumUnits_Increase(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumFiles = 1
	commitment := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	// Increase `opts.NumUnits`.
	opts.NumUnits++

	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	// Cleanup.
	r.NoError(init.Reset())
}

func TestInitialize_NumUnits_Decrease(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumUnits++
	opts.NumFiles = 1
	commitment := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	// Decrease `opts.NumUnits`.
	opts.NumUnits--

	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	// Cleanup.
	r.NoError(init.Reset())
}

func TestInitialize_NumUnits_MultipleFiles(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumUnits++
	opts.NumFiles = 2
	commitment := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()
	}

	prevNumUnits := opts.NumUnits

	// Increase `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits + 1
	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	cfgMissErr := &shared.ConfigMismatchError{}
	r.ErrorAs(init.Initialize(), cfgMissErr)
	r.Equal("NumUnits", cfgMissErr.Param)

	// Decrease `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits - 1
	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	r.ErrorAs(init.Initialize(), cfgMissErr)
	r.Equal("NumUnits", cfgMissErr.Param)

	// Cleanup.
	r.NoError(init.Reset())
}

func TestInitialize_MultipleFiles(t *testing.T) {
	r := require.New(t)

	cfg, opts := getTestConfig(t)
	commitment := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	r.NoError(init.Initialize())

	oneFileData, err := initData(opts.DataDir, uint(cfg.BitsPerLabel))
	r.NoError(err)

	// Cleanup.
	r.NoError(init.Reset())

	for numFiles := uint32(2); numFiles <= 16; numFiles <<= 1 {
		opts := opts
		opts.NumFiles = numFiles

		init, err := NewInitializer(cfg, opts, commitment)
		r.NoError(err)
		r.NoError(init.Initialize())

		multipleFilesData, err := initData(opts.DataDir, uint(cfg.BitsPerLabel))
		r.NoError(err)

		r.Equal(multipleFilesData, oneFileData)

		// Cleanup.
		r.NoError(init.Reset())
	}
}

func TestNumLabelsWritten(t *testing.T) {
	req := require.New(t)

	cfg, opts := getTestConfig(t)
	commitment := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, commitment)
	req.NoError(err)

	// Check initial state.
	numLabelsWritten, err := init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(0), numLabelsWritten)

	// Initialize.
	err = init.Initialize()
	req.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)

	// Initialize repeated.
	err = init.Initialize()
	req.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)

	// Initialize repeated, using a new instance.
	init, err = NewInitializer(cfg, opts, commitment)
	req.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)
	err = init.Initialize()
	req.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(opts.NumUnits)*cfg.LabelsPerUnit, numLabelsWritten)

	// Cleanup.
	req.NoError(init.Reset())
}

func TestValidateMetadata(t *testing.T) {
	r := require.New(t)

	cfg, opts := getTestConfig(t)
	commitment := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)

	m, err := init.loadMetadata()
	r.Equal(ErrStateMetadataFileMissing, err)
	r.Nil(m)

	r.NoError(init.Initialize())
	m, err = init.loadMetadata()
	r.NoError(err)
	r.NoError(init.verifyMetadata(m))

	// Attempt to initialize with different `Commitment`.
	newCommitment := make([]byte, 32)
	newCommitment[0] = newCommitment[0] + 1
	init, err = NewInitializer(cfg, opts, newCommitment)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok := err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("Commitment", errConfigMismatch.Param)

	// Attempt to initialize with different `cfg.BitsPerLabel`.
	newCfg := cfg
	newCfg.BitsPerLabel = cfg.BitsPerLabel + 1
	init, err = NewInitializer(newCfg, opts, commitment)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok = err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("BitsPerLabel", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumFiles`.
	newOpts := opts
	newOpts.NumFiles = 4
	init, err = NewInitializer(cfg, newOpts, commitment)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok = err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("NumFiles", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumUnits` while `opts.NumFiles` > 1.
	newOpts = opts
	newOpts.NumUnits++
	init, err = NewInitializer(cfg, newOpts, commitment)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok = err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("NumUnits", errConfigMismatch.Param)

	// Cleanup.
	r.NoError(init.Reset())
}

func TestStop(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumUnits = 10
	commitment := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	// Start initialization and stop it after a short while.
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(func() error {
			time.Sleep(2 * time.Second)
			return init.Stop()
		})
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.ErrorIs(init.Initialize(), ErrStopped)
		cancel()
		eg.Wait()

		c, err := init.Completed()
		assert.False(t, c)
		assert.NoError(t, err)
	}

	// Continue the initialization to completion.
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var eg errgroup.Group
		eg.Go(assertNumLabelsWritten(ctx, t, init))
		r.NoError(init.Initialize())
		cancel()
		eg.Wait()

		c, err := init.Completed()
		assert.True(t, c)
		assert.NoError(t, err)
	}

	// Cleanup.
	r.NoError(init.Reset())
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
