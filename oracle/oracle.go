package oracle

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/shared"
)

type Challenge = shared.Challenge

func WorkOracle(computeProviderId uint, id []byte, startPosition, endPosition uint64, bitsPerLabel uint32) ([]byte, error) {
	salt := make([]byte, 32) // TODO(moshababo): apply salt

	res, err := gpu.ScryptPositions(computeProviderId,
		gpu.WithCommitment(id),
		gpu.WithSalt(salt),
		gpu.WithStartAndEndPosition(startPosition, endPosition),
		gpu.WithBitsPerLabel(bitsPerLabel),
	)
	if err != nil {
		return nil, err
	}

	return res.Output, nil
}

func WorkOracleOne(id []byte, position uint64, bitsPerLabel uint32) []byte {
	cpuProviderID := uint(gpu.CPUProviderID())
	output, err := WorkOracle(cpuProviderID, id, position, position, bitsPerLabel)
	if err != nil {
		panic(err)
	}

	return output
}

func FastOracle(ch Challenge, nonce uint32, label []byte) [32]byte {
	input := make([]byte, 32+4+len(label))

	copy(input, ch)
	binary.LittleEndian.PutUint32(input[32:], nonce)
	copy(input[36:], label[:])

	return sha256.Sum256(input)
}
