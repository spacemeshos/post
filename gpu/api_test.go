package gpu

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScryptPositions(t *testing.T) {
	req := require.New(t)

	id := []byte("id")
	salt := []byte("salt")
	startPosition := uint64(1)
	endPosition := uint64(2048)
	hashLenBits := uint8(4)

	options := uint32(CPU)
	output, err := ScryptPositions(id, salt, startPosition, endPosition, options, hashLenBits)
	req.NoError(err)
	req.NotNil(output)
	req.Equal(1024, len(output))
}

//func TestStats(t *testing.T) {
//	req := require.New(t)
//
//	c := Stats()
//	req.True(c.CPU)
//	req.False(c.GPUCuda)
//	req.False(c.GPUOpenCL)
//	req.True(c.GPUVulkan)
//}
//
//func TestGPUCount(t *testing.T) {
//	req := require.New(t)
//
//	count := GPUCount(GPUVulkan, true)
//	req.Equal(2, count)
//}
