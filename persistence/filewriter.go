package persistence

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spacemeshos/post/shared"
)

type FileWriter struct {
	file *os.File
	buf  *bufio.Writer

	bitsPerLabel uint
}

func NewFileWriter(filename string, bitsPerLabel uint) (*FileWriter, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, shared.OwnerReadWrite)
	if err != nil {
		return nil, err
	}
	f.Seek(0, io.SeekEnd)
	return &FileWriter{
		file:         f,
		buf:          bufio.NewWriter(f),
		bitsPerLabel: bitsPerLabel,
	}, nil
}

func (w *FileWriter) Write(b []byte) error {
	_, err := w.buf.Write(b)
	return err
}

func (w *FileWriter) Flush() error {
	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("failed to flush disk writer: %w", err)
	}

	return nil
}

func (w *FileWriter) NumLabelsWritten() (uint64, error) {
	info, err := w.file.Stat()
	if err != nil {
		return 0, err
	}

	return uint64(info.Size()) * 8 / uint64(w.bitsPerLabel), nil
}

func (w *FileWriter) Truncate(numLabels uint64) error {
	bitSize := numLabels * uint64(w.bitsPerLabel)
	if bitSize%8 != 0 {
		return fmt.Errorf("invalid `numLabels`; expected: evenly divisible by 8 (alone, or when multiplied by `labelSize`), given: %d", numLabels)
	}

	size := int64(bitSize / 8)
	if err := w.file.Truncate(size); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	w.file.Sync()
	return nil
}

func (w *FileWriter) Close() error {
	if err := w.buf.Flush(); err != nil {
		return err
	}

	return w.file.Close()
}
