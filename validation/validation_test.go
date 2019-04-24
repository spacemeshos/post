package validation

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

const (
	defaultDifficulty           = 5
	defaultSpace                = proving.Space(16 * initialization.LabelGroupSize)
	defaultNumberOfProvenLabels = 4
)

var (
	defaultId        = hexDecode("deadbeef")
	defaultChallenge = hexDecode("this is a challenge")
)

func TestValidate(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, defaultDifficulty)
}

func TestValidate2(t *testing.T) {
	r := require.New(t)

	const difficulty = 6

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, difficulty)
}

func TestValidate3(t *testing.T) {
	r := require.New(t)

	const difficulty = 7

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, difficulty)
}

func TestValidate4(t *testing.T) {
	r := require.New(t)

	const difficulty = 8

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.NoError(err)

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.Nil(err)

	testGenerateProof(r, defaultId, difficulty)
}

func TestValidateBadDifficulty(t *testing.T) {
	r := require.New(t)

	const difficulty = 4

	err := Validate(proving.Proof{}, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.EqualError(err, fmt.Sprintf("difficulty must be between 5 and 8 (received %d)", difficulty))
}

func testGenerateProof(r *require.Assertions, id []byte, difficulty proving.Difficulty) {
	proof2, err := proving.GenerateProof(id, defaultChallenge, defaultNumberOfProvenLabels, difficulty)
	r.NoError(err)

	err = Validate(proof2, defaultSpace, defaultNumberOfProvenLabels, difficulty)
	r.Nil(err)
}

func TestGenerateProofFailure(t *testing.T) {
	r := require.New(t)

	const difficulty = 4

	proof, err := proving.GenerateProof(defaultId, defaultChallenge, defaultNumberOfProvenLabels, difficulty)
	r.EqualError(err, fmt.Sprintf("proof generation failed: difficulty must be between 5 and 8 (received %d)", difficulty))
	r.Empty(proof)
}

func TestValidateFail(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	proof.Identity[0] = 0

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: label at index 91 should be 01101111, but found 00011101")
}

func TestValidateFail2(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	proof.Challenge = []byte{1}

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail3(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	proof.ProvenLeaves[0][0] += 1

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail4(t *testing.T) {
	r := require.New(t)

	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	proof.ProvenLeaves = proof.ProvenLeaves[1:]

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: number of derived leaf indices (4) doesn't match number of included proven leaves (3)")
}

func TestValidateFail5(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	proof.ProofNodes[0][0] += 1

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail6(t *testing.T) {
	r := require.New(t)

	proof, err := initialization.Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	proof.ProofNodes = proof.ProofNodes[1:]

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail7(t *testing.T) {
	r := require.New(t)

	id := hexDecode("deadbeef")

	proof, err := initialization.Initialize(id, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
	r.NoError(err)

	proof.MerkleRoot[0] += 1

	err = Validate(proof, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
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
