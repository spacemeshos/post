package initialization

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitializer_CalcParallelism(t *testing.T) {
	r := require.New(t)

	files, infile := NewInitializer(&Config{}, nil).
		calcParallelism(0)
	r.Equal(files, 1)
	r.Equal(infile, 1)

	files, infile = NewInitializer(&Config{MaxWriteFilesParallelism: 2, MaxWriteInFileParallelism: 1}, nil).
		calcParallelism(2)
	r.Equal(files, 2)
	r.Equal(infile, 1)

	files, infile = NewInitializer(&Config{MaxWriteFilesParallelism: 2, MaxWriteInFileParallelism: 3}, nil).
		calcParallelism(5)
	r.Equal(files, 1)
	r.Equal(infile, 3)

	files, infile = NewInitializer(&Config{MaxWriteFilesParallelism: 2, MaxWriteInFileParallelism: 3}, nil).
		calcParallelism(7)
	r.Equal(files, 2)
	r.Equal(infile, 3)

	files, infile = NewInitializer(&Config{MaxWriteFilesParallelism: 2, MaxWriteInFileParallelism: 100}, nil).
		calcParallelism(6)
	r.Equal(files, 1)
	r.Equal(infile, 6)
}
