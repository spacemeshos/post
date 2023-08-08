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

func TestOptsValidateScryptParams(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	opts := config.DefaultInitOpts()
	opts.ProviderID = new(uint32)
	*opts.ProviderID = 1

	require.NoError(t, config.Validate(cfg, opts))

	opts.Scrypt.N = 0
	require.ErrorContains(t, config.Validate(cfg, opts), "scrypt parameter N cannot be 0")
}
