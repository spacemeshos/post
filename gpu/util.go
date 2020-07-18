package gpu

func calcOutputSize(startPosition, endPosition uint64, hashLenBits uint8) uint64 {
	numPositions := endPosition - startPosition + 1
	outputSizeBits := numPositions * uint64(hashLenBits)
	outputSizeBytes := outputSizeBits / 8
	if outputSizeBits%8 > 0 {
		outputSizeBytes++
	}

	return outputSizeBytes
}
