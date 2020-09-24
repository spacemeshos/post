package persistence

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

var (
	tempdir, _ = ioutil.TempDir("", "post-test")
)

type Label []byte

func TestMain(m *testing.M) {
	res := m.Run()
	_ = os.RemoveAll(tempdir)
	os.Exit(res)
}

func TestLabelsReaderAndWriter(t *testing.T) {
	req := require.New(t)
	id, labelGroups := generateIdAndLabels()
	labelsToWriter := make([]Label, 0)

	for i, labelGroup := range labelGroups {
		writer, err := NewLabelsWriter(tempdir, id, i, labelSize*8)
		req.NoError(err)

		for _, label := range labelGroup {
			err := writer.Write(label)
			req.NoError(err)

			// For later assertion.
			labelsToWriter = append(labelsToWriter, label)
		}
		_, err = writer.Close()
		req.NoError(err)
	}

	reader, err := NewLabelsReader(tempdir, id, labelSize*8)
	req.NoError(err)

	labelsFromReader := make([]Label, len(labelsToWriter))
	for i := range labelsFromReader {
		labelsFromReader[i], err = reader.ReadNext()
		req.NoError(err)
	}
	shouldBeNil, err := reader.ReadNext()
	req.Equal(io.EOF, err)
	req.Nil(shouldBeNil)

	req.EqualValues(labelsToWriter, labelsFromReader)
}

func generateIdAndLabels() ([]byte, [][]Label) {
	id, _ := hex.DecodeString("deadbeef")
	labels := [][]Label{
		{
			NewLabelFromUint64(0, labelSize),
			NewLabelFromUint64(1, labelSize),
			NewLabelFromUint64(2, labelSize),
			NewLabelFromUint64(3, labelSize),
		},
		{
			NewLabelFromUint64(4, labelSize),
			NewLabelFromUint64(5, labelSize),
			NewLabelFromUint64(6, labelSize),
			NewLabelFromUint64(7, labelSize),
		},
		{
			NewLabelFromUint64(8, labelSize),
			NewLabelFromUint64(9, labelSize),
			NewLabelFromUint64(10, labelSize),
			NewLabelFromUint64(11, labelSize),
		},
		{
			NewLabelFromUint64(12, labelSize),
			NewLabelFromUint64(13, labelSize),
			NewLabelFromUint64(14, labelSize),
			NewLabelFromUint64(15, labelSize),
		},
	}
	return id, labels
}
