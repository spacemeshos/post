package initialization

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	smlog "github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	id   = make([]byte, 32)
	cfg  config.Config
	opts config.InitOpts

	log   = flag.Bool("log", false, "")
	debug = flag.Bool("debug", false, "")
)

func TestMain(m *testing.M) {
	cfg = config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts = config.DefaultInitOpts()
	opts.DataDir, _ = ioutil.TempDir("", "post-test")
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = CPUProviderID()

	res := m.Run()
	os.Exit(res)
}

func TestCPUProviderExists(t *testing.T) {
	r := require.New(t)

	p := cpuProvider(providers)
	r.Equal("CPU", p.Model)
	r.Equal(gpu.ComputeAPIClassCPU, p.ComputeAPI)
}

func TestInitialize(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

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

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Initialize again using the same config & opts.
	init, err = NewInitializer(cfg, opts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

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

	newOpts := opts
	newOpts.NumFiles = 1

	init, err := NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Increase `opts.NumUnits`.
	newOpts.NumUnits++

	init, err = NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

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

	newOpts := opts
	newOpts.NumUnits++
	newOpts.NumFiles = 1

	init, err := NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	// Decrease `opts.NumUnits`.
	newOpts.NumUnits--

	init, err = NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

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

	newOpts := opts
	newOpts.NumUnits++
	newOpts.NumFiles = 2

	init, err := NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	doneChan := assertNumLabelsWrittenChan(init, r)
	err = init.Initialize()
	r.NoError(err)
	<-doneChan

	prevNumUnits := newOpts.NumUnits

	// Increase `opts.NumUnits` while `opts.NumFiles` > 1.
	newOpts.NumUnits = prevNumUnits + 1
	init, err = NewInitializer(cfg, opts, id)
	r.NoError(err)
	err = init.Initialize()
	errConfigMismatch, ok := err.(ConfigMismatchError)
	r.True(ok)
	r.Equal("NumUnits", errConfigMismatch.Param)

	// Decrease `opts.NumUnits` while `opts.NumFiles` > 1.
	newOpts.NumUnits = prevNumUnits - 1
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

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)

	m, err := init.loadMetadata()
	r.Equal(ErrStateMetadataFileMissing, err)

	err = init.Initialize()
	r.NoError(err)
	m, err = init.loadMetadata()
	r.NoError(err)
	err = init.verifyMetadata(m)
	r.NoError(err)

	// Attempt to initialize with different `ID`.
	newID := make([]byte, 32)
	copy(newID, id)
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
	newOpts.NumFiles++
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

	newOpts := opts
	newOpts.NumUnits = 5

	init, err := NewInitializer(cfg, newOpts, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

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

			if *debug {
				fmt.Printf("num labels written: %v\n", p)
			}
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

			if *debug {
				fmt.Printf("num labels written: %v\n", p)
			}
		}
		r.True(init.Completed())
		close(doneChan)
	}()

	return doneChan
}

func initData(datadir string, bitsPerLabel uint) ([]byte, error) {
	reader, err := persistence.NewLabelsReader(datadir, bitsPerLabel)
	if err != nil {
		return nil, err
	}

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
