package config_test

import (
	"testing"

	"github.com/spacemeshos/post/config"
	"github.com/stretchr/testify/require"
)

func TestOptsValidateScryptParams(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	opts := config.DefaultInitOpts()

	require.Nil(t, config.Validate(cfg, opts))

	opts.Scrypt.N = 0
	require.Error(t, config.Validate(cfg, opts))
}
