package shared

// PostMetadata is the data associated with the PoST init procedure, persisted in the datadir next to the init files.
type PostMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	BitsPerLabel  uint8
	LabelsPerUnit uint64
	NumUnits      uint32
	MaxFileSize   uint64
	Nonce         *uint64 `json:",omitempty"`
	LastPosition  *uint64 `json:",omitempty"`
}
