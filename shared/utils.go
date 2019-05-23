package shared

import (
	"encoding/hex"
	"path/filepath"
)

func GetDir(datadir string, id []byte) string {
	return filepath.Join(datadir, hex.EncodeToString(id))
}
