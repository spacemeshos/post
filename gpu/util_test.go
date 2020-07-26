package gpu

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScryptOutputSize(t *testing.T) {
	req := require.New(t)

	req.Equal(uint64(0), calcOutputSize(0, 0, 0))
	req.Equal(uint64(1), calcOutputSize(0, 0, 1))
	req.Equal(uint64(1), calcOutputSize(0, 0, 4))
	req.Equal(uint64(1), calcOutputSize(0, 0, 5))

	req.Equal(uint64(0), calcOutputSize(0, 1, 0))
	req.Equal(uint64(1), calcOutputSize(0, 1, 1))
	req.Equal(uint64(1), calcOutputSize(0, 1, 4))
	req.Equal(uint64(2), calcOutputSize(0, 1, 5))
}
