package initialization

import (
	"github.com/spacemeshos/post/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitializer_CalcParallelism(t *testing.T) {
	r := require.New(t)

	cfg := config.DefaultConfig()

	init, err := NewInitializer(cfg, nil)
	r.NoError(err)
	files, infile := init.calcParallelism(0)
	r.Equal(files, 1)
	r.Equal(infile, 1)

	cfg.MaxWriteFilesParallelism = 2
	cfg.MaxWriteInFileParallelism = 1
	init, err = NewInitializer(cfg, nil)
	r.NoError(err)
	files, infile = init.calcParallelism(2)
	r.Equal(files, 2)
	r.Equal(infile, 1)

	cfg.MaxWriteFilesParallelism = 2
	cfg.MaxWriteInFileParallelism = 3
	init, err = NewInitializer(cfg, nil)
	r.NoError(err)
	files, infile = init.calcParallelism(5)
	r.Equal(files, 1)
	r.Equal(infile, 3)
	files, infile = init.calcParallelism(7)
	r.Equal(files, 2)
	r.Equal(infile, 3)

	cfg.MaxWriteFilesParallelism = 2
	cfg.MaxWriteInFileParallelism = 100
	init, err = NewInitializer(cfg, nil)
	files, infile = init.calcParallelism(6)
	r.Equal(files, 1)
	r.Equal(infile, 6)
}
