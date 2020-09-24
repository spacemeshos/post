package persistence

import (
	"errors"
	"io"
)

type ReadWriterGroup struct {
	chunks           []ReadWriter
	activeChunkIndex int
	widthPerChunk    uint64
	lastChunkWidth   uint64
}

// A compile time check to ensure that ReadWriterGroup fully implements ReadWriter.
var _ ReadWriter = (*ReadWriterGroup)(nil)

// Group groups a slice of ReadWriter into one continuous ReadWriter.
func Group(chunks []ReadWriter) (*ReadWriterGroup, error) {
	if len(chunks) < 2 {
		return nil, errors.New("number of chunks must be at least 2")
	}

	widthPerChunk, err := chunks[0].Width()
	if err != nil {
		return nil, err
	}
	if widthPerChunk == 0 {
		return nil, errors.New("0 width chunks are not allowed")
	}

	// Verify that all chunks, except the last one, have the same width.
	var lastLayerWidth uint64
	for i := 1; i < len(chunks); i++ {
		chunk := chunks[i]
		if chunk == nil {
			return nil, errors.New("nil chunks are not allowed")
		}
		width, err := chunks[i].Width()
		if err != nil {
			return nil, err
		}

		if i == len(chunks)-1 {
			lastLayerWidth = width
		} else {
			if width != widthPerChunk && i < len(chunks)-1 {
				return nil, errors.New("chunks width mismatch")
			}
		}
	}

	g := &ReadWriterGroup{
		chunks:         chunks,
		widthPerChunk:  widthPerChunk,
		lastChunkWidth: lastLayerWidth,
	}

	return g, nil
}

func (g *ReadWriterGroup) Seek(index uint64) error {
	// Find the target chunk.
	chunkIndex := int(index / g.widthPerChunk)
	if chunkIndex >= len(g.chunks) {
		return io.EOF
	}

	g.activeChunkIndex = chunkIndex

	indexInChunk := index % g.widthPerChunk
	return g.chunks[chunkIndex].Seek(indexInChunk)
}

func (g *ReadWriterGroup) ReadNext() ([]byte, error) {
	val, err := g.chunks[g.activeChunkIndex].ReadNext()
	if err != nil {
		if err == io.EOF && g.activeChunkIndex < len(g.chunks)-1 {
			g.activeChunkIndex++
			err = g.chunks[g.activeChunkIndex].Seek(0)
			if err != nil {
				return nil, err
			}
			return g.ReadNext()
		}
		return nil, err
	}

	return val, nil
}

func (g *ReadWriterGroup) Width() (uint64, error) {
	return uint64(len(g.chunks)-1)*g.widthPerChunk + g.lastChunkWidth, nil
}

func (g *ReadWriterGroup) Append(p []byte) (n int, err error) { return 0, nil }

func (g *ReadWriterGroup) Flush() error { return nil }

func (g *ReadWriterGroup) Close() error {
	for _, chunk := range g.chunks {
		err := chunk.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
