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

	K1 uint32 // K1 specifies the difficulty for a label to be a candidate for a proof.
	K2 uint32 // K2 is the number of labels below the required difficulty required for a proof.
	B  uint32 // B is the number of labels used per AES invocation when generating a proof.
	N  uint32 // N is the number of nonces tried at the same time when generating a proof.
}

type VRFNonce uint64

type VRFNonceMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	NumUnits      uint32
	BitsPerLabel  uint8
	LabelsPerUnit uint64
}
