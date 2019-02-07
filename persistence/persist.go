package persistence

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"post-private/datatypes"
)

type PostLabelsWriter interface {
	Write(label datatypes.Label) error
	Close() error
}

type postLabelsWriter struct {
	f *os.File
	w *bufio.Writer
}

func NewPostLabelsWriter(id []byte) (PostLabelsWriter, error) {
	labelsPath := filepath.Join(GetPostDataPath(), hex.EncodeToString(id))
	s, _ := filepath.Abs(labelsPath)
	fmt.Println("creating directory:", s)
	err := os.MkdirAll(labelsPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	fullFilename := filepath.Join(labelsPath, filename)
	f, err := os.OpenFile(fullFilename, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &postLabelsWriter{
		f: f,
		w: bufio.NewWriter(f),
	}, nil
}

func (w *postLabelsWriter) Write(label datatypes.Label) error {
	nn, err := w.w.Write(label)
	if err != nil {
		return err
	}
	if nn != datatypes.LabelSize {
		return fmt.Errorf("expected label size of %v bytes, but wrote %v bytes (len(label)=%v)", datatypes.LabelSize, nn, len(label))
	}
	return nil
}

func (w *postLabelsWriter) Close() error {
	err := w.w.Flush()
	if err != nil {
		return err
	}
	w.w = nil
	if info, err := w.f.Stat(); err == nil {
		fmt.Printf("closing file: '%v' (%v bytes)\n", info.Name(), info.Size())
	}
	err = w.f.Close()
	if err != nil {
		return err
	}
	w.f = nil
	return nil
}

type PostLabelsReader interface {
	Read() (datatypes.Label, error)
	Close() error
}

type postLabelsReader struct {
	f *os.File
	r *bufio.Reader
}

func NewPostLabelsReader(id []byte) (PostLabelsReader, error) {
	fullFilename := filepath.Join(GetPostDataPath(), hex.EncodeToString(id), filename)
	f, err := os.OpenFile(fullFilename, os.O_RDONLY, os.ModePerm)
	if os.IsNotExist(err) {
		return nil, err
	}
	if err != nil {
		panic(err)
	}
	return &postLabelsReader{
		f: f,
		r: bufio.NewReader(f),
	}, nil
}

func (r *postLabelsReader) Read() (datatypes.Label, error) {
	var l datatypes.Label = make([]byte, datatypes.LabelSize)
	n, err := r.r.Read(l)
	if err != nil {
		if err == io.EOF && n != 0{
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return l, nil
}

func (r *postLabelsReader) Close() error {
	r.r = nil
	err := r.f.Close()
	if err != nil {
		return err
	}
	r.f = nil
	return nil
}
