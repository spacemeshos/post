package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type Proof struct {
	Nonce   uint32
	Indices []byte
}

// Encode encodes Proof according to the following format:
//
// +-----------+-------------------+
// | nonce     | indices           |
// | (4 bytes) | (remaining bytes) |
// +-----------+-------------------+
//
func (p *Proof) Encode() []byte {
	size := 4 + len(p.Indices)
	b := make([]byte, size)

	binary.LittleEndian.PutUint32(b, p.Nonce)
	copy(b[4:], p.Indices[:])

	return b
}

// Decode decodes []byte slice according to the encoding format
// defined in the `Encode` method.
// If completed successfully, the result is assigned to the
// method pointer receiver value, hence the previous value is overridden.
// This method is intended to be called on a zero-value instance.
func (p *Proof) Decode(data []byte) error {
	const minIndicesLen = 10 // TODO(moshababo): implement (after applying bit granularity)
	if len(data) < 4+minIndicesLen {
		return errors.New("invalid input: too short")
	}
	buf := bytes.NewBuffer(data)

	proof := Proof{}
	proof.Nonce = binary.LittleEndian.Uint32(buf.Next(4))
	proof.Indices = buf.Bytes()

	// Override the method pointer receiver value.
	*p = proof

	return nil
}

type ProofMetadata struct {
	Challenge Challenge
	NumLabels uint64
	LabelSize uint
	K1        uint
	K2        uint
}
