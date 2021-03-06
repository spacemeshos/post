package persistence

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"os"
	"path/filepath"
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

func NewLabelsWriter(datadir string, id []byte, index int, size uint) (*Writer, error) {
	dir := shared.GetInitDir(datadir, id)
	if len(id) > 64 {
		return nil, fmt.Errorf("id cannot be longer than 64 bytes (got %d bytes)", len(id))
	}

	err := os.MkdirAll(dir, shared.OwnerReadWriteExec)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(dir, shared.InitFileName(id, index))
	return newWriter(filename, uint64(size))
}

func newWriter(filename string, itemSize uint64) (*Writer, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, shared.OwnerReadWrite)
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
