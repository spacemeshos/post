package shared

import (
	"github.com/spacemeshos/bitstream"
	"io"
)

// GranSpecificReader provides a wrapper for io.Reader to allow granularity-specific
// access to the stream according to the defined item size, where bit-granular and
// byte-granular sizes are supported via a specialized code path.
type GranSpecificReader struct {
	ReadNext       func() ([]byte, error)
	ReadNextUintBE func() (uint64, error)
}

func NewGranSpecificReader(rd io.Reader, itemBitSize uint) *GranSpecificReader {
	gsReader := new(GranSpecificReader)
	if itemBitSize%8 == 0 {
		// Byte-granular reader is using the underlying reader directly.
		gsReader.ReadNext = func() ([]byte, error) {
			b := make([]byte, itemBitSize/8)
			_, err := io.ReadFull(rd, b)
			if err != nil {
				return nil, err
			}
			return b, nil
		}
		gsReader.ReadNextUintBE = func() (uint64, error) {
			b, err := gsReader.ReadNext()
			if err != nil {
				return 0, err
			}
			return UintBE(b), nil
		}
	} else {
		// Bit-granular reader is using bitstream as a wrapper for the underlying reader.
		br := bitstream.NewReader(rd)
		gsReader.ReadNext = func() ([]byte, error) {
			return br.Read(itemBitSize)
		}
		gsReader.ReadNextUintBE = func() (uint64, error) {
			return br.ReadUint64BE(int(itemBitSize))
		}
	}

	return gsReader
}

// GranSpecificWriter provides a wrapper for io.Writer to allow granularity-specific
// access to the stream according to the defined item size, where bit-granular and
// byte-granular sizes are supported via a specialized code path.
type GranSpecificWriter struct {
	Write       func([]byte) error
	WriteUintBE func(uint64) error
	Flush       func() error
}

func NewGranSpecificWriter(w io.Writer, itemBitSize uint) *GranSpecificWriter {
	gsWriter := new(GranSpecificWriter)
	if itemBitSize%8 == 0 {
		// Byte-granular writer is using the underlying writer directly.
		gsWriter.Write = func(b []byte) error {
			if _, err := w.Write(b); err != nil {
				return err
			}
			return nil
		}
		gsWriter.WriteUintBE = func(v uint64) error {
			b := make([]byte, itemBitSize/8)
			PutUintBE(b, v)
			return gsWriter.Write(b)
		}
		gsWriter.Flush = func() error { return nil }
	} else {
		// Bit-granular writer is using bitstream as a wrapper for the underlying writer.
		br := bitstream.NewWriter(w)
		gsWriter.Write = func(b []byte) error {
			return br.Write(b, int(itemBitSize))
		}
		gsWriter.WriteUintBE = func(v uint64) error {
			return br.WriteUint64BE(v, int(itemBitSize))
		}
		gsWriter.Flush = func() error {
			return br.Flush(bitstream.Zero)
		}
	}

	return gsWriter
}
