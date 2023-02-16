package verifying

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"unsafe"

	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

func VerifyNew(p *shared.Proof, m *shared.ProofMetadata, opts ...OptionFunc) error {
	options := &option{}
	for _, opt := range opts {
		opt(options)
	}
	if err := options.validate(); err != nil {
		return err
	}

	if (m.BitsPerLabel) != 8 {
		return fmt.Errorf("invalid `bitsPerLabel, only 8-bit label is supported, given: %v", m.BitsPerLabel)
	}
	if len(m.NodeId) != 32 {
		return fmt.Errorf("invalid `nodeId` length; expected: 32, given: %v", len(m.NodeId))
	}
	if len(m.CommitmentAtxId) != 32 {
		return fmt.Errorf("invalid `commitmentAtxId` length; expected: 32, given: %v", len(m.CommitmentAtxId))
	}

	numLabels := uint64(m.NumUnits) * uint64(m.LabelsPerUnit)
	bitsPerIndex := uint(shared.BinaryRepresentationMinBits(numLabels))
	expectedSize := shared.Size(bitsPerIndex, uint(m.K2))
	if expectedSize != uint(len(p.Indices)) {
		return fmt.Errorf("invalid indices set size; expected %d, given: %d", expectedSize, len(p.Indices))
	}

	if options.verifyFunc == nil {
		difficulty := shared.ProvingDifficulty(numLabels, uint64(m.K1))
		options.verifyFunc = func(val uint64) bool {
			return val < difficulty
		}
	}

	buf := bytes.NewBuffer(p.Indices)
	gsReader := shared.NewGranSpecificReader(buf, bitsPerIndex)
	indicesSet := make(map[uint64]struct{}, m.K2)

	for i := uint(0); i < uint(m.K2); i++ {
		index, err := gsReader.ReadNextUintBE()
		if err != nil {
			return err
		}
		if _, ok := indicesSet[index]; ok {
			return fmt.Errorf("non-unique index: %d", index)
		}
		indicesSet[index] = struct{}{}

		// Recreate B-long labels block
		labelStart := index * uint64(m.B)
		labelEnd := labelStart + uint64(m.B) - 1
		res, err := WorkOracle(
			oracle.WithCommitment(oracle.CommitmentBytes(m.NodeId, m.CommitmentAtxId)),
			oracle.WithStartAndEndPosition(labelStart, labelEnd),
			oracle.WithBitsPerLabel(uint32(m.BitsPerLabel)),
		)
		if err != nil {
			return err
		}

		// cipher with the keys needed to extract the relevant nonce
		d := uint32(oracle.CalcD(numLabels, m.B))
		offset := p.Nonce * d
		nonceBlock := uint8(offset / aes.BlockSize)

		ciphers := make([]cipher.Block, 2)
		for i := uint8(0); i < 2; i++ {
			c, err := oracle.CreateBlockCipher(m.Challenge, nonceBlock+i)
			if err != nil {
				return fmt.Errorf("creating cipher for block %d: %w", nonceBlock, err)
			}
			ciphers[i] = c
		}

		out := make([]byte, aes.BlockSize*2)
		u64 := unsafe.Slice((*uint64)(unsafe.Pointer(&out[offset%aes.BlockSize])), 1)
		mask := (uint64(1) << (d * 8)) - 1
		ciphers[0].Encrypt(out[:aes.BlockSize], res.Output)
		ciphers[1].Encrypt(out[aes.BlockSize:], res.Output)

		val := u64[0] & mask
		if !options.verifyFunc(val) {
			return fmt.Errorf(
				"fast oracle output is doesn't pass difficulty check; index: %d, labels block: %x, value: %d", index, res.Output, val)
		}
	}

	return nil
}
