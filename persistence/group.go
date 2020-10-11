package persistence

import (
	"errors"
	"io"
)

type GroupReader struct {
	readers           []Reader
	activeReaderIndex int
	readerWidth       uint64
	lastReaderWidth   uint64
}

// A compile time check to ensure that GroupReader fully implements the Reader interface.
var _ Reader = (*GroupReader)(nil)

// Group groups a slice of ReadWriter into one continuous ReadWriter.
func Group(readers []Reader) (*GroupReader, error) {
	if len(readers) < 2 {
		return nil, errors.New("number of readers must be at least 2")
	}

	// Verify that all readers, except the last one, have the same width.
	var readerWidth uint64
	var lastReaderWidth uint64
	for i := 0; i < len(readers); i++ {
		if readers[i] == nil {
			return nil, errors.New("nil readers are not allowed")
		}
		width, err := readers[i].Width()
		if err != nil {
			return nil, err
		}

		if width == 0 {
			return nil, errors.New("0 width readers are not allowed")
		}

		if i == len(readers)-1 {
			lastReaderWidth = width
			continue
		}

		if readerWidth == 0 {
			readerWidth = width
		} else if width != readerWidth {
			return nil, errors.New("readers width mismatch")
		}
	}

	return &GroupReader{
		readers:         readers,
		readerWidth:     readerWidth,
		lastReaderWidth: lastReaderWidth,
	}, nil
}

func (g *GroupReader) ReadNext() ([]byte, error) {
	val, err := g.readers[g.activeReaderIndex].ReadNext()
	if err != nil {
		if err == io.EOF && g.activeReaderIndex < len(g.readers)-1 {
			g.activeReaderIndex++
			return g.ReadNext()
		}
		return nil, err
	}

	return val, nil
}

func (g *GroupReader) Width() (uint64, error) {
	return uint64(len(g.readers)-1)*g.readerWidth + g.lastReaderWidth, nil
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
