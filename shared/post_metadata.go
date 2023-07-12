package shared

import (
	"encoding/hex"
	"fmt"
)

// PostMetadata is the data associated with the PoST init procedure, persisted in the datadir next to the init files.
type PostMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	LabelsPerUnit uint64
	NumUnits      uint32
	MaxFileSize   uint64
	Nonce         *uint64    `json:",omitempty"`
	NonceValue    NonceValue `json:",omitempty"`
	LastPosition  *uint64    `json:",omitempty"`
}

type NonceValue []byte

// Unmarshal JSON from hex format.
func (n *NonceValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	// trim quotes
	data = data[1 : len(data)-1]
	*n = make([]byte, hex.DecodedLen(len(data)))
	_, err := hex.Decode(*n, data)
	return err
}

// Marshal to JSON in hex format.
func (n NonceValue) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte{}, nil
	}
	dst := make([]byte, hex.EncodedLen(len(n)))
	hex.Encode(dst, n[:])
	return []byte(fmt.Sprintf("\"%s\"", dst)), nil
}
