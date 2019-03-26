package persistence

import (
	"encoding/hex"
	"github.com/spacemeshos/post-private/util"
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

	reader, err := NewPostLabelsFileReader(id)
	req.NoError(err)

	labelsFromReader := make([]util.Label, 4)
	var idx uint64
	for i := range labelsFromReader {
		idx, labelsFromReader[i], err = reader.Read()
		req.Equal(idx, uint64(i))
		req.NoError(err)
	}
	idx, shouldBeNil, err := reader.Read()
	req.Equal(io.EOF, err)
	req.Nil(shouldBeNil)
	req.Zero(idx)

	err = reader.Close()
	req.NoError(err)

	req.EqualValues(labels, labelsFromReader)
}

func generateIdAndLabels() ([]byte, []util.Label) {
	id, _ := hex.DecodeString("deadbeef")
	labels := []util.Label{
		util.NewLabel(0),
		util.NewLabel(1),
		util.NewLabel(2),
		util.NewLabel(3),
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
