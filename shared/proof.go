package shared

type Proof struct {
	Nonce   uint32
	Indices []byte
}

type ProofMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	Challenge     Challenge
	NumUnits      uint32
	BitsPerLabel  uint8
	LabelsPerUnit uint64
	K1            uint32
	K2            uint32
}
