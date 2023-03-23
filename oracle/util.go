package oracle

import "github.com/zeebo/blake3"

// CommitmentBytes returns the commitment bytes for the given Node ID and Commitment ATX ID.
func CommitmentBytes(nodeId, commitmentAtxId []byte) []byte {
	hh := blake3.New()
	hh.Write(nodeId)
	hh.Write(commitmentAtxId)
	return hh.Sum(nil)
}
