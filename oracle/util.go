package oracle

import "github.com/spacemeshos/sha256-simd"

// CommitmentBytes returns the commitment bytes for the given Node ID and Commitment ATX ID.
func CommitmentBytes(nodeId, commitmentAtxId []byte) []byte {
	hh := sha256.New()
	hh.Write(nodeId)
	hh.Write(commitmentAtxId)
	return hh.Sum(nil)
}
