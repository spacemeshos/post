package persistence

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"io"
	"os"
	"path/filepath"
)

func NewLeafReader(id []byte) (*LeafReader, error) {
	if len(id) > 64 {
		return nil, fmt.Errorf("id cannot be longer than 64 bytes (got %d bytes)", len(id))
	}
	fullFilename := filepath.Join(GetPostDataPath(), hex.EncodeToString(id), filename)
	return newLeafReader(fullFilename)
}

type LeafReader struct {
	f *os.File
	b *bufio.Reader
}

func newLeafReader(name string) (*LeafReader, error) {
	f, err := os.OpenFile(name, os.O_RDONLY, OwnerReadWrite)
	if err != nil {
		return nil, err
	}
	return &LeafReader{
		f: f,
		b: bufio.NewReader(f),
	}, nil
}

func (l *LeafReader) Seek(index uint64) error {
	_, err := l.f.Seek(int64(index*32), io.SeekStart)
	if err != nil {
		return err
	}
	l.b.Reset(l.f)
	return err
}

func (l *LeafReader) ReadNext() ([]byte, error) {
	ret := make([]byte, merkle.NodeSize)
	_, err := l.b.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (l *LeafReader) Width() uint64 {
	info, err := l.f.Stat()
	if err != nil {
		return 0
	}
	return uint64(info.Size()) >> 5
}

func (l *LeafReader) Append(p []byte) (n int, err error) {
	return 0, errors.New("writing not permitted")
}

func (l *LeafReader) Close() error {
	l.b = nil
	return l.f.Close()
}
