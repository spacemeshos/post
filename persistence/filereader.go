package persistence

import (
	"bufio"
	"fmt"
	"github.com/spacemeshos/post/bitstream"
	"github.com/spacemeshos/post/shared"
	"io"
	"os"
)

type FileReader struct {
	f               *os.File
	itemSize        uint
	readNextHandler func() ([]byte, error)
}

// A compile time check to ensure that FileReader fully implements the Reader interface.
var _ Reader = (*FileReader)(nil)

func NewFileReader(name string, itemSize uint) (*FileReader, error) {
	f, err := os.OpenFile(name, os.O_RDONLY, shared.OwnerReadWrite)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for labels reader: %v", err)
	}

	// Create the reading delegate function according to the
	// required resolution: byte-granular or bit-granular.
	var readNextHandler func() ([]byte, error)
	if itemSize%8 == 0 {
		// Byte-granular read is using bufio.
		buf := bufio.NewReader(f)

		readNextHandler = func() ([]byte, error) {
			ret := make([]byte, itemSize/8)
			_, err := io.ReadFull(buf, ret)
			if err != nil {
				return nil, err
			}
			return ret, nil
		}
	} else {
		// Bit-granular read is using bitstream.
		bs := bitstream.NewReader(f)

		readNextHandler = func() ([]byte, error) {
			return bs.Read(itemSize)
		}
	}

	return &FileReader{
		f,
		itemSize,
		readNextHandler,
	}, nil
}

func (r *FileReader) ReadNext() ([]byte, error) {
	return r.readNextHandler()
}

func (r *FileReader) Width() (uint64, error) {
	info, err := r.f.Stat()
	if err != nil {
		return 0, err
	}
	return uint64(info.Size()) * 8 / uint64(r.itemSize), nil
}

func (r *FileReader) Close() error {
	r.readNextHandler = nil
	return r.f.Close()
}
