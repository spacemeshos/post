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

type Label []byte

func TestLabelsReaderAndWriter(t *testing.T) {
	req := require.New(t)
	id, labelGroupGroupGroups := generateIdAndLabels()
	labelsToWriter := make([]Label, 0)
	size := uint(32)

	for i, labelGroupGroup := range labelGroupGroupGroups {
		writer, err := NewLabelsWriter(tempdir, id, i, size)
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

	reader, err := NewLabelsReader(tempdir, id, size)
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
			NewLabel(0),
			NewLabel(1),
			NewLabel(2),
			NewLabel(3),
		},
		{
			NewLabel(4),
			NewLabel(5),
			NewLabel(6),
			NewLabel(7),
		},
		{
			NewLabel(8),
			NewLabel(9),
			NewLabel(10),
			NewLabel(11),
		},
		{
			NewLabel(12),
			NewLabel(13),
			NewLabel(14),
			NewLabel(15),
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

func NewLabel(cnt uint64) []byte {
	b := make([]byte, 32)
	binary.LittleEndian.PutUint64(b, cnt)
	return b
}
