package oracle

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/shared"
)

type (
	Challenge = shared.Challenge
)

func WorkOracle(computeProviderId uint, id []byte, startPosition, endPosition uint64, bitsPerLabel uint32) ([]byte, error) {
	salt := make([]byte, 32) // TODO(moshababo): apply salt

	res, err := gpu.ScryptPositions(computeProviderId, id, salt, startPosition, endPosition, bitsPerLabel)
	if err != nil {
		return nil, err
	}

	return res.Output, nil
}

func WorkOracleOne(cpuProviderID uint, id []byte, position uint64, bitsPerLabel uint32) []byte {
	salt := make([]byte, 32) // TODO(moshababo): apply salt

	res, err := gpu.ScryptPositions(cpuProviderID, id, salt, position, position, bitsPerLabel)
	if err != nil {
		panic(err)
	}

	return res.Output

	/*
		// A template for an alternative Go implementation:

		input := make([]byte, len(id)+binary.Size(position))
		copy(input, id)
		binary.LittleEndian.PutUint64(input[len(id):], position)
		output := scrypt(input)
		return output[:labelSize/8] // Must also include the last (bitsPerLabel%8) bits as an additional byte.
	*/
}

func FastOracle(ch Challenge, nonce uint32, label []byte) [32]byte {
	input := make([]byte, 32+4+len(label))

	copy(input, ch)
	binary.LittleEndian.PutUint32(input[32:], nonce)
	copy(input[36:], label[:])

	return sha256.Sum256(input)
}
