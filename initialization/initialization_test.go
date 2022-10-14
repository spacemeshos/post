package initialization

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
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
	id := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestInitialize_Repeated(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	id := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Initialize again using the same config & opts.
	init, err = NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan = assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestInitialize_NumUnits_Increase(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumFiles = 1
	id := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Increase `opts.NumUnits`.
	opts.NumUnits++

	init, err = NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan = assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestInitialize_NumUnits_Decrease(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumUnits++
	opts.NumFiles = 1
	id := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Decrease `opts.NumUnits`.
	opts.NumUnits--

	init, err = NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan = assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestInitialize_NumUnits_MultipleFiles(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumUnits++
	opts.NumFiles = 2
	id := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	prevNumUnits := opts.NumUnits

	// Increase `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits + 1
	init, err = NewInitializer(cfg, opts, id)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok := err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("NumUnits", errConfigMismatch.Param)

	// Decrease `opts.NumUnits` while `opts.NumFiles` > 1.
	opts.NumUnits = prevNumUnits - 1
	init, err = NewInitializer(cfg, opts, id)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok = err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("NumUnits", errConfigMismatch.Param)

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestInitialize_MultipleFiles(t *testing.T) {
	r := require.New(t)

	cfg, opts := getTestConfig(t)
	id := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	err = init.Initialize()
	r.NoError(err)

	oneFileData, err := initData(opts.DataDir, cfg.BitsPerLabel)
	r.NoError(err)

	// Cleanup.
	err = init.Reset()
	r.NoError(err)

	for numFiles := uint(2); numFiles <= 16; numFiles <<= 1 {
		opts := opts
		opts.NumFiles = numFiles

		init, err := NewInitializer(cfg, opts, id)
		r.NoError(err)
		err = init.Initialize()
		r.NoError(err)

		multipleFilesData, err := initData(opts.DataDir, cfg.BitsPerLabel)
		r.NoError(err)

		r.Equal(multipleFilesData, oneFileData)

		// Cleanup.
		err = init.Reset()
		r.NoError(err)
	}
}

func TestNumLabelsWritten(t *testing.T) {
	req := require.New(t)

	cfg, opts := getTestConfig(t)
	id := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, id)
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
	req.Equal(uint64(opts.NumUnits*cfg.LabelsPerUnit), numLabelsWritten)

	// Initialize repeated.
	err = init.Initialize()
	req.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(opts.NumUnits*cfg.LabelsPerUnit), numLabelsWritten)

	// Initialize repeated, using a new instance.
	init, err = NewInitializer(cfg, opts, id)
	req.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(opts.NumUnits*cfg.LabelsPerUnit), numLabelsWritten)
	err = init.Initialize()
	req.NoError(err)
	numLabelsWritten, err = init.diskState.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(opts.NumUnits*cfg.LabelsPerUnit), numLabelsWritten)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func TestValidateMetadata(t *testing.T) {
	r := require.New(t)

	cfg, opts := getTestConfig(t)
	id := make([]byte, 32)
	init, err := NewInitializer(cfg, opts, id)
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

	// Attempt to initialize with different `ID`.
	newID := make([]byte, 32)
	newID[0] = newID[0] + 1
	init, err = NewInitializer(cfg, opts, newID)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok := err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("ID", errConfigMismatch.Param)

	// Attempt to initialize with different `cfg.BitsPerLabel`.
	newCfg := cfg
	newCfg.BitsPerLabel = cfg.BitsPerLabel + 1
	init, err = NewInitializer(newCfg, opts, id)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok = err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("BitsPerLabel", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumFiles`.
	newOpts := opts
	newOpts.NumFiles = 4
	init, err = NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok = err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("NumFiles", errConfigMismatch.Param)

	// Attempt to initialize with different `opts.NumUnits` while `opts.NumFiles` > 1.
	newOpts = opts
	newOpts.NumUnits++
	init, err = NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok = err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("NumUnits", errConfigMismatch.Param)

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestStop(t *testing.T) {
	r := require.New(t)
	log := testLogger{t: t}

	cfg, opts := getTestConfig(t)
	opts.NumUnits = 10
	id := make([]byte, 32)

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	init.SetLogger(log)

	// Start initialization and stop it after a short while.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		time.Sleep(2 * time.Second)
		err := init.Stop()
		r.NoError(err)
		wg.Done()
	}()
	var prev uint64
	go func() {
		for p := range init.SessionNumLabelsWrittenChan() {
			r.True(p > prev)
			prev = p
			log.Info("num labels written: %v\n", p)
		}
		r.True(init.Started())
		r.False(init.Completed())
		wg.Done()
	}()
	err = init.Initialize()
	r.Equal(ErrStopped, err)
	wg.Wait()

	// Continue the initialization to completion.
	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func assertNumLabelsWrittenChan(init *Initializer, r *require.Assertions) chan struct{} {
	doneChan := make(chan struct{})
	go func() {
		var prev uint64
		for p := range init.SessionNumLabelsWrittenChan() {
			r.True(p > prev)
			prev = p
			log.Info("num labels written: %v\n", p)
		}
		c, err := init.Completed()
		r.NoError(err)
		r.True(c)
		close(doneChan)
	}()

	return doneChan
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
