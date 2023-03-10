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
	BitsPerLabel  uint8
	LabelsPerUnit uint64

	K1 uint32 // K1 specifies the difficulty for a label to be a candidate for a proof.
	K2 uint32 // K2 is the number of labels below the required difficulty required for a proof.
	B  uint32 // B is the number of labels used per AES invocation when generating a proof. Lower values speed up verification, higher values proof generation.
	N  uint32 // N is the number of nonces tried at the same time when generating a proof.
	// TODO(mafa): N should probably either be derived from the other parameters or be a configuration option of the node.
}

type VRFNonce uint64

type VRFNonceMetadata struct {
	NodeId          []byte
	CommitmentAtxId []byte

	NumUnits      uint32
	BitsPerLabel  uint8
	LabelsPerUnit uint64
}
