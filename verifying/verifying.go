package verifying

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

// VerifyVRFNonce ensures the validity of a nonce for a given node.
// AtxId is the id of the ATX that was selected by the node for its commitment.
func VerifyVRFNonce(nonce *uint64, m *shared.VRFNonceMetadata) error {
	if nonce == nil {
		return errors.New("invalid `nonce` value; expected: non-nil, given: nil")
	}

	if len(m.NodeId) != 32 {
		return fmt.Errorf("invalid `nodeId` length; expected: 32, given: %v", len(m.NodeId))
	}

	if len(m.CommitmentAtxId) != 32 {
		return fmt.Errorf("invalid `commitmentAtxId` length; expected: 32, given: %v", len(m.CommitmentAtxId))
	}

	numLabels := uint64(m.NumUnits) * uint64(m.LabelsPerUnit)
	difficulty := shared.PowDifficulty(numLabels)
	threshold := new(big.Int).SetBytes(difficulty)

	res, err := oracle.WorkOracle(
		oracle.WithCommitment(oracle.CommitmentBytes(m.NodeId, m.CommitmentAtxId)),
		oracle.WithPosition(*nonce),
		oracle.WithBitsPerLabel(uint32(m.BitsPerLabel)*32),
	)
	if err != nil {
		return err
	}

	label := new(big.Int).SetBytes(res.Output)
	if label.Cmp(threshold) > 0 {
		return fmt.Errorf("label is above the threshold; label: %#32x, threshold: %#32x", label, threshold)
	}

	return nil
}

// Verify ensures the validity of a proof in respect to its metadata.
// It returns nil if the proof is valid or an error describing the failure, otherwise.
func Verify(p *shared.Proof, m *shared.ProofMetadata, opts ...OptionFunc) error {
	options := &option{
		logger: &shared.DisabledLogger{},
	}
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
		difficulty := shared.ProvingDifficulty(numLabels, m.B, m.K1)
		options.logger.Debug("verifying difficulty %d", difficulty)
		options.verifyFunc = func(val uint64) bool {
			return val < difficulty
		}
	}

	buf := bytes.NewBuffer(p.Indices)
	gsReader := shared.NewGranSpecificReader(buf, bitsPerIndex)
	indicesSet := make(map[uint64]struct{}, m.K2)

	// create the ciphers for the specific nonce
	offset := p.Nonce * 8
	nonceBlock := uint8(offset / aes.BlockSize)

	// since the value can be on a boundary between two blocks, we need to create two ciphers
	ciphers := make([]cipher.Block, 2)
	for i := uint8(0); i < 2; i++ {
		c, err := oracle.CreateBlockCipher(m.Challenge, nonceBlock+i)
		if err != nil {
			return fmt.Errorf("creating cipher for block %d: %w", nonceBlock, err)
		}
		ciphers[i] = c
	}

	block := make([]byte, aes.BlockSize)
	out := make([]byte, aes.BlockSize*2)
	u64 := (*uint64)(unsafe.Pointer(&out[offset%aes.BlockSize]))

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
		res, err := oracle.WorkOracle(
			oracle.WithCommitment(oracle.CommitmentBytes(m.NodeId, m.CommitmentAtxId)),
			oracle.WithStartAndEndPosition(labelStart, labelEnd),
			oracle.WithBitsPerLabel(uint32(m.BitsPerLabel)),
		)
		if err != nil {
			return err
		}
		copy(block, res.Output)

		ciphers[0].Encrypt(out[:aes.BlockSize], block)
		ciphers[1].Encrypt(out[aes.BlockSize:], block)

		val := *u64
		options.logger.Debug("verifying: index %d value %d", index, val)
		if !options.verifyFunc(val) {
			return fmt.Errorf("fast oracle output is doesn't pass difficulty check; index: %d, labels block: %x, value: %d", index, res.Output, val)
		}
	}

	return nil
}
