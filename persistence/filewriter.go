package persistence

import (
	"bufio"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"os"
)

type FileWriter struct {
	f        *os.File
	buf      *bufio.Writer
	itemSize uint
}

// A compile time check to ensure that FileWriter fully implements the Writer interface.
//var _ Writer = (*FileWriter)(nil)

func NewFileWriter(filename string, itemSize uint) (*FileWriter, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, shared.OwnerReadWrite)
	if err != nil {
		return nil, err
	}
	return &FileWriter{
		f:        f,
		buf:      bufio.NewWriter(f),
		itemSize: itemSize,
	}, nil
}

func (w *FileWriter) Append(b []byte) error {
	_, err := w.buf.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write: %v", err)
	}

	return nil
}

func (r *FileWriter) Width() (uint64, error) {
	info, err := r.f.Stat()
	if err != nil {
		return 0, err
	}

	return uint64(info.Size()) * 8 / uint64(r.itemSize), nil
}

func (w *FileWriter) Flush() error {
	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("failed to flush disk writer: %v", err)
	}

	return nil
}

func (w *FileWriter) Close() (os.FileInfo, error) {
	err := w.buf.Flush()
	if err != nil {
		return nil, err
	}
	w.buf = nil

	info, err := w.f.Stat()
	if err != nil {
		return nil, err
	}

	err = w.f.Close()
	if err != nil {
		return nil, err
	}
	w.f = nil

	return info, nil
}
