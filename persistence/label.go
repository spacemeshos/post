package persistence

import "encoding/hex"

type Label []byte

func (l Label) String() string {
	return hex.EncodeToString(l)[:5]
}
