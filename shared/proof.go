package shared

import (
	"encoding/binary"
)

//go:generate scalegen -types Proof

type Proof struct {
	Nonce   uint32
	Indices []byte
}

// Encode encodes Proof according to the following format:
//
//	+-----------+-------------------+
//	| nonce     | indices           |
//	| (4 bytes) | (remaining bytes) |
//	+-----------+-------------------+
func (p *Proof) Encode() []byte {
	size := 4 + len(p.Indices)
	b := make([]byte, size)

	binary.LittleEndian.PutUint32(b, p.Nonce)
	copy(b[4:], p.Indices)

	return b
}

// Decode decodes []byte slice according to the encoding format
// defined in the `Encode` method.
// If completed successfully, the result is assigned to the
// method pointer receiver value, hence the previous value is overridden.
// This method is intended to be called on a zero-value instance.
func (p *Proof) Decode(data []byte) error {
	proof := Proof{}
	proof.Nonce = binary.LittleEndian.Uint32(data[:4])
	proof.Indices = data[4:]

	// Override the method pointer receiver value.
	*p = proof

	return nil
}

type ProofMetadata struct {
	Commitment    []byte
	Challenge     Challenge
	NumUnits      uint32
	BitsPerLabel  uint8
	LabelsPerUnit uint64
	K1            uint32
	K2            uint32
}
