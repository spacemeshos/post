package shared

import (
	"encoding/hex"
	"encoding/json"
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

func (n NonceValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(n))
}
func (n *NonceValue) UnmarshalJSON(data []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(data, &hexString); err != nil {
		return
	}
	*n, err = hex.DecodeString(hexString)
	return
}
