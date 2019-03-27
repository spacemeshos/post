package proving

type Proof struct {
	MerkleRoot    []byte
	ProofNodes    [][]byte
	ProvenLeaves  [][]byte
	ProvenIndices []uint64
}
