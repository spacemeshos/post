package initialization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxFileSize(t *testing.T) {
	r := require.New(t)

	cfg := Config{
		LabelsPerUnit: 128,
	}
	opts := InitOpts{
		NumUnits:    10,
		MaxFileSize: 2048,
	}

	layout, err := deriveFilesLayout(cfg, opts)
	r.NoError(err)
	r.Equal(0, layout.FirstFileIdx)
	r.Equal(9, layout.LastFileIdx)
	r.Equal(128, int(layout.FileNumLabels))
	r.Equal(128, int(layout.LastFileNumLabels))

	opts.MaxFileSize = 2000

	layout, err = deriveFilesLayout(cfg, opts)
	r.NoError(err)
	r.Equal(10*128, 10*125+30)
	r.Equal(0, layout.FirstFileIdx)
	r.Equal(10, layout.LastFileIdx)
	r.Equal(125, int(layout.FileNumLabels))
	r.Equal(30, int(layout.LastFileNumLabels))
}

func TestCustomFrom(t *testing.T) {
	r := require.New(t)

	cfg := Config{
		LabelsPerUnit: 128,
	}
	opts := InitOpts{
		NumUnits:    100,
		MaxFileSize: 2048,
		FromFileIdx: 1,
	}

	layout, err := deriveFilesLayout(cfg, opts)
	r.NoError(err)
	r.Equal(1, layout.FirstFileIdx) // should skip the first file
	r.Equal(99, layout.LastFileIdx)
	r.Equal(128, int(layout.FileNumLabels))
	r.Equal(128, int(layout.LastFileNumLabels))
}

func TestCustomTo(t *testing.T) {
	r := require.New(t)

	cfg := Config{
		LabelsPerUnit: 128,
	}
	to := 2
	opts := InitOpts{
		NumUnits:    100,
		MaxFileSize: 2048,
		ToFileIdx:   &to,
	}

	layout, err := deriveFilesLayout(cfg, opts)
	r.NoError(err)
	r.Equal(0, layout.FirstFileIdx)
	r.Equal(2, layout.LastFileIdx)
	r.Equal(128, int(layout.FileNumLabels))
	r.Equal(128, int(layout.LastFileNumLabels))
}

func TestCustomFromAndTo(t *testing.T) {
	r := require.New(t)

	cfg := Config{
		LabelsPerUnit: 128,
	}
	to := 2
	opts := InitOpts{
		NumUnits:    100,
		MaxFileSize: 2048,
		FromFileIdx: 1,
		ToFileIdx:   &to,
	}

	layout, err := deriveFilesLayout(cfg, opts)
	r.NoError(err)
	r.Equal(1, layout.FirstFileIdx)
	r.Equal(2, layout.LastFileIdx)
	r.Equal(128, int(layout.FileNumLabels))
	r.Equal(128, int(layout.LastFileNumLabels))
}

func TestInvalidRange(t *testing.T) {
	cfg := Config{
		LabelsPerUnit: 128,
	}
	to := 0
	opts := InitOpts{
		NumUnits:    100,
		MaxFileSize: 2048,
		FromFileIdx: 1,
		ToFileIdx:   &to,
	}

	_, err := deriveFilesLayout(cfg, opts)
	require.Error(t, err)
}

func TestToCannotBeNegative(t *testing.T) {
	cfg := Config{
		LabelsPerUnit: 128,
	}
	to := -1
	opts := InitOpts{
		NumUnits:    100,
		MaxFileSize: 2048,
		ToFileIdx:   &to,
	}

	_, err := deriveFilesLayout(cfg, opts)
	require.Error(t, err)
}

func TestToOutOfRange(t *testing.T) {
	cfg := Config{
		LabelsPerUnit: 128,
	}
	to := 1000000
	opts := InitOpts{
		NumUnits:    100,
		MaxFileSize: 2048,
		ToFileIdx:   &to,
	}

	_, err := deriveFilesLayout(cfg, opts)
	require.Error(t, err)
}

func TestCustomToPartialLastFile(t *testing.T) {
	r := require.New(t)

	cfg := Config{
		LabelsPerUnit: 128,
	}
	to := 49
	opts := InitOpts{
		MaxFileSize: 4096, // 2 units per file
		NumUnits:    99,   // last file will be partial
		ToFileIdx:   &to,
	}

	layout, err := deriveFilesLayout(cfg, opts)
	r.NoError(err)
	r.Equal(0, layout.FirstFileIdx)
	r.Equal(49, layout.LastFileIdx)
	r.Equal(256, int(layout.FileNumLabels))
	r.Equal(128, int(layout.LastFileNumLabels))
}
