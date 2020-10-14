package persistence

import (
	"fmt"
	"github.com/spacemeshos/post/shared"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

type Writer interface {
	Write(p []byte) error
	Flush() error
	Close() (*os.FileInfo, error)
}

type Reader interface {
	Read(p []byte) (n int, err error)
	Width() (uint64, error)
	Close() error
}

func NewLabelsWriter(datadir string, id []byte, index int, labelSize uint) (*FileWriter, error) {
	if len(id) > 64 {
		return nil, fmt.Errorf("id cannot be longer than 64 bytes; given: %d", len(id))
	}

	if err := os.MkdirAll(datadir, shared.OwnerReadWriteExec); err != nil {
		return nil, err
	}

	filename := filepath.Join(datadir, shared.InitFileName(id, index))
	return NewFileWriter(filename, labelSize)
}

// NewLabelsReader returns a new labels reader from the initialization files.
// If the initialization was split into multiple files, they will be grouped
// into one unified reader.
func NewLabelsReader(datadir string, id []byte, labelSize uint) (Reader, error) {
	readers, err := GetReaders(datadir, id, labelSize)
	if err != nil {
		return nil, err
	}
	if len(readers) == 1 {
		return readers[0], nil
	}

	return Group(readers)
}

func GetReaders(datadir string, id []byte, labelSize uint) ([]Reader, error) {
	files, err := ioutil.ReadDir(datadir)
	if err != nil {
		return nil, fmt.Errorf("initialization directory not found: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("initialization directory (%v) is empty", datadir)
	}
	sort.Sort(numericalSorter(files))

	readers := make([]Reader, 0)
	for _, file := range files {
		if !shared.IsInitFile(id, file) {
			continue
		}
		reader, err := NewFileReader(filepath.Join(datadir, file.Name()), labelSize)
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}

	return readers, nil
}
