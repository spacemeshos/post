package persistence

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/spacemeshos/post/config"
	"os"
	"path/filepath"
)

const (
	LabelGroupSize = config.LabelGroupSize

	// OwnerReadWriteExec is a standard owner read / write / exec file permission.
	OwnerReadWriteExec = 0700

	// OwnerReadWrite is a standard owner read / write file permission.
	OwnerReadWrite = 0600
)

var (
	ErrDirNotFound = errors.New("initialization directory not found")
	ErrDirEmpty    = errors.New("initialization directory is empty")
)

type Writer struct {
	f        *os.File
	w        *bufio.Writer
	itemSize uint64
}

func NewLabelsWriter(id []byte, index int, dir string) (*Writer, error) {
	if len(id) > 64 {
		return nil, fmt.Errorf("id cannot be longer than 64 bytes (got %d bytes)", len(id))
	}

	err := os.MkdirAll(dir, OwnerReadWriteExec)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(dir, fmt.Sprintf("%x-%d", id, index))
	return newWriter(filename, LabelGroupSize)
}

func newWriter(filename string, itemSize uint64) (*Writer, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, OwnerReadWrite)
	if err != nil {
		return nil, err
	}
	return &Writer{
		f:        f,
		w:        bufio.NewWriter(f),
		itemSize: itemSize,
	}, nil
}

func (w *Writer) Write(item []byte) error {
	nn, err := w.w.Write(item)
	if err != nil {
		return fmt.Errorf("failed to write: %v", err)
	}
	if uint64(nn) != w.itemSize {
		return fmt.Errorf("failed to write: expected size of %v bytes, but wrote %v bytes (len(lg)=%v)", w.itemSize, nn, len(item))
	}
	return nil
}

func (w *Writer) Close() (os.FileInfo, error) {
	err := w.w.Flush()
	if err != nil {
		return nil, err
	}
	w.w = nil

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

func (w *Writer) GetReader() (*Reader, error) {
	err := w.w.Flush()
	if err != nil {
		return nil, err
	}

	return newReader(w.f.Name(), w.itemSize)
}

func (w *Writer) Filename() string { return w.f.Name() }
