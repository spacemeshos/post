package persistence

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"os"
	"path/filepath"
)

// OwnerReadWriteExec is a standard owner read / write / exec file permission.
const OwnerReadWriteExec = 0700

// OwnerReadWrite is a standard owner read / write file permission.
const OwnerReadWrite = 0600

type Writer struct {
	f        *os.File
	w        *bufio.Writer
	itemSize uint64
}

func NewLabelsWriter(id []byte, index int) (*Writer, error) {
	if len(id) > 64 {
		return nil, fmt.Errorf("id cannot be longer than 64 bytes (got %d bytes)", len(id))
	}

	labelsPath := filepath.Join(GetPostDataPath(), hex.EncodeToString(id))
	log.Info("creating directory: \"%v\"", labelsPath)
	err := os.MkdirAll(labelsPath, OwnerReadWriteExec)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(labelsPath, fmt.Sprintf("%x-%d", id, index))
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
		log.With().Info("closing file",
			log.String("filename", info.Name()),
			log.Uint64("size_in_bytes", uint64(info.Size())),
		)
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
