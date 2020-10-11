package persistence

import (
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/oracle"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestFileWriter_Width(t *testing.T) {
	req := require.New(t)

	labelSize := uint(1)
	id := make([]byte, 32)
	index := 0
	datadir, _ := ioutil.TempDir("", "post-test")

	writer, err := NewLabelsWriter(datadir, id, index, labelSize)
	req.NoError(err)
	width, err := writer.Width()
	req.NoError(err)
	req.Equal(uint64(0), width)

	// Write 2 bytes (16 labels, 1 bit each)
	err = writer.Append([]byte{0xFF, 0xFF})
	req.NoError(err)
	width, err = writer.Width()
	req.NoError(err)
	req.Equal(uint64(0), width)

	err = writer.Flush()
	req.NoError(err)
	width, err = writer.Width()
	req.NoError(err)
	req.Equal(uint64(16), width)
	info, err := writer.Close()
	req.NoError(err)
	req.Equal(int64(2), info.Size())

	writer, err = NewLabelsWriter(datadir, id, index, labelSize)
	req.NoError(err)
	width, err = writer.Width()
	req.NoError(err)
	req.Equal(uint64(16), width)

	_ = os.RemoveAll(datadir)
}

// TestLabelCorrectness tests, for variation of label sizes, the correctness of
// reading labels from disk (written in multiple files).
func TestLabelCorrectness(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	req := require.New(t)

	numFiles := 2
	numFileBatches := 2
	batchSize := 256
	id := make([]byte, 32)
	datadir, _ := ioutil.TempDir("", "post-test")

	for labelSize := uint32(config.MinLabelSize); labelSize <= config.MaxLabelSize; labelSize++ {
		// Write labels to files.
		for i := 0; i < numFiles; i++ {
			writer, err := NewLabelsWriter(datadir, id, i, uint(labelSize))
			req.NoError(err)
			for j := 0; j < numFileBatches; j++ {
				numBatch := i*numFileBatches + j
				startPosition := uint64(numBatch * batchSize)
				endPosition := startPosition + uint64(batchSize) - 1

				labels, err := oracle.WorkOracle(2, id, startPosition, endPosition, labelSize)
				req.NoError(err)
				err = writer.Append(labels)
				req.NoError(err)

			}
			_, err = writer.Close()
			req.NoError(err)
		}

		// Read labels from files and compare each to a label compute.
		reader, err := NewLabelsReader(datadir, id, uint(labelSize))
		req.NoError(err)
		var position uint64
		for {
			label, err := reader.ReadNext()
			if err != nil {
				if err == io.EOF {
					req.Equal(uint64(numFiles*numFileBatches*batchSize), position)
					break
				}
				req.Fail(err.Error())
			}

			labelCompute := oracle.WorkOracleOne(2, id, position, labelSize)
			req.Equal(labelCompute, label, fmt.Sprintf("position: %v, labelSize: %v", position, labelSize))

			position++
		}
		_ = os.RemoveAll(datadir)
	}
}
