package shared

type Proof struct {
	Challenge    Challenge
	MerkleRoot   []byte
	ProofNodes   [][]byte
	ProvenLeaves [][]byte
}
