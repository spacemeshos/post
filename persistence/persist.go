package persistence

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/post-private/util"
	"io"
	"os"
	"path/filepath"
)

// OwnerReadWriteExec is a standard owner read / write / exec file permission.
const OwnerReadWriteExec = 0700

// OwnerReadWrite is a standard owner read / write file permission.
const OwnerReadWrite = 0600

type PostLabelsFileWriter struct {
	f *os.File
	w *bufio.Writer
}

func NewPostLabelsFileWriter(id []byte) (PostLabelsFileWriter, error) {
	labelsPath := filepath.Join(GetPostDataPath(), hex.EncodeToString(id))
	log.Info("creating directory: \"%v\"", labelsPath)
	err := os.MkdirAll(labelsPath, OwnerReadWriteExec)
	if err != nil {
		return PostLabelsFileWriter{}, err
	}
	fullFilename := filepath.Join(labelsPath, filename)
	f, err := os.OpenFile(fullFilename, os.O_CREATE|os.O_WRONLY, OwnerReadWrite)
	if err != nil {
		return PostLabelsFileWriter{}, err
	}
	return PostLabelsFileWriter{
		f: f,
		w: bufio.NewWriter(f),
	}, nil
}

func (w *PostLabelsFileWriter) Write(label util.Label) error {
	nn, err := w.w.Write(label)
	if err != nil {
		return err
	}
	if nn != util.LabelSize {
		return fmt.Errorf("expected label size of %v bytes, but wrote %v bytes (len(label)=%v)", util.LabelSize, nn, len(label))
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

type PostLabelsFileReader struct {
	f        *os.File
	r        *bufio.Reader
	position uint64
}

func NewPostLabelsFileReader(id []byte) (PostLabelsFileReader, error) {
	fullFilename := filepath.Join(GetPostDataPath(), hex.EncodeToString(id), filename)
	f, err := os.OpenFile(fullFilename, os.O_RDONLY, OwnerReadWrite)
	if os.IsNotExist(err) {
		return PostLabelsFileReader{}, err
	}
	if err != nil {
		panic(err)
	}
	return PostLabelsFileReader{
		f:        f,
		r:        bufio.NewReader(f),
		position: 0,
	}, nil
}

func (r *PostLabelsFileReader) Read() (uint64, util.Label, error) {
	var l util.Label = make([]byte, util.LabelSize)
	n, err := r.r.Read(l)
	if err != nil {
		if err == io.EOF && n != 0 { // n < util.LabelSize or we wouldn't get EOF
			return 0, nil, io.ErrUnexpectedEOF
		}
		return 0, nil, err
	}
	position := r.position
	r.position++
	return position, l, nil
}

func (r *PostLabelsFileReader) Close() error {
	r.r = nil
	err := r.f.Close()
	if err != nil {
		return err
	}
	r.f = nil
	return nil
}
