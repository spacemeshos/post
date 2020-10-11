// Package bitstream provides wrappers for io.Writer and io.Reader to allow
// bit-granularity access to the stream, following the LSB pattern, where
// least-significant bits are written/read first.
package bitstream

type Bit bool

const (
	Zero Bit = false
	One  Bit = true
)
