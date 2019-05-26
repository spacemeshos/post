package validation

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

const (
	defaultDifficulty        = 5
	defaultSpace             = 16 * shared.LabelGroupSize
	defaultNumOfProvenLabels = 4
)

var (
	defaultId        = hexDecode("deadbeef")
	defaultChallenge = hexDecode("this is a challenge")
)

func TestValidate(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, defaultDifficulty)
}

func TestValidate2(t *testing.T) {
	r := require.New(t)

	const difficulty = 6

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, difficulty, false)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, difficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, difficulty)
}

func TestValidate3(t *testing.T) {
	r := require.New(t)

	const difficulty = 7

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, difficulty, false)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, difficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, difficulty)
}

func TestValidate4(t *testing.T) {
	r := require.New(t)

	const difficulty = 8

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, difficulty, false)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, difficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, difficulty)
}

func TestValidateBadDifficulty(t *testing.T) {
	r := require.New(t)

	const difficulty = 4

	err := Validate(new(proving.Proof), defaultSpace, defaultNumOfProvenLabels, difficulty)
	r.EqualError(err, fmt.Sprintf("difficulty must be between 5 and 8 (received %d)", difficulty))
}

func testGenerateProof(r *require.Assertions, id []byte, difficulty proving.Difficulty) {
	proof2, err := proving.GenerateProof(id, defaultChallenge, defaultNumOfProvenLabels, difficulty)
	r.NoError(err)

	err = Validate(&proof2, defaultSpace, defaultNumOfProvenLabels, difficulty)
	r.Nil(err)
}

func TestGenerateProofFailure(t *testing.T) {
	r := require.New(t)

	const difficulty = 4

	proof, err := proving.GenerateProof(defaultId, defaultChallenge, defaultNumOfProvenLabels, difficulty)
	r.EqualError(err, fmt.Sprintf("proof generation failed: difficulty must be between 5 and 8 (received %d)", difficulty))
	r.Empty(proof)
}

func TestValidateFail(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	proof.Identity = append([]byte{0}, proof.Identity[1:]...)

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: label at index 91 should be 01101111, but found 00011101")
}

func TestValidateFail2(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	proof.Challenge = []byte{1}

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail3(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	proof.ProvenLeaves[0] = append([]byte{}, proof.ProvenLeaves[0]...)
	proof.ProvenLeaves[0][0] += 1

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail4(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	proof.ProvenLeaves = proof.ProvenLeaves[1:]

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: number of derived leaf indices (4) doesn't match number of included proven leaves (3)")
}

func TestValidateFail5(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	proof.ProofNodes[0] = append([]byte{}, proof.ProofNodes[0]...)
	proof.ProofNodes[0][0] += 1

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail6(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	proof.ProofNodes = proof.ProofNodes[1:]

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail7(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false)
	r.NoError(err)

	proof.MerkleRoot = append([]byte{}, proof.MerkleRoot...)
	proof.MerkleRoot[0] += 1

	err = Validate(proof, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func hexDecode(hexStr string) []byte {
	node, _ := hex.DecodeString(hexStr)
	return node
}

func TestMain(m *testing.M) {
	flag.Parse()
	res := m.Run()
	cleanup()
	os.Exit(res)
}

func cleanup() {
	_ = os.RemoveAll(filepath.Join(persistence.GetPostDataPath(), "deadbeef"))
}
