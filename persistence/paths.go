package persistence

import (
	"github.com/spacemeshos/go-spacemesh/filesystem"
	"path/filepath"
)

const (
	dataPath = "post-data" // TODO @noam: put in config
	filename = "all.labels"
)

func GetPostDataPath() string {
	smData, err := filesystem.GetSpacemeshDataDirectoryPath()
	if err != nil {
		panic(err)
	}
	return filepath.Join(smData, dataPath)
}
