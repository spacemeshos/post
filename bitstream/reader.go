package bitstream

import (
	"io"
)

// BitReader reads bits from an io.Reader.
type BitReader struct {
	stream    io.Reader
	pending   [1]byte
	alignment uint8
}

// NewReader returns a new instance of BitReader.
func NewReader(r io.Reader) *BitReader {
	b := new(BitReader)
	b.stream = r
	b.alignment = 8
	return b
}

// Read reads the next numBits from the stream, regardless of the alignment,
// following the LSB pattern.
func (br *BitReader) Read(numBits uint) ([]byte, error) {
	size := numBits / 8
	if numBits%8 > 0 {
		size++
	}

	data := make([]byte, size)
	var idx int

	for numBits >= 8 {
		byt, err := br.ReadByte()
		if err != nil {
			return nil, err
		}

		data[idx] = byt
		idx++
		numBits -= 8
	}

	if numBits > 0 {
		var lastByte byte
		var alignment uint
		for numBits > 0 {
			bit, err := br.ReadBit()
			if err != nil {
				return nil, err
			}

			if bit {
				lastByte |= 1 << alignment
			}

			numBits--
			alignment++
		}
		data[idx] = lastByte
	}

	return data, nil
}

// ReadUint64BE reads the next numBits from the stream as uint64 in Big-Endian byte order,
// regardless of the alignment, following the LSB pattern.
func (br *BitReader) ReadUint64BE(numBits int) (uint64, error) {
	var val uint64

	for numBits >= 8 {
		byt, err := br.ReadByte()
		if err != nil {
			return 0, err
		}

		val = uint64(byt) | (val << 8)
		numBits -= 8
	}

	var err error
	for numBits > 0 && err != io.EOF {
		bit, err := br.ReadBit()
		if err != nil {
			return 0, err
		}

		val <<= 1
		if bit {
			val |= 1
		}
		numBits--
	}

	return val, nil
}

// ReadByte reads the next single byte from the stream, regardless of the alignment.
// If the byte is split, the LSB pattern is followed in bit-groups.
func (br *BitReader) ReadByte() (byte, error) {
	if br.alignment == 8 {
		n, err := br.stream.Read(br.pending[:])
		if n != 1 || (err != nil && err != io.EOF) {
			br.pending[0] = 0
			return br.pending[0], err
		}
		// Mask io.EOF for the last byte.
		if err == io.EOF {
			err = nil
		}
		return br.pending[0], err
	}

	// The byte stream is not aligned.
	// Use the current byte LS bits, combined with the next byte LS bits as MS bits.

	current := br.pending[0]
	n, err := br.stream.Read(br.pending[:])
	if n != 1 || (err != nil && err != io.EOF) {
		return 0, err
	}

	// Use the next pending byte LS bits to fill MS bits.
	current |= br.pending[0] << (8 - br.alignment)

	// Remove the used LS bits from the next pending byte.
	br.pending[0] >>= br.alignment

	return current, err
}

// ReadBit reads the next single bit from the stream, LSB first.
func (br *BitReader) ReadBit() (Bit, error) {
	if br.alignment == 8 {
		n, err := br.stream.Read(br.pending[:])
		if n != 1 || (err != nil && err != io.EOF) {
			return Zero, err
		}
		br.alignment = 0
	}
	br.alignment++

	// Read LS bit.
	lsb := Bit(br.pending[0]&1 == 1)

	// Remove LS bit.
	br.pending[0] >>= 1

	return lsb, nil
}
