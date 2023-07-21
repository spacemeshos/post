package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spacemeshos/post/shared"
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
	files, err := os.ReadDir(datadir)
	if err != nil {
		return nil, fmt.Errorf("initialization directory not found: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("initialization directory (%v) is empty", datadir)
	}

	// Filter.
	var initFiles []os.FileInfo
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		if shared.IsInitFile(info) {
			initFiles = append(initFiles, info)
		}
	}

	// Sort.
	sort.Sort(NumericalSorter(initFiles))

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
