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

	K1 uint32 // K1 specifies the difficulty for a label to be a candidate for a proof.
	K2 uint32 // K2 is the number of labels below the required difficulty required for a proof.
}

type VRFNonce uint64

type VRFNonceMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	NumUnits      uint32
	BitsPerLabel  uint8
	LabelsPerUnit uint64
}
