package validation

import (
	"encoding/hex"
	"github.com/spacemeshos/post-private/initialization"
	"github.com/spacemeshos/post-private/proving"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidate(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	err = Validate(proof, 16, 4, difficulty)
	r.Nil(err)

	testGenerateProof(r, id, difficulty)
}

func TestValidate2(t *testing.T) {
	r := require.New(t)

	const difficulty = 6
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	err = Validate(proof, 16, 4, difficulty)
	r.Nil(err)

	testGenerateProof(r, id, difficulty)
}

func TestValidate3(t *testing.T) {
	r := require.New(t)

	const difficulty = 7
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	err = Validate(proof, 16, 4, difficulty)
	r.Nil(err)

	testGenerateProof(r, id, difficulty)
}

func TestValidate4(t *testing.T) {
	r := require.New(t)

	const difficulty = 8
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	err = Validate(proof, 16, 4, difficulty)
	r.Nil(err)

	testGenerateProof(r, id, difficulty)
}

func testGenerateProof(r *require.Assertions, id []byte, difficulty proving.Difficulty) {
	challenge := proving.Challenge{1, 2, 3}
	proof2, err := proving.GenerateProof(id, challenge, 4, difficulty)
	r.NoError(err)

	err = Validate(proof2, 16, 4, difficulty)
	r.Nil(err)
}

func TestValidateFail(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	proof.Identity[0] = 0

	err = Validate(proof, 16, 4, difficulty)
	r.EqualError(err, "validation failed: label at index 91 should be 01101111, but found 00011101")
}

func TestValidateFail2(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	proof.Challenge = []byte{1}

	err = Validate(proof, 16, 4, difficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail3(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	proof.ProvenLeaves[0][0] += 1

	err = Validate(proof, 16, 4, difficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail4(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	proof.ProvenLeaves = proof.ProvenLeaves[1:]

	err = Validate(proof, 16, 4, difficulty)
	r.EqualError(err, "validation failed: number of derived leaf indices (4) doesn't match number of included proven leaves (3)")
}

func TestValidateFail5(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	proof.ProofNodes[0][0] += 1

	err = Validate(proof, 16, 4, difficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail6(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	proof.ProofNodes = proof.ProofNodes[1:]

	err = Validate(proof, 16, 4, difficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail7(t *testing.T) {
	r := require.New(t)

	const difficulty = 5
	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, 16, 4, difficulty)
	r.NoError(err)

	proof.MerkleRoot[0] += 1

	err = Validate(proof, 16, 4, difficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func hexDecode(hexStr string) []byte {
	node, _ := hex.DecodeString(hexStr)
	return node
}
