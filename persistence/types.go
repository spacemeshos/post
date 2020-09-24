package persistence

// ReadWriter is a combined reader-writer.
type ReadWriter interface {
	ReadNext() ([]byte, error)
	Seek(index uint64) error
	Width() (uint64, error)
	Append(p []byte) (n int, err error)
	Flush() error
	Close() error
}

type Reader interface {
	ReadNext() ([]byte, error)
	Seek(index uint64) error
	Width() (uint64, error)
	Close() error
}

type Writer interface {
	Append(p []byte) (n int, err error)
	Flush() error
	Close() error
}
