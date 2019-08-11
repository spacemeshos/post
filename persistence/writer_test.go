package persistence

import (
	"encoding/binary"
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

type LabelGroup []byte

func TestLabelsReaderAndWriter(t *testing.T) {
	req := require.New(t)
	id, labelGroupGroupGroups := generateIdAndLabels()
	labelsToWriter := make([]LabelGroup, 0)

	for i, labelGroupGroup := range labelGroupGroupGroups {
		writer, err := NewLabelsWriter(tempdir, id, i)
		req.NoError(err)

		for _, labelGroup := range labelGroupGroup {
			err := writer.Write(labelGroup)
			req.NoError(err)

			// For later assertion.
			labelsToWriter = append(labelsToWriter, labelGroup)
		}
		_, err = writer.Close()
		req.NoError(err)
	}

	reader, err := NewLabelsReader(tempdir, id)
	req.NoError(err)

	labelsFromReader := make([]LabelGroup, len(labelsToWriter))
	for i := range labelsFromReader {
		labelsFromReader[i], err = reader.ReadNext()
		req.NoError(err)
	}
	shouldBeNil, err := reader.ReadNext()
	req.Equal(io.EOF, err)
	req.Nil(shouldBeNil)

	req.EqualValues(labelsToWriter, labelsFromReader)
}

func generateIdAndLabels() ([]byte, [][]LabelGroup) {
	id, _ := hex.DecodeString("deadbeef")
	labels := [][]LabelGroup{
		{
			NewLabelGroup(0),
			NewLabelGroup(1),
			NewLabelGroup(2),
			NewLabelGroup(3),
		},
		{
			NewLabelGroup(4),
			NewLabelGroup(5),
			NewLabelGroup(6),
			NewLabelGroup(7),
		},
		{
			NewLabelGroup(8),
			NewLabelGroup(9),
			NewLabelGroup(10),
			NewLabelGroup(11),
		},
		{
			NewLabelGroup(12),
			NewLabelGroup(13),
			NewLabelGroup(14),
			NewLabelGroup(15),
		},
	}
	return id, labels
}

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	res := m.Run()
	cleanup()
	os.Exit(res)
}

func cleanup() {
	_ = os.RemoveAll(tempdir)
}

func NewLabelGroup(cnt uint64) []byte {
	b := make([]byte, LabelGroupSize)
	binary.LittleEndian.PutUint64(b, cnt)
	return b
}
