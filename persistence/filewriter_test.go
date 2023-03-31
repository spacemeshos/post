package persistence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileWriter_Width(t *testing.T) {
	req := require.New(t)

	labelSize := uint(1)
	index := 0
	datadir := t.TempDir()

	writer, err := NewLabelsWriter(datadir, index, labelSize)
	req.NoError(err)
	width, err := writer.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(0), width)

	// Write 2 bytes (16 labels, 1 bit each)
	err = writer.Write([]byte{0xFF, 0xFF})
	req.NoError(err)
	width, err = writer.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(0), width)

	err = writer.Flush()
	req.NoError(err)
	width, err = writer.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(16), width)
	req.NoError(writer.Close())

	writer, err = NewLabelsWriter(datadir, index, labelSize)
	req.NoError(err)
	width, err = writer.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(16), width)
	req.NoError(writer.Close())
}
