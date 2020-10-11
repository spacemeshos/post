package bitstream

import (
	"io"
)

// BitWriter writes bits to an io.Writer.
type BitWriter struct {
	stream    io.Writer
	pending   [1]byte
	alignment uint8
}

// NewWriter returns a new instance of BitWriter.
func NewWriter(w io.Writer) *BitWriter {
	bw := new(BitWriter)
	bw.stream = w
	bw.alignment = 0 // less-significant bit
	return bw
}

// WriteNum writes the numBits of data to the stream, regardless of the alignment.
// If bytes are to be split (from data due to numBits, or on stream due to alignment), the LSB pattern is followed in bit-groups.
func (bw *BitWriter) Write(data []byte, numBits int) error {
	var idx uint
	for numBits >= 8 {
		if err := bw.WriteByte(data[idx]); err != nil {
			return err
		}
		numBits -= 8
		idx++
	}

	for numBits > 0 {
		if err := bw.WriteBit(data[idx]&1 == 1); err != nil {
			return err
		}
		data[idx] >>= 1
		numBits--
	}

	return nil
}

// WriteNum writes the next numBits LS bits of val, in Big-Endian byte order, regardless of the alignment.
// If bytes are to be split (from data due to numBits, or on stream due to alignment), the LSB pattern is followed in bit-groups.
func (bw *BitWriter) WriteUint64BE(val uint64, numBits int) error {
	// Eliminate unnecessary MS bits.
	val <<= 64 - uint(numBits)

	// Write bytes in Big-Endian order.
	for numBits >= 8 {
		if err := bw.WriteByte(byte(val >> 56)); err != nil {
			return err
		}
		val <<= 8
		numBits -= 8
	}

	// Write the remaining bits.
	for numBits > 0 {
		if err := bw.WriteBit((val >> 63) == 1); err != nil {
			return err
		}
		val <<= 1
		numBits--
	}

	return nil
}

// WriteByte writes a single byte to the stream, regardless of the alignment.
// If the byte is to be split due to alignment, the LSB pattern is followed in bit-groups.
func (bw *BitWriter) WriteByte(byte byte) error {
	// Fill the pending byte MS bits with LS bits.
	bw.pending[0] |= byte << bw.alignment

	if n, err := bw.stream.Write(bw.pending[:]); n != 1 || err != nil {
		return err
	}

	// Fill the new pending byte LS bits with MS bits.
	bw.pending[0] = byte >> (8 - bw.alignment)

	return nil
}

// WriteBit writes a single bit to the stream, LSB first.
func (bw *BitWriter) WriteBit(bit Bit) error {
	if bit {
		bw.pending[0] |= 1 << bw.alignment
	}

	bw.alignment++

	if bw.alignment == 8 {
		if n, err := bw.stream.Write(bw.pending[:]); n != 1 || err != nil {
			return err
		}
		bw.pending[0] = 0
		bw.alignment = 0
	}

	return nil
}

// Flush flushes the currently pending byte to the stream by filling it with bit.
func (bw *BitWriter) Flush(bit Bit) error {
	for bw.alignment != 0 {
		if err := bw.WriteBit(bit); err != nil {
			return err
		}
	}

	return nil
}
