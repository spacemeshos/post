package initialization

import (
	"bytes"
	"context"
	"errors"
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

	var eg errgroup.Group
	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

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

	var eg errgroup.Group
	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

	// Initialize again using the same config & opts.
	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

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

	var eg errgroup.Group
	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

	// Increase `opts.NumUnits`.
	opts.NumUnits++

	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

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

	var eg errgroup.Group
	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

	// Decrease `opts.NumUnits`.
	opts.NumUnits--

	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

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

	var eg errgroup.Group
	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

	prevNumUnits := opts.NumUnits

	// Increase `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits + 1
	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	err = init.Initialize()
	cfgMissErr := &shared.ConfigMismatchError{}
	r.ErrorAs(err, cfgMissErr)
	r.Equal("NumUnits", cfgMissErr.Param)

	// Decrease `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits - 1
	init, err = NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	err = init.Initialize()
	r.ErrorAs(err, cfgMissErr)
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
	err = init.Initialize()
	r.NoError(err)

	oneFileData, err := initData(opts.DataDir, uint(cfg.BitsPerLabel))
	r.NoError(err)

	// Cleanup.
	err = init.Reset()
	r.NoError(err)

	for numFiles := uint32(2); numFiles <= 16; numFiles <<= 1 {
		opts := opts
		opts.NumFiles = numFiles

		init, err := NewInitializer(cfg, opts, commitment)
		r.NoError(err)
		err = init.Initialize()
		r.NoError(err)

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

	err = init.Initialize()
	r.NoError(err)
	m, err = init.loadMetadata()
	r.NoError(err)
	err = init.verifyMetadata(m)
	r.NoError(err)

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
	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		time.Sleep(2 * time.Second)
		return init.Stop()
	})
	var prev uint64
	eg.Go(func() error {
		for p := range init.SessionNumLabelsWrittenChan() {
			assert.Less(t, prev, p)
			prev = p
			t.Logf("num labels written: %v\n", p)
		}
		s, err := (init.Started())
		assert.NoError(t, err)
		assert.True(t, s)
		c, err := init.Completed()
		assert.False(t, c)
		assert.NoError(t, err)
		return nil
	})
	err = init.Initialize()
	r.Equal(ErrStopped, err)
	r.NoError(eg.Wait())

	// Continue the initialization to completion.
	eg.Go(assertNumLabelsWrittenChan(init, t))
	r.NoError(init.Initialize())
	r.NoError(eg.Wait())

	// Cleanup.
	r.NoError(init.Reset())
}

func Test_SessionNumLabelsWrittenChan_Racefree(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumUnits = 10
	commitment := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, commitment)
	r.NoError(err)
	init.SetLogger(log)

	eg, _ := errgroup.WithContext(context.Background())

	// call SessionNumLabelsWrittenChan() during initialization
	eg.Go(func() error {
		ch := make([]<-chan uint64, 0, 10)

		for i := 0; i < 10; i++ {
			time.Sleep(600 * time.Millisecond)
			ch = append(ch, init.SessionNumLabelsWrittenChan())
		}

		if ch[0] != nil {
			return errors.New("channel should be nil")
		}

		for _, c := range ch[1:] {
			<-c
		}
		return nil
	})
	eg.Go(func() error {
		time.Sleep(1 * time.Second)
		return init.Initialize()
	})
	r.NoError(eg.Wait())

	// call SessionNumLabelsWrittenChan() after initialization
	eg.Go(func() error {
		ch := make([]<-chan uint64, 0, 10)

		for i := 0; i < 10; i++ {
			ch = append(ch, init.SessionNumLabelsWrittenChan())
		}

		for _, c := range ch {
			select {
			case <-c:
			default:
				return errors.New("channel should be closed")
			}
		}
		return nil
	})
	r.NoError(eg.Wait())

	// Cleanup.
	r.NoError(init.Reset())
}

func assertNumLabelsWrittenChan(init *Initializer, t *testing.T) func() error {
	return func() error {
		// hack to avoid receiving a nil channel from SessionNumLabelsWrittenChan()
		// TODO (mafa): for a proper fix see https://github.com/spacemeshos/post/issues/78
		var labelsChan <-chan uint64
		assert.Eventually(t, func() bool {
			labelsChan = init.SessionNumLabelsWrittenChan()
			return labelsChan != nil
		}, time.Second, time.Millisecond)

		var prev uint64
		for p := range labelsChan {
			assert.Less(t, prev, p)
			prev = p
			t.Logf("num labels written: %v\n", p)
		}
		c, err := init.Completed()
		assert.True(t, c)
		return err
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
