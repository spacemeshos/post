package persistence

import (
	"encoding/binary"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestGroup(t *testing.T) {
	r := require.New(t)

	// Create 9 labels.
	labels := genLabels(9, labelSize)

	// Split the labels into 3 separate writers.
	writers := make([]Writer, 3)
	slices := make([][][]byte, 3)
	writers[0] = newSliceWriter(&slices[0], labelSize)
	writers[1] = newSliceWriter(&slices[1], labelSize)
	writers[2] = newSliceWriter(&slices[2], labelSize)
	_ = writers[0].Append(labels[0])
	_ = writers[0].Append(labels[1])
	_ = writers[0].Append(labels[2])
	_ = writers[1].Append(labels[3])
	_ = writers[1].Append(labels[4])
	_ = writers[1].Append(labels[5])
	_ = writers[2].Append(labels[6])
	_ = writers[2].Append(labels[7])
	_ = writers[2].Append(labels[8])

	// Create group reader.
	readers := make([]Reader, 3)
	readers[0] = newSliceReader(slices[0], labelSize)
	readers[1] = newSliceReader(slices[1], labelSize)
	readers[2] = newSliceReader(slices[2], labelSize)
	reader, err := Group(readers)
	r.NoError(err)

	width, err := reader.Width()
	r.NoError(err)
	r.Equal(width, uint64(len(labels)))

	for _, label := range labels {
		val, err := reader.ReadNext()
		r.NoError(err)
		r.Equal(val, label)
	}

	// Verify EOF.
	val, err := reader.ReadNext()
	r.Equal(err, io.EOF)
	r.Nil(val)

	err = reader.Close()
	r.NoError(err)
}

func TestGroupWithShorterLastLayer(t *testing.T) {
	r := require.New(t)

	// Create 9 labels.
	labels := genLabels(7, labelSize)

	// Split the labels into 3 separate writers.
	writers := make([]Writer, 3)
	slices := make([][][]byte, 3)
	writers[0] = newSliceWriter(&slices[0], labelSize)
	writers[1] = newSliceWriter(&slices[1], labelSize)
	writers[2] = newSliceWriter(&slices[2], labelSize)
	_ = writers[0].Append(labels[0])
	_ = writers[0].Append(labels[1])
	_ = writers[0].Append(labels[2])
	_ = writers[1].Append(labels[3])
	_ = writers[1].Append(labels[4])
	_ = writers[1].Append(labels[5])
	_ = writers[2].Append(labels[6])

	// Create group reader.
	readers := make([]Reader, 3)
	readers[0] = newSliceReader(slices[0], labelSize)
	readers[1] = newSliceReader(slices[1], labelSize)
	readers[2] = newSliceReader(slices[2], labelSize)
	reader, err := Group(readers)
	r.NoError(err)

	width, err := reader.Width()
	r.NoError(err)
	r.Equal(width, uint64(len(labels)))

	// Iterate over the layer.
	for _, label := range labels {
		val, err := reader.ReadNext()
		r.NoError(err)
		r.Equal(val, label)
	}

	// Verify EOF.
	val, err := reader.ReadNext()
	r.Equal(err, io.EOF)
	r.Nil(val)

	err = reader.Close()
	r.NoError(err)

	// Test last reader with 0 width.
	readers = make([]Reader, 3)
	readers[0] = newSliceReader(slices[0], labelSize)
	readers[1] = newSliceReader(slices[1], labelSize)
	readers[2] = newSliceReader([][]byte{}, labelSize)
	_, err = Group(readers)
	r.EqualError(err, "0 width readers are not allowed")
}

func TestGroupWithShorterMidReader(t *testing.T) {
	r := require.New(t)

	// Create 7 labels.
	labels := genLabels(7, labelSize)

	// Split the labels into 3 separate writers.
	writers := make([]Writer, 3)
	slices := make([][][]byte, 3)
	writers[0] = newSliceWriter(&slices[0], labelSize)
	writers[1] = newSliceWriter(&slices[1], labelSize)
	writers[2] = newSliceWriter(&slices[2], labelSize)
	_ = writers[0].Append(labels[0])
	_ = writers[0].Append(labels[1])
	_ = writers[0].Append(labels[2])
	_ = writers[1].Append(labels[3])
	_ = writers[2].Append(labels[4])
	_ = writers[2].Append(labels[5])
	_ = writers[2].Append(labels[6])

	// Create group reader.
	readers := make([]Reader, 3)
	readers[0] = newSliceReader(slices[0], labelSize)
	readers[1] = newSliceReader(slices[1], labelSize)
	readers[2] = newSliceReader(slices[2], labelSize)
	_, err := Group(readers)
	r.EqualError(err, "readers width mismatch")
}

func genLabels(num int, labelSize uint) [][]byte {
	labels := make([][]byte, num)
	for i := 0; i < num; i++ {
		labels[i] = NewLabelFromUint64(uint64(i), labelSize)
	}
	return labels
}

func NewLabelFromUint64(i uint64, labelSize uint) []byte {
	b := make([]byte, labelSize/8)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

type sliceWriter struct {
	slice    *[][]byte
	position uint64
	itemSize uint
}

// A compile time check to ensure that sliceWriter fully implements the Writer interface.
var _ Writer = (*sliceWriter)(nil)

func newSliceWriter(slice *[][]byte, itemSize uint) *sliceWriter {
	return &sliceWriter{
		slice:    slice,
		itemSize: itemSize,
	}
}

func (s *sliceWriter) Append(p []byte) error {
	*s.slice = append(*s.slice, p)
	return nil
}

func (s *sliceWriter) Flush() error {
	return nil
}

func (s *sliceWriter) Close() error {
	return nil
}

type sliceReader struct {
	slice    [][]byte
	position uint64
	itemSize uint
}

// A compile time check to ensure that sliceReader fully implements the Reader interface.
var _ Reader = (*sliceReader)(nil)

func newSliceReader(slice [][]byte, itemSize uint) *sliceReader {
	return &sliceReader{
		slice:    slice,
		itemSize: itemSize,
	}
}

func (s *sliceReader) ReadNext() ([]byte, error) {
	if s.position >= uint64(len(s.slice)) {
		return nil, io.EOF
	}
	value := make([]byte, s.itemSize/8)
	copy(value, s.slice[s.position])
	s.position++
	return value, nil
}

func (s *sliceReader) Width() (uint64, error) {
	return uint64(len(s.slice)), nil
}

func (s *sliceReader) Close() error {
	return nil
}
