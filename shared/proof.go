package shared

type Proof struct {
	Nonce   uint32
	Indices []byte
	K2Pow   uint64
	K3Pow   uint64
}

type ProofMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	Challenge     Challenge
	NumUnits      uint32
	LabelsPerUnit uint64
}

type VRFNonce uint64

type VRFNonceMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	NumUnits      uint32
	LabelsPerUnit uint64
}
