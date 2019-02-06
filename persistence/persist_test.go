package persistence

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"post-private/datatypes"
	"testing"
)

func TestPostLabelsReaderAndWriter(t *testing.T) {
	req := require.New(t)
	id, labels := generateIdAndLabels()

	writer, err := NewPostLabelsWriter(id)
	req.NoError(err)

	for _, l := range labels {
		err := writer.Write(l)
		req.NoError(err)
	}
	err = writer.Close()
	req.NoError(err)

	reader, err := NewPostLabelsReader(id)
	req.NoError(err)

	labelsFromReader := make([]datatypes.Label, 4)
	for i := range labelsFromReader {
		labelsFromReader[i], err = reader.Read()
		req.NoError(err)
	}
	shouldBeNil, err := reader.Read()
	req.Nil(shouldBeNil)
	req.Equal(err, io.EOF)

	req.EqualValues(labels, labelsFromReader)
}

func generateIdAndLabels() ([]byte, []datatypes.Label) {
	id, _ := hex.DecodeString("deadbeef")
	labels := []datatypes.Label{
		datatypes.NewLabel(0),
		datatypes.NewLabel(1),
		datatypes.NewLabel(2),
		datatypes.NewLabel(3),
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
