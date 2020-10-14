package persistence

import (
	"bufio"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"os"
)

type FileReader struct {
	file     *os.File
	buf      *bufio.Reader
	itemSize uint
}

// A compile time check to ensure that FileReader fully implements the Reader interface.
var _ Reader = (*FileReader)(nil)

func NewFileReader(name string, itemSize uint) (*FileReader, error) {
	file, err := os.OpenFile(name, os.O_RDONLY, shared.OwnerReadWrite)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for labels reader: %v", err)
	}
	buf := bufio.NewReader(file)

	return &FileReader{
		file,
		buf,
		itemSize,
	}, nil
}

func (r *FileReader) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}

func (r *FileReader) Width() (uint64, error) {
	info, err := r.file.Stat()
	if err != nil {
		return 0, err
	}
	return uint64(info.Size()) * 8 / uint64(r.itemSize), nil
}

func (r *FileReader) Close() error {
	r.buf = nil
	return r.file.Close()
}
