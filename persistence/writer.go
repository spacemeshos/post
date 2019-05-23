package persistence

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	LabelGroupSize = shared.LabelGroupSize

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

	log.Infof("creating directory: \"%v\"", dir)
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

func (w *Writer) Close() error {
	err := w.w.Flush()
	if err != nil {
		return err
	}
	w.w = nil
	if info, err := w.f.Stat(); err == nil {
		log.Infof("closing file \"%v\", %v bytes written", info.Name(), info.Size())
	}

	err = w.f.Close()
	if err != nil {
		return err
	}
	w.f = nil
	return nil
}

func (w *Writer) GetReader() (*Reader, error) {
	err := w.w.Flush()
	if err != nil {
		return nil, err
	}

	return newReader(w.f.Name(), w.itemSize)
}

type ResetResult struct {
	DeletedDir        string
	NumOfDeletedFiles int
}

func Reset(dir string) (*ResetResult, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, ErrDirNotFound
	}

	if len(files) == 0 {
		return nil, ErrDirEmpty
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to delete directory (%v)", dir)
	}

	return &ResetResult{
		DeletedDir:        dir,
		NumOfDeletedFiles: len(files),
	}, nil
}
