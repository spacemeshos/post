package persistence

import (
	"github.com/spacemeshos/go-spacemesh/filesystem"
	"github.com/spacemeshos/post/config"
)

const (
	filename = "all.labels"
)

func GetPostDataPath() string {
	return filesystem.GetCanonicalPath(config.Post.DataFolder)
}
