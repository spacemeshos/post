package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
)

func TestTotalFiles(t *testing.T) {
	r := require.New(t)

	opts := config.InitOpts{
		NumUnits:    100,
		MaxFileSize: 2048,
	}
	r.Equal(100, opts.TotalFiles(128))

	opts = config.InitOpts{
		NumUnits:    1,
		MaxFileSize: 2048,
	}
	r.Equal(1, opts.TotalFiles(128))

	opts = config.InitOpts{
		NumUnits:    1,
		MaxFileSize: 10000000,
	}
	r.Equal(1, opts.TotalFiles(128))

	opts = config.InitOpts{
		NumUnits:    0,
		MaxFileSize: 10000000,
	}
	r.Equal(0, opts.TotalFiles(128))
}
