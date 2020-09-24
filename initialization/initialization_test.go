package initialization

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var (
	challenge = shared.ZeroChallenge
	id        = make([]byte, 32)
	cfg       *Config
)

func TestMain(m *testing.M) {
	cfg = config.DefaultConfig()
	cfg.DataDir, _ = ioutil.TempDir("", "post-test")

	flag.StringVar(&cfg.DataDir, "datadir", cfg.DataDir, "")
	flag.Uint64Var(&cfg.NumLabels, "numlabels", cfg.NumLabels, "")
	flag.UintVar(&cfg.LabelSize, "labelsize", cfg.LabelSize, "")
	flag.UintVar(&cfg.NumFiles, "numfiles", cfg.NumFiles, "")
	flag.Parse()

	res := m.Run()
	os.Exit(res)
}

func TestInitialize(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 15
	cfg.LabelSize = 8

	init, err := NewInitializer(&cfg, id)
	r.NoError(err)

	go func() {
		var prev float64
		for p := range init.Progress() {
			r.True(p > prev)
			prev = p
		}
		r.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	r.NoError(err)

	err = init.Reset()
	r.NoError(err)
}

func TestStop(t *testing.T) {
	r := require.New(t)
	//	t.Skip()

	cfg := *cfg
	cfg.NumLabels = 1 << 15
	cfg.LabelSize = 8

	init, err := NewInitializer(&cfg, id)
	r.NoError(err)
	init.SetLogger(log.AppLog)

	go func() {
		var prev float64
		for p := range init.Progress() {
			r.True(p > prev)
			prev = p
		}
		r.Equal(float64(1), prev)
	}()
	go func() {
		time.Sleep(1 * time.Second)
		err := init.Stop()
		r.NoError(err)
	}()
	err = init.Initialize(CPUProviderID())
	r.Equal(ErrStopped, err)

	go func() {
		var prev float64
		for p := range init.Progress() {
			r.True(p > prev)
			prev = p
		}
		r.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	r.NoError(err)
	err = init.Reset()
	r.NoError(err)
}

func TestInitializerMultipleFiles(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.NumLabels = (1 << 10) - 16
	cfg.LabelSize = 8
	cfg.NumFiles = 1

	init, err := NewInitializer(&cfg, id)
	r.NoError(err)
	go func() {
		for p := range init.Progress() {
			fmt.Printf("%v\n", p)
		}
	}()
	err = init.Initialize(CPUProviderID())
	r.NoError(err)
	oneFileData, err := initData(cfg.DataDir, id, cfg.LabelSize)
	r.NoError(err)
	err = init.Reset()
	r.NoError(err)

	for numFiles := uint(2); numFiles <= 16; numFiles <<= 1 {
		cfg := cfg
		cfg.NumFiles = numFiles

		init, err := NewInitializer(&cfg, id)
		r.NoError(err)
		go func() {
			var prev float64
			for p := range init.Progress() {
				r.True(p > prev)
				prev = p
			}
			r.Equal(float64(1), prev)
		}()
		err = init.Initialize(CPUProviderID())
		r.NoError(err)

		multipleFilesData, err := initData(cfg.DataDir, id, cfg.LabelSize)
		r.NoError(err)
		r.Equal(oneFileData, multipleFilesData)

		err = init.Reset()
		r.NoError(err)
	}
}

func TestDiskState(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 15
	cfg.NumFiles = 2

	init, err := NewInitializer(&cfg, id)
	r.NoError(err)

	diskState, err := init.DiskState()
	r.NoError(err)
	r.Equal(InitStateNotStarted, diskState.InitState)
	r.Equal(uint64(0), diskState.BytesWritten)

	err = init.Initialize(CPUProviderID())
	r.NoError(err)

	diskState, err = init.DiskState()
	r.NoError(err)
	r.Equal(InitStateCompleted, diskState.InitState)
	r.Equal(shared.DataSize(cfg.NumLabels, cfg.LabelSize), diskState.BytesWritten)

	err = init.Initialize(CPUProviderID())
	r.Equal(err, shared.ErrInitCompleted)

	// Initialize using a new instance.

	init, err = NewInitializer(&cfg, id)
	r.NoError(err)

	diskState, err = init.DiskState()
	r.NoError(err)
	r.Equal(InitStateCompleted, diskState.InitState)
	r.Equal(shared.DataSize(cfg.NumLabels, cfg.LabelSize), diskState.BytesWritten)

	err = init.Initialize(CPUProviderID())
	r.Equal(err, shared.ErrInitCompleted)

	// Use a new instance with a different id.

	newID := make([]byte, 32)
	copy(newID, id)
	newID[0] = newID[0] + 1
	init, err = NewInitializer(&cfg, newID)
	r.NoError(err)

	_, err = init.DiskState()
	errConfigMismatch, ok := err.(configMismatchError)
	r.True(ok)
	r.Equal("id", errConfigMismatch.param)

	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("id", errConfigMismatch.param)

	err = init.Reset()
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("id", errConfigMismatch.param)

	// Use a new instance with a different LabelSize config.

	newCfg := cfg
	newCfg.LabelSize = cfg.LabelSize + 1

	init, err = NewInitializer(&newCfg, id)
	r.NoError(err)

	_, err = init.DiskState()
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("LabelSize", errConfigMismatch.param)

	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("LabelSize", errConfigMismatch.param)

	err = init.Reset()
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("LabelSize", errConfigMismatch.param)

	// Use a new instance with a different NumFiles config.

	newCfg = cfg
	newCfg.NumFiles = cfg.NumFiles * 2

	init, err = NewInitializer(&newCfg, id)
	r.NoError(err)

	_, err = init.DiskState()
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("NumFiles", errConfigMismatch.param)

	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("NumFiles", errConfigMismatch.param)

	err = init.Reset()
	errConfigMismatch, ok = err.(configMismatchError)
	r.True(ok)
	r.Equal("NumFiles", errConfigMismatch.param)

	// Reset with the correct config.

	init, err = NewInitializer(&cfg, id)
	r.NoError(err)
	err = init.Reset()
	r.NoError(err)
}

func BenchmarkInitializeGeneric(b *testing.B) {
	// Use cli flags (TestMain) to utilize this test.
	init, err := NewInitializer(cfg, id)
	require.NoError(b, err)
	init.SetLogger(log.AppLog)
	err = init.Initialize(CPUProviderID())
	require.NoError(b, err)
}

func initData(datadir string, id []byte, labelSize uint) ([]byte, error) {
	reader, err := persistence.NewLabelsReader(datadir, id, labelSize)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	for {
		b, err := reader.ReadNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		buf.Write(b)
	}

	return buf.Bytes(), nil
}
