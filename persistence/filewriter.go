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

type FileWriter struct {
	f        *os.File
	w        *bufio.Writer
	itemSize uint
}

func NewLabelsWriter(datadir string, id []byte, index int, itemSize uint) (*FileWriter, error) {
	// TODO(moshababo): support bit granularity
	if itemSize%8 != 0 {
		return nil, errors.New("`itemSize` must be a multiple of 8")
	}

	if len(id) > 64 {
		return nil, fmt.Errorf("id cannot be longer than 64 bytes; given: %d", len(id))
	}

	if err := os.MkdirAll(datadir, shared.OwnerReadWriteExec); err != nil {
		return nil, err
	}

	filename := filepath.Join(datadir, shared.InitFileName(id, index))
	return newFileWriter(filename, itemSize)
}

func newFileWriter(filename string, itemSize uint) (*FileWriter, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, shared.OwnerReadWrite)
	if err != nil {
		return nil, err
	}
	return &FileWriter{
		f:        f,
		w:        bufio.NewWriter(f),
		itemSize: itemSize,
	}, nil
}

func (w *FileWriter) Write(b []byte) error {
	_, err := w.w.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write: %v", err)
	}

	// TODO(moshababo): add validation to num of bytes written
	//expectedWritten := w.itemSize*batchSize / 8
	//if uint(nn) != expectedWritten {
	//	return fmt.Errorf("failed to write: invalid number of bytes written; expected: %d, actual: %d",expectedWritten, nn)
	//}
	return nil
}

func (w *FileWriter) Flush() error {
	if err := w.w.Flush(); err != nil {
		return fmt.Errorf("failed to flush disk writer: %v", err)
	}

	return nil
}

func (w *FileWriter) Close() (os.FileInfo, error) {
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

func (w *FileWriter) GetReader() (*FileReader, error) {
	err := w.w.Flush()
	if err != nil {
		return nil, err
	}

	return newFileReader(w.f.Name(), w.itemSize)
}

func (w *FileWriter) Filename() string { return w.f.Name() }
