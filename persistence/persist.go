package persistence

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"os"
	"path/filepath"
)

const LabelSize = 32

// OwnerReadWriteExec is a standard owner read / write / exec file permission.
const OwnerReadWriteExec = 0700

// OwnerReadWrite is a standard owner read / write file permission.
const OwnerReadWrite = 0600

type PostLabelsFileWriter struct {
	f *os.File
	w *bufio.Writer
}

func NewPostLabelsFileWriter(id []byte) (*PostLabelsFileWriter, error) {
	if len(id) > 64 {
		return nil, fmt.Errorf("id cannot be longer than 64 bytes (got %d bytes)", len(id))
	}
	labelsPath := filepath.Join(GetPostDataPath(), hex.EncodeToString(id))
	log.Info("creating directory: \"%v\"", labelsPath)
	err := os.MkdirAll(labelsPath, OwnerReadWriteExec)
	if err != nil {
		return nil, err
	}
	fullFilename := filepath.Join(labelsPath, filename)
	f, err := os.OpenFile(fullFilename, os.O_CREATE|os.O_WRONLY, OwnerReadWrite)
	if err != nil {
		return nil, err
	}
	return &PostLabelsFileWriter{
		f: f,
		w: bufio.NewWriter(f),
	}, nil
}

func (w *PostLabelsFileWriter) Write(label Label) error {
	nn, err := w.w.Write(label)
	if err != nil {
		return fmt.Errorf("failed to write label: %v", err)
	}
	if nn != LabelSize {
		return fmt.Errorf("failed to write label: expected label size of %v bytes, but wrote %v bytes (len(label)=%v)", LabelSize, nn, len(label))
	}
	return nil
}

func (w *PostLabelsFileWriter) Close() error {
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

func (w *PostLabelsFileWriter) GetLeafReader() (*LeafReader, error) {
	err := w.w.Flush()
	if err != nil {
		return nil, err
	}
	name := w.f.Name()
	return newLeafReader(name)
}
