package persistence

import (
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

const labelSize = uint(64)

type Label []byte

func TestLabelsReaderAndWriter(t *testing.T) {
	req := require.New(t)

	id := make([]byte, 32)
	labelGroups := genLabelGroups(labelSize)
	writtenLabels := make([]Label, 0)
	datadir, _ := ioutil.TempDir("", "post-test")

	for i, labelGroup := range labelGroups {
		writer, err := NewLabelsWriter(datadir, id, i, labelSize)
		req.NoError(err)

		for _, label := range labelGroup {
			err := writer.Append(label)
			req.NoError(err)

			// For later assertion.
			writtenLabels = append(writtenLabels, label)
		}
		_, err = writer.Close()
		req.NoError(err)
	}

	reader, err := NewLabelsReader(datadir, id, labelSize)
	req.NoError(err)

	readLabels := make([]Label, len(writtenLabels))
	for i := range readLabels {
		readLabels[i], err = reader.ReadNext()
		req.NoError(err)
	}
	shouldBeNil, err := reader.ReadNext()
	req.Equal(io.EOF, err)
	req.Nil(shouldBeNil)

	req.EqualValues(writtenLabels, readLabels)

	_ = os.RemoveAll(datadir)
}

func genLabelGroups(labelSize uint) [][]Label {
	return [][]Label{
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
}
