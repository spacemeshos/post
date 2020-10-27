package persistence

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestFileWriter_Width(t *testing.T) {
	req := require.New(t)

	labelSize := uint(1)
	index := 0
	datadir, _ := ioutil.TempDir("", "post-test")

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
	info, err := writer.Close()
	req.NoError(err)
	req.Equal(int64(2), (*info).Size())

	writer, err = NewLabelsWriter(datadir, index, labelSize)
	req.NoError(err)
	width, err = writer.NumLabelsWritten()
	req.NoError(err)
	req.Equal(uint64(16), width)

	_ = os.RemoveAll(datadir)
}
