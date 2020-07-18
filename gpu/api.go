package gpu

import (
	"fmt"
	"github.com/pkg/errors"
)

const n, r, p = 512, 1, 1

func ScryptPositions(id, salt []byte, startPosition, endPosition uint64, options uint32, hashLenBits uint8) ([]byte, error) {
	if hashLenBits < 1 || hashLenBits > 8 {
		return nil, fmt.Errorf("invalid hashLenBits value, expected range: 1-8, given: %v", hashLenBits)
	}

	outputSize := calcOutputSize(startPosition, endPosition, hashLenBits)

	output, retVal := cScryptPositions(id, salt, startPosition, endPosition, options, hashLenBits, outputSize, n, r, p)

	switch retVal {
	case 0:
		return output, nil
	case -1:
		return nil, errors.New("no available gpu")
	default:
		panic("unreachable")
	}
}

type Capabilities struct {
	CPU       bool
	GPUCuda   bool
	GPUOpenCL bool
	GPUVulkan bool
}

func Stats() Capabilities {
	s := cStats()

	c := Capabilities{}
	c.CPU = s&int(CPU) == int(CPU)
	c.GPUCuda = s&int(GPUCuda) == int(GPUCuda)
	c.GPUOpenCL = s&int(GPUOpenCL) == int(GPUOpenCL)
	c.GPUVulkan = s&int(GPUVulkan) == int(GPUVulkan)
	return c
}

func GPUCount(apiType APIType, onlyAvailable bool) int {
	return cGPUCount(apiType, onlyAvailable)
}
