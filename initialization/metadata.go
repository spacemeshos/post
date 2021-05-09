package initialization

// metadata is the data associated with the PoST init procedure, persisted in the datadir next to the init files.
type metadata struct {
	ID            []byte
	BitsPerLabel  uint
	LabelsPerUnit uint
	NumUnits      uint
	NumFiles      uint
}

const metadataFileName = "init.json"
