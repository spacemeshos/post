package initialization

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	smlog "github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var (
	id  = make([]byte, 32)
	cfg = config.DefaultConfig()

	log   = flag.Bool("log", false, "")
	debug = flag.Bool("debug", false, "")
)

func TestMain(m *testing.M) {
	cfg.DataDir, _ = ioutil.TempDir("", "post-test")

	res := m.Run()
	os.Exit(res)
}

func TestInitialize(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.NumFiles = 2
	cfg.LabelSize = 8

	init, err := NewInitializer(&cfg, id)
	r.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	go func() {
		var prev float64
		for p := range init.SessionNumLabelsWrittenChan() {
			r.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		r.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	r.NoError(err)

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestInitialize_Repeated(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.NumFiles = 2
	cfg.ComputeBatchSize = 1 << 10
	cfg.LabelSize = 8

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	go func() {
		var prev float64
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	// Initialize again using the same config.
	init, err = NewInitializer(&cfg, id)
	req.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	go func() {
		var prev float64
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func TestInitialize_AlterNumLabels_Increase(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.NumFiles = 1
	cfg.ComputeBatchSize = 1 << 10
	cfg.LabelSize = 8

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	go func() {
		var prev float64
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	// Increase the number of labels.
	cfg.NumLabels = cfg.NumLabels << 1

	init, err = NewInitializer(&cfg, id)
	req.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	go func() {
		var prev float64
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func TestInitialize_AlterNumLabels_Decrease(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.NumFiles = 1
	cfg.ComputeBatchSize = 1 << 10
	cfg.LabelSize = 8

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	go func() {
		var prev float64
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	// Decrease the number of labels.
	cfg.NumLabels = cfg.NumLabels >> 1

	init, err = NewInitializer(&cfg, id)
	req.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	go func() {
		var prev float64
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func TestInitialize_AlterNumLabels_MultipleFiles(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.NumFiles = 2

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	prevNumLabels := cfg.NumLabels

	// Increase the number of labels.
	cfg.NumLabels = prevNumLabels << 1
	init, err = NewInitializer(&cfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok := err.(ConfigMismatchError)
	req.True(ok)
	req.Equal("NumLabels", errConfigMismatch.Param)

	// Decrease the number of labels.
	cfg.NumLabels = prevNumLabels >> 1
	init, err = NewInitializer(&cfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok = err.(ConfigMismatchError)
	req.True(ok)
	req.Equal("NumLabels", errConfigMismatch.Param)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func TestInitialize_MultipleFiles(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.LabelSize = 8
	cfg.NumFiles = 1

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	req.NoError(err)
	oneFileData, err := initData(cfg.DataDir, cfg.LabelSize)
	req.NoError(err)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)

	for numFiles := uint(2); numFiles <= 16; numFiles <<= 1 {
		cfg := cfg
		cfg.NumFiles = numFiles

		init, err := NewInitializer(&cfg, id)
		req.NoError(err)
		err = init.Initialize(CPUProviderID())
		req.NoError(err)
		multipleFilesData, err := initData(cfg.DataDir, cfg.LabelSize)
		req.NoError(err)

		req.Equal(multipleFilesData, oneFileData)

		// Cleanup.
		err = init.Reset()
		req.NoError(err)
	}
}

func TestNumLabelsWritten(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.NumFiles = 2

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)

	// Check initial state.
	numLabelsWritten, err := init.DiskNumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(0), numLabelsWritten)

	// Initialize.
	err = init.Initialize(CPUProviderID())
	req.NoError(err)
	numLabelsWritten, err = init.DiskNumLabelsWritten()
	req.NoError(err)
	req.Equal(cfg.NumLabels, numLabelsWritten)

	// Initialize repeated.
	err = init.Initialize(CPUProviderID())
	req.NoError(err)
	numLabelsWritten, err = init.DiskNumLabelsWritten()
	req.NoError(err)
	req.Equal(cfg.NumLabels, numLabelsWritten)

	// Initialize repeated, using a new instance.
	init, err = NewInitializer(&cfg, id)
	req.NoError(err)
	numLabelsWritten, err = init.DiskNumLabelsWritten()
	req.NoError(err)
	req.Equal(cfg.NumLabels, numLabelsWritten)
	err = init.Initialize(CPUProviderID())
	req.NoError(err)
	numLabelsWritten, err = init.DiskNumLabelsWritten()
	req.NoError(err)
	req.Equal(cfg.NumLabels, numLabelsWritten)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func TestValidateMetadata(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 12
	cfg.NumFiles = 2

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)

	err = init.VerifyMetadata()
	req.Equal(ErrStateMetadataFileMissing, err)

	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	err = init.VerifyMetadata()
	req.NoError(err)

	// Attempt to initialize with different `id`.
	newID := make([]byte, 32)
	copy(newID, id)
	newID[0] = newID[0] + 1
	init, err = NewInitializer(&cfg, newID)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok := err.(ConfigMismatchError)
	req.True(ok)
	req.Equal("ID", errConfigMismatch.Param)

	// Attempt to initialize with different `labelSize`.
	newCfg := cfg
	newCfg.LabelSize = cfg.LabelSize + 1
	init, err = NewInitializer(&newCfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok = err.(ConfigMismatchError)
	req.True(ok)
	req.Equal("LabelSize", errConfigMismatch.Param)

	// Attempt to initialize with different `numFiles`.
	newCfg = cfg
	newCfg.NumFiles = cfg.NumFiles << 1
	init, err = NewInitializer(&newCfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok = err.(ConfigMismatchError)
	req.True(ok)
	req.Equal("NumFiles", errConfigMismatch.Param)

	// Attempt to initialize with different `numLabels` while `numFiles` > 1.
	newCfg = cfg
	newCfg.NumLabels = newCfg.NumLabels << 1
	init, err = NewInitializer(&newCfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID())
	errConfigMismatch, ok = err.(ConfigMismatchError)
	req.True(ok)
	req.Equal("NumLabels", errConfigMismatch.Param)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func TestStop(t *testing.T) {
	req := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 15
	cfg.LabelSize = 8

	init, err := NewInitializer(&cfg, id)
	req.NoError(err)
	if *log {
		init.SetLogger(smlog.AppLog)
	}

	// Start initialization and stop it after a short while.
	go func() {
		time.Sleep(1 * time.Second)
		err := init.Stop()
		req.NoError(err)
	}()
	var prev float64
	go func() {
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.True(float64(1) > prev)
		req.True(float64(0) < prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.Equal(ErrStopped, err)

	// Continue the initialization to completion.
	go func() {
		for p := range init.SessionNumLabelsWrittenChan() {
			req.True(p > prev)
			prev = p

			if *debug {
				fmt.Printf("progress: %v\n", p)
			}
		}
		req.Equal(float64(1), prev)
	}()
	err = init.Initialize(CPUProviderID())
	req.NoError(err)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

func initData(datadir string, labelSize uint) ([]byte, error) {
	reader, err := persistence.NewLabelsReader(datadir, labelSize)
	if err != nil {
		return nil, err
	}

	gsReader := shared.NewGranSpecificReader(reader, labelSize)
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
