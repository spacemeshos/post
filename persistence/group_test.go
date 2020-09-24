package persistence

import (
	"encoding/binary"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

const labelSize = uint(32)

func TestGroupLayers(t *testing.T) {
	r := require.New(t)

	// Create 9 labels.
	labels := genLabels(9, labelSize)

	// Split the labels into 3 separate files.
	files := make([]ReadWriter, 3)
	files[0] = newSliceReadWriter(labelSize)
	_, _ = files[0].Append(labels[0])
	_, _ = files[0].Append(labels[1])
	_, _ = files[0].Append(labels[2])
	files[1] = newSliceReadWriter(labelSize)
	_, _ = files[1].Append(labels[3])
	_, _ = files[1].Append(labels[4])
	_, _ = files[1].Append(labels[5])
	files[2] = newSliceReadWriter(labelSize)
	_, _ = files[2].Append(labels[6])
	_, _ = files[2].Append(labels[7])
	_, _ = files[2].Append(labels[8])

	// Group the files.
	readwriter, err := Group(files)
	r.NoError(err)

	width, err := readwriter.Width()
	r.NoError(err)
	r.Equal(width, uint64(len(labels)))

	for _, label := range labels {
		val, err := readwriter.ReadNext()
		r.NoError(err)
		r.Equal(val, label)
	}

	// Verify EOF.
	val, err := readwriter.ReadNext()
	r.Equal(err, io.EOF)
	r.Nil(val)

	// Reset the group position, and iterate once again.
	// This verifies that deactivated-chunks position is being reset.
	err = readwriter.Seek(0)
	for _, label := range labels {
		val, err := readwriter.ReadNext()
		r.NoError(err)
		r.Equal(val, label)
	}

	// Iterate over the group layer with Seek.
	for i, label := range labels {
		err := readwriter.Seek(uint64(i))
		r.NoError(err)
		val, err := readwriter.ReadNext()
		r.NoError(err)
		r.Equal(val, label)
	}
	_, err = readwriter.ReadNext()
	r.Equal(err, io.EOF)

	// Iterate over the group layer with Seek in reverse.
	for i := len(labels) - 1; i >= 0; i-- {
		err := readwriter.Seek(uint64(i))
		r.NoError(err)
		val, err := readwriter.ReadNext()
		r.NoError(err)
		r.Equal(val, labels[i])
	}
	err = readwriter.Seek(0)
	r.NoError(err)

	err = readwriter.Close()
	r.NoError(err)
}

func TestGroupLayersWithShorterLastLayer(t *testing.T) {
	r := require.New(t)

	// Create 7 labels.
	labels := genLabels(7, labelSize)

	// Split the labels into 3 separate chunks in groups of [3,3,1].
	chunks := make([]ReadWriter, 3)
	chunks[0] = newSliceReadWriter(labelSize)
	_, _ = chunks[0].Append(labels[0])
	_, _ = chunks[0].Append(labels[1])
	_, _ = chunks[0].Append(labels[2])
	chunks[1] = newSliceReadWriter(labelSize)
	_, _ = chunks[1].Append(labels[3])
	_, _ = chunks[1].Append(labels[4])
	_, _ = chunks[1].Append(labels[5])
	chunks[2] = newSliceReadWriter(labelSize)
	_, _ = chunks[2].Append(labels[6])

	// Group the chunks.
	layer, err := Group(chunks)
	r.NoError(err)

	width, err := layer.Width()
	r.NoError(err)
	r.Equal(width, uint64(len(labels)))

	// Iterate over the layer.
	for _, label := range labels {
		val, err := layer.ReadNext()
		r.NoError(err)
		r.Equal(val, label)
	}

	// Arrive to EOF with ReadNext.
	err = layer.Seek(uint64(6))
	r.NoError(err)
	val, err := layer.ReadNext()
	r.NoError(err)
	r.Equal(val, labels[6])
	val, err = layer.ReadNext()
	r.Equal(io.EOF, err)

	// Arrive to EOF with Seek.
	err = layer.Seek(uint64(7))
	r.Equal(io.EOF, err)
	err = layer.Seek(uint64(666))
	r.Equal(io.EOF, err)
}

func TestGroupLayersWithShorterMidLayer(t *testing.T) {
	r := require.New(t)

	// Create 7 labels.
	labels := genLabels(7, labelSize)

	// Split the labels into 3 separate chunks in groups of [3,1,3].
	chunks := make([]ReadWriter, 3)
	chunks[0] = &sliceReadWriter{}
	_, _ = chunks[0].Append(labels[0])
	_, _ = chunks[0].Append(labels[1])
	_, _ = chunks[0].Append(labels[2])
	chunks[1] = &sliceReadWriter{}
	_, _ = chunks[1].Append(labels[3])
	chunks[2] = &sliceReadWriter{}
	_, _ = chunks[2].Append(labels[4])
	_, _ = chunks[2].Append(labels[5])
	_, _ = chunks[2].Append(labels[6])

	// Group the chunks.
	_, err := Group(chunks)
	r.Equal("chunks width mismatch", err.Error())
}

func genLabels(num int, labelSize uint) [][]byte {
	labels := make([][]byte, num)
	for i := 0; i < num; i++ {
		labels[i] = NewLabelFromUint64(uint64(i), labelSize)
	}
	return labels
}

func NewLabelFromUint64(i uint64, labelSize uint) []byte {
	b := make([]byte, labelSize)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

type sliceReadWriter struct {
	slice    [][]byte
	position uint64
	itemSize uint
}

// A compile time check to ensure that sliceReadWriter fully implements ReadWriter.
var _ ReadWriter = (*sliceReadWriter)(nil)

func newSliceReadWriter(itemSize uint) *sliceReadWriter {
	return &sliceReadWriter{itemSize: itemSize}
}

func (s *sliceReadWriter) Width() (uint64, error) {
	return uint64(len(s.slice)), nil
}

func (s *sliceReadWriter) Seek(index uint64) error {
	if index >= uint64(len(s.slice)) {
		return io.EOF
	}
	s.position = index
	return nil
}

func (s *sliceReadWriter) ReadNext() ([]byte, error) {
	if s.position >= uint64(len(s.slice)) {
		return nil, io.EOF
	}
	value := make([]byte, s.itemSize)
	copy(value, s.slice[s.position])
	s.position++
	return value, nil
}

func (s *sliceReadWriter) Append(p []byte) (n int, err error) {
	s.slice = append(s.slice, p)
	return len(p), nil
}

func (s *sliceReadWriter) Flush() error {
	return nil
}

func (s *sliceReadWriter) Close() error {
	return nil
}
