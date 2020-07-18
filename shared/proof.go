package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type Proof struct {
	Challenge Challenge
	Nonce     uint32
	Indices   []byte
}

// Encode encodes Proof according to the following format:
//
// +------------+-----------+-------------------+
// | challenge  | nonce     | indices           |
// | (32 bytes) | (4 bytes) | (remaining bytes) |
// +------------+-----------+-------------------+
//
func (p *Proof) Encode() []byte {
	size := 32 + 4 + len(p.Indices)
	b := make([]byte, size)

	copy(b, p.Challenge)
	binary.LittleEndian.PutUint32(b[32:], p.Nonce)
	copy(b[36:], p.Indices[:])

	return b
}

// Decode decodes []byte slice according to the encoding format
// defined in the `Encode` method.
// If completed successfully, the result is assigned to the
// method pointer receiver value, hence the previous value is overridden.
// This method is intended to be called on a zero-value instance.
func (p *Proof) Decode(data []byte) error {
	const minIndicesLen = 10 // TODO(moshababo): implement
	if len(data) < 32+4+minIndicesLen {
		return errors.New("invalid input: too short")
	}

	buf := bytes.NewBuffer(data)
	proof := Proof{}

	proof.Challenge = buf.Next(32)
	proof.Nonce = binary.LittleEndian.Uint32(buf.Next(4))
	proof.Indices = buf.Bytes() // TODO(moshababo): verify expected length

	// Override the method pointer receiver value.
	*p = proof

	return nil
}
