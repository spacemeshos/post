package initialization

// metadata is the data associated with the PoST init procedure, persisted in the datadir next to the init files.
type metadata struct {
	ID        []byte
	NumLabels uint64
	NumFiles  uint
	LabelSize uint
}

const metadataFileName = "init.json"
