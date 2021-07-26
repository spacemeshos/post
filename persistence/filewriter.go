package persistence

import (
	"bufio"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"os"
)

type FileWriter struct {
	file *os.File
	buf  *bufio.Writer

	bitsPerLabel uint
}

func NewFileWriter(filename string, bitsPerLabel uint) (*FileWriter, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, shared.OwnerReadWrite)
	if err != nil {
		return nil, err
	}
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
		return fmt.Errorf("failed to flush disk writer: %v", err)
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
		return fmt.Errorf("invalid `numLabels`; expected: evenly divisible by 8 (alone, or when multipled by `labelSize`), given: %d", numLabels)
	}

	size := int64(bitSize / 8)
	if err := w.file.Truncate(size); err != nil {
		return fmt.Errorf("failed to truncate file: %v", err)
	}

	return nil
}

func (w *FileWriter) Close() (*os.FileInfo, error) {
	err := w.buf.Flush()
	if err != nil {
		return nil, err
	}
	w.buf = nil

	info, err := w.file.Stat()
	if err != nil {
		return nil, err
	}

	err = w.file.Close()
	if err != nil {
		return nil, err
	}
	w.file = nil

	return &info, nil
}
