package persistence

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestFileReader_ReadNext_BitGranular(t *testing.T) {
	req := require.New(t)

	labelSize := uint(1)
	id := make([]byte, 32)
	datadir, _ := ioutil.TempDir("", "post-test")

	writer, err := NewLabelsWriter(datadir, make([]byte, 32), 0, labelSize)
	req.NoError(err)
	err = writer.Append([]byte{0xFF})
	req.NoError(err)
	_, err = writer.Close()
	req.NoError(err)

	reader, err := NewLabelsReader(datadir, id, labelSize)
	req.NoError(err)
	label, err := reader.ReadNext()
	req.NoError(err)
	req.Len(label, 1)
	req.Equal(byte(0x01), label[0])

	_ = os.RemoveAll(datadir)

}

func TestFileReader_ReadNext_ByteGranular(t *testing.T) {
	req := require.New(t)

	labelSize := uint(16)
	id := make([]byte, 32)
	datadir, _ := ioutil.TempDir("", "post-test")

	writer, err := NewLabelsWriter(datadir, make([]byte, 32), 0, labelSize)
	req.NoError(err)
	err = writer.Append([]byte{0xFF, 0xFF})
	req.NoError(err)
	_, err = writer.Close()
	req.NoError(err)

	reader, err := NewLabelsReader(datadir, id, labelSize)
	req.NoError(err)
	label, err := reader.ReadNext()
	req.NoError(err)
	req.Len(label, 2)
	req.Equal([]byte{0xFF, 0xFF}, label)

	_ = os.RemoveAll(datadir)
}
