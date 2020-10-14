package bitstream_test

import (
	"bytes"
	"errors"
	"github.com/spacemeshos/post/bitstream"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

const (
	Zero = bitstream.Zero
	One  = bitstream.One
)

var (
	NewWriter = bitstream.NewWriter
	NewReader = bitstream.NewReader
	NumBits   = shared.NumBits
)

func TestUint64BE(t *testing.T) {
	req := require.New(t)

	buf := bytes.NewBuffer(nil)
	w := NewWriter(buf)
	r := NewReader(buf)
	from := uint64(1)
	to := uint64(1 << 15)

	// Write.
	for i := from; i < to; i++ {
		err := w.WriteUint64BE(i, NumBits(i))
		req.NoError(err)
		err = w.WriteUint64BE(i, 64)
		req.NoError(err)

	}
	err := w.Flush(Zero)
	req.NoError(err)

	// Read.
	for i := from; i < to; i++ {
		num, err := r.ReadUint64BE(NumBits(i))
		req.NoError(err)
		req.Equal(i, num)
		num, err = r.ReadUint64BE(64)
		req.NoError(err)
		req.Equal(i, num)
	}
}

func TestUint64BE_Mixed(t *testing.T) {
	req := require.New(t)

	from := uint64(1)
	to := uint64(1 << 15)

	for i := from; i < to; i++ {
		buf := bytes.NewBuffer(nil)
		w := NewWriter(buf)
		r := NewReader(buf)

		// Write 3 arbitrary bits.
		err := w.WriteBit(One)
		req.NoError(err)
		err = w.WriteBit(Zero)
		req.NoError(err)
		err = w.WriteBit(One)
		req.NoError(err)

		// Write i.
		numBits := NumBits(i)
		err = w.WriteUint64BE(i, numBits)
		req.NoError(err)

		// Write the 3 LS bits of 0xFF.
		err = w.Write([]byte{0xFF}, 3)
		req.NoError(err)

		// Write i again.
		err = w.WriteUint64BE(i, numBits)
		req.NoError(err)

		// Write 3 arbitrary bits.
		err = w.WriteBit(One)
		req.NoError(err)
		err = w.WriteBit(Zero)
		req.NoError(err)
		err = w.WriteBit(One)
		req.NoError(err)

		err = w.Flush(Zero)
		req.NoError(err)

		// Read

		bit, err := r.ReadBit()
		req.NoError(err)
		req.Equal(bit, One)

		bit, err = r.ReadBit()
		req.NoError(err)
		req.Equal(bit, Zero)

		bit, err = r.ReadBit()
		req.NoError(err)
		req.Equal(bit, One)

		num, err := r.ReadUint64BE(numBits)
		req.NoError(err)
		req.Equal(i, num)

		data, err := r.Read(3)
		req.Len(data, 1)
		req.Equal(uint8(0x07), data[0])

		num, err = r.ReadUint64BE(numBits)
		req.NoError(err)
		req.Equal(i, num)

		bit, err = r.ReadBit()
		req.NoError(err)
		req.Equal(bit, One)

		bit, err = r.ReadBit()
		req.NoError(err)
		req.Equal(bit, Zero)

		bit, err = r.ReadBit()
		req.NoError(err)
		req.Equal(bit, One)
	}
}

func TestString(t *testing.T) {
	req := require.New(t)

	s := "a string"
	br := NewReader(strings.NewReader(s))
	buf := bytes.NewBuffer(nil)
	bw := NewWriter(buf)

	for {
		bit, err := br.ReadBit()
		if err == io.EOF {
			break
		}
		if err != nil {
			req.Fail(err.Error())
		}
		err = bw.WriteBit(bit)
		req.NoError(err)
	}

	req.Equal(s, buf.String())
}

func TestAlignment(t *testing.T) {
	req := require.New(t)

	s := "a string!" // 9 bytes, 72 bits.
	batchSize := 3   // 72 is divisible by 3.
	br := NewReader(strings.NewReader(s))
	buf := bytes.NewBuffer(nil)
	bw := NewWriter(buf)

	for i := 0; i < batchSize; i++ {
		bit, err := br.ReadBit()
		if err == io.EOF {
			break
		}
		if err != nil {
			req.Fail(err.Error())
		}
		err = bw.WriteBit(bit)
		req.NoError(err)
	}

	for {
		data, err := br.Read(uint(batchSize))
		if err == io.EOF {
			break
		}
		if err != nil {
			req.Fail(err.Error())
		}
		err = bw.Write(data, batchSize)
		req.NoError(err)
	}

	req.Equal(buf.String(), s)
}

func TestEOF_0(t *testing.T) {
	req := require.New(t)

	_, err := NewReader(bytes.NewReader(nil)).ReadBit()
	req.Equal(io.EOF, err)
	_, err = NewReader(bytes.NewReader(nil)).ReadByte()
	req.Equal(io.EOF, err)
	_, err = NewReader(bytes.NewReader([]byte{})).ReadBit()
	req.Equal(io.EOF, err)
	_, err = NewReader(bytes.NewReader([]byte{})).ReadByte()
	req.Equal(io.EOF, err)
}

func TestEOF_1(t *testing.T) {
	req := require.New(t)

	br := NewReader(strings.NewReader("abc"))

	b, err := br.ReadByte()
	req.NoError(err)
	req.Equal(byte('a'), b)
	b, err = br.ReadByte()
	req.NoError(err)
	req.Equal(byte('b'), b)
	b, err = br.ReadByte()
	req.NoError(err)
	req.Equal(byte('c'), b)

	b, err = br.ReadByte()
	req.Equal(io.EOF, err)
	req.Equal(byte(0), b)
}

func TestEOF_2(t *testing.T) {
	req := require.New(t)

	br := NewReader(strings.NewReader("abc"))
	buf := bytes.NewBuffer(nil)
	bw := NewWriter(buf)

	for {
		bit, err := br.ReadBit()
		if err == io.EOF {
			break
		}
		if err != nil {
			req.Fail(err.Error())
		}
		err = bw.WriteBit(bit)
		req.NoError(err)
	}

	req.Equal("abc", buf.String())
}

func TestEOF_3(t *testing.T) {
	req := require.New(t)

	br := NewReader(bytes.NewReader([]byte{0x0F}))
	buf := bytes.NewBuffer(nil)
	bw := NewWriter(buf)

	for i := 0; i < 4; i++ {
		bit, err := br.ReadBit()
		if err == io.EOF {
			break
		}
		if err != nil {
			req.Fail(err.Error())
		}
		err = bw.WriteBit(bit)
		req.NoError(err)
	}

	err := bw.Flush(One)
	req.NoError(err)

	err = bw.WriteByte(0xAA)
	req.NoError(err)

	data := buf.Bytes()
	req.Len(data, 2)
	req.Equal(byte(0xFF), data[0])
	req.Equal(byte(0xAA), data[1])
}

func TestBadWriter_0(t *testing.T) {
	req := require.New(t)

	br := NewWriter(&badWriter{})
	for i := 0; i < 7; i++ {
		err := br.WriteBit(One)
		req.NoError(err)

	}
	err := br.WriteBit(One)
	req.Equal(err, ErrBadWriter)
}

func TestBadWriter_1(t *testing.T) {
	req := require.New(t)

	br := NewWriter(&badWriter{})
	err := br.WriteUint64BE(256, 8)
	req.Equal(err, ErrBadWriter)
}

type badWriter struct{}

var ErrBadWriter = errors.New("bad writer")

func (w *badWriter) Write(p []byte) (n int, err error) {
	return 0, ErrBadWriter
}
