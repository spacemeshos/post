package initialization

import "github.com/spacemeshos/post/config"

// metadata is the data associated with the PoST init procedure, persisted in the datadir next to the init files.
type metadata struct {
	Cfg   config.Config
	ID    []byte
	State metadataInitState
}

const metadataFileName = ".init"

type metadataInitState int

const (
	MetadataInitStateStarted metadataInitState = 1 + iota
	MetadataInitStateCompleted
	MetadataInitStateStopped
)
