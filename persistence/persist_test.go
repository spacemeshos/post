package persistence

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"post-private/datatypes"
	"testing"
)

func TestPersistPostLabels(t *testing.T) {
	req := require.New(t)

	id, _ := hex.DecodeString("deadbeef")
	labels := []datatypes.Label{
		datatypes.NewLabel(0),
		datatypes.NewLabel(1),
		datatypes.NewLabel(2),
		datatypes.NewLabel(3),
	}
	PersistPostLabels(id, labels)

	f, err := os.Open(filepath.Join("data", "deadbeef", "all.labels"))
	req.NoError(err)
	defer f.Close()

	req.Equal(datatypes.NewLabel(0), datatypes.Label(readEightBytes(req, f)))
	req.Equal(datatypes.NewLabel(1), datatypes.Label(readEightBytes(req, f)))
	req.Equal(datatypes.NewLabel(2), datatypes.Label(readEightBytes(req, f)))
	req.Equal(datatypes.NewLabel(3), datatypes.Label(readEightBytes(req, f)))
}

func readEightBytes(req *require.Assertions, f *os.File) []byte {
	b := make([]byte, datatypes.LabelSize)
	n, err := f.Read(b)
	req.NoError(err)
	req.Equal(datatypes.LabelSize, n)
	return b
}

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	res := m.Run()
	cleanup()
	os.Exit(res)
}

func cleanup() {
	_ = os.RemoveAll(filepath.Join("data", "deadbeef"))
}
