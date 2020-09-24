package persistence

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// NewLabelsReader returns a new labels reader from the initialization files.
// If the initialization was split into multiple files, they will be grouped
// into one unified reader.
func NewLabelsReader(datadir string, id []byte, labelSize uint) (ReadWriter, error) {
	readers, err := GetReaders(datadir, id, labelSize)
	if err != nil {
		return nil, err
	}

	return Merge(readers)
}

func GetReaders(datadir string, id []byte, labelSize uint) ([]ReadWriter, error) {
	files, err := ioutil.ReadDir(datadir)
	if err != nil {
		return nil, fmt.Errorf("initialization directory not found: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("initialization directory (%v) is empty", datadir)
	}
	sort.Sort(numericalSorter(files))

	readers := make([]ReadWriter, 0)
	for _, file := range files {
		if !shared.IsInitFile(id, file) {
			continue
		}
		reader, err := newFileReader(filepath.Join(datadir, file.Name()), labelSize)
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}

	return readers, nil
}

func Merge(chunks []ReadWriter) (ReadWriter, error) {
	if len(chunks) == 1 {
		return chunks[0], nil
	} else {
		reader, err := Group(chunks)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}
}

type FileReader struct {
	f        *os.File
	b        *bufio.Reader
	itemSize uint
}

func newFileReader(name string, itemSize uint) (*FileReader, error) {
	// TODO(moshababo): support bit granularity
	if itemSize%8 != 0 {
		return nil, errors.New("`itemSize` must be a multiple of 8")
	}

	f, err := os.OpenFile(name, os.O_RDONLY, shared.OwnerReadWrite)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for labels reader: %v", err)
	}

	return &FileReader{
		f:        f,
		b:        bufio.NewReader(f),
		itemSize: itemSize,
	}, nil
}

func (r *FileReader) Seek(index uint64) error {
	_, err := r.f.Seek(int64(index*uint64(r.itemSize/8)), io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek in labels reader: %v", err)
	}
	r.b.Reset(r.f)
	return err
}

func (r *FileReader) ReadNext() ([]byte, error) {
	ret := make([]byte, r.itemSize/8)
	_, err := r.b.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *FileReader) Width() (uint64, error) {
	info, err := r.f.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get stats for file reader: %v", err)
	}
	return uint64(info.Size()) / uint64(r.itemSize/8), nil
}

func (r *FileReader) Flush() error {
	return nil
}

func (r *FileReader) Append(p []byte) (n int, err error) {
	return 0, errors.New("writing not permitted")
}

func (r *FileReader) Close() error {
	r.b = nil
	return r.f.Close()
}

type numericalSorter []os.FileInfo

// A compile time check to ensure that widthReader fully implements LayerReadWriter.
var _ sort.Interface = (*numericalSorter)(nil)

func (s numericalSorter) Len() int      { return len(s) }
func (s numericalSorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s numericalSorter) Less(i, j int) bool {
	pathA := s[i].Name()
	pathB := s[j].Name()

	// Get the integer values of each filename, placed after the delimiter.
	a, err1 := strconv.ParseInt(pathA[strings.Index(pathA, "-")+1:], 10, 64)
	b, err2 := strconv.ParseInt(pathB[strings.Index(pathB, "-")+1:], 10, 64)

	// If any were not numbers, sort lexicographically.
	if err1 != nil || err2 != nil {
		return pathA < pathB
	}

	return a < b
}
