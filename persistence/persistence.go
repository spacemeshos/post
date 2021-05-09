package persistence

import (
	"fmt"
	"github.com/spacemeshos/post/shared"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

type Reader interface {
	Read(p []byte) (n int, err error)
	NumLabels() (uint64, error)
	Close() error
}

// NewLabelsReader returns a new labels reader from the initialization files.
// If the initialization was split into multiple files, they will be grouped
// into one unified reader.
func NewLabelsReader(datadir string, bitsPerLabel uint) (Reader, error) {
	readers, err := GetReaders(datadir, bitsPerLabel)
	if err != nil {
		return nil, err
	}
	if len(readers) == 1 {
		return readers[0], nil
	}

	return Group(readers)
}

func GetReaders(datadir string, bitsPerLabel uint) ([]Reader, error) {
	files, err := ioutil.ReadDir(datadir)
	if err != nil {
		return nil, fmt.Errorf("initialization directory not found: %v", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("initialization directory (%v) is empty", datadir)
	}

	// Filter.
	var initFiles []os.FileInfo
	for _, file := range files {
		if shared.IsInitFile(file) {
			initFiles = append(initFiles, file)
		}
	}

	// Sort.
	sort.Sort(numericalSorter(initFiles))

	// Initialize readers.
	var readers []Reader
	for _, file := range initFiles {
		reader, err := NewFileReader(filepath.Join(datadir, file.Name()), bitsPerLabel)
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}

	return readers, nil
}

func NewLabelsWriter(datadir string, index int, bitsPerLabel uint) (*FileWriter, error) {
	if err := os.MkdirAll(datadir, shared.OwnerReadWriteExec); err != nil {
		return nil, err
	}

	filename := filepath.Join(datadir, shared.InitFileName(index))
	return NewFileWriter(filename, bitsPerLabel)
}
