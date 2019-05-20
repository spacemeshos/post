package persistence

import (
	"github.com/spacemeshos/go-spacemesh/filesystem"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/shared"
)

const (
	LabelGroupSize = shared.LabelGroupSize
)

type LabelGroup []byte

func GetPostDataPath() string {
	return filesystem.GetCanonicalPath(config.Post.DataFolder)
}
