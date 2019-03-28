package persistence

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestPostLabelsReaderAndWriter(t *testing.T) {
	req := require.New(t)
	id, labels := generateIdAndLabels()

	writer, err := NewPostLabelsFileWriter(id)
	req.NoError(err)

	for _, l := range labels {
		err := writer.Write(l)
		req.NoError(err)
	}
	err = writer.Close()
	req.NoError(err)

	reader, err := NewLeafReader(id)
	req.NoError(err)

	labelsFromReader := make([]Label, 4)
	for i := range labelsFromReader {
		labelsFromReader[i], err = reader.ReadNext()
		req.NoError(err)
	}
	shouldBeNil, err := reader.ReadNext()
	req.Equal(io.EOF, err)
	req.Nil(shouldBeNil)

	err = reader.Close()
	req.NoError(err)

	req.EqualValues(labels, labelsFromReader)
}

func generateIdAndLabels() ([]byte, []Label) {
	id, _ := hex.DecodeString("deadbeef")
	labels := []Label{
		NewLabel(0),
		NewLabel(1),
		NewLabel(2),
		NewLabel(3),
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
	_ = os.RemoveAll(filepath.Join(GetPostDataPath(), "deadbeef"))
}

func NewLabel(cnt uint64) []byte {
	b := make([]byte, LabelSize)
	binary.LittleEndian.PutUint64(b, cnt)
	return b
}
