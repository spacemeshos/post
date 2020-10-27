package persistence

import (
	"errors"
	"io"
)

type GroupReader struct {
	readers             []Reader
	activeReaderIndex   int
	readerNumLabels     uint64
	lastReaderNumLabels uint64
}

// A compile time check to ensure that GroupReader fully implements the Reader interface.
var _ Reader = (*GroupReader)(nil)

// Group groups a slice of Reader into one continuous Reader.
func Group(readers []Reader) (*GroupReader, error) {
	if len(readers) < 2 {
		return nil, errors.New("number of readers must be at least 2")
	}

	// Verify that all readers, except the last one, have the same number of labels.
	var readerNumLabels uint64
	var lastReaderNumLabels uint64
	for i := 0; i < len(readers); i++ {
		if readers[i] == nil {
			return nil, errors.New("nil readers are not allowed")
		}
		numLabels, err := readers[i].NumLabels()
		if err != nil {
			return nil, err
		}

		if numLabels == 0 {
			return nil, errors.New("0 labels readers are not allowed")
		}

		if i == len(readers)-1 {
			lastReaderNumLabels = numLabels
			continue
		}

		if readerNumLabels == 0 {
			readerNumLabels = numLabels
		} else if numLabels != readerNumLabels {
			return nil, errors.New("readers' number of labels mismatch")
		}
	}

	return &GroupReader{
		readers:             readers,
		readerNumLabels:     readerNumLabels,
		lastReaderNumLabels: lastReaderNumLabels,
	}, nil
}

func (g *GroupReader) Read(p []byte) (int, error) {
	n, err := g.readers[g.activeReaderIndex].Read(p)
	if err != nil {
		if err == io.EOF && g.activeReaderIndex < len(g.readers)-1 {
			g.activeReaderIndex++
			return g.Read(p)
		}
		return n, err
	}

	return n, nil
}

func (g *GroupReader) NumLabels() (uint64, error) {
	return uint64(len(g.readers)-1)*g.readerNumLabels + g.lastReaderNumLabels, nil
}

func (g *GroupReader) Close() error {
	for _, r := range g.readers {
		err := r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
