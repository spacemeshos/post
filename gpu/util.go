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

func cStringArrayToGoString(src [256]cChar) string {
	var dst []byte
	for i := 0; i < 256; i++ {
		if src[i] == 0 {
			break
		}
		dst = append(dst, byte(src[i]))
	}
	return string(dst)
}
