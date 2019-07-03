package validation

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

var (
	tempdir, _ = ioutil.TempDir("", "post-test")
	id         = hexDecode("deadbeef")
	challenge  = hexDecode("this is a challenge")
	logger     = shared.DisabledLogger{}
	cfg        = &Config{
		SpacePerUnit:                            16 * shared.LabelGroupSize,
		FileSize:                                16 * shared.LabelGroupSize,
		Difficulty:                              5,
		NumProvenLabels:                         4,
		LowestLayerToCacheDuringProofGeneration: 0,
		DataDir:                                 tempdir,
		LabelsLogRate:                           uint64(math.MaxUint64),
	}
	NewInitializer = initialization.NewInitializer
	NewProver      = proving.NewProver
)

func TestValidate(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	err = NewValidator(cfg).Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, cfg)
}

func TestValidate2(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	newCfg := *cfg
	newCfg.Difficulty = 6

	proof, err := NewInitializer(&newCfg, logger).Initialize(id)
	r.NoError(err)

	err = NewValidator(&newCfg).Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, &newCfg)
}

func TestValidate3(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	newCfg := *cfg
	newCfg.Difficulty = 7

	proof, err := NewInitializer(&newCfg, logger).Initialize(id)
	r.NoError(err)

	err = NewValidator(&newCfg).Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, &newCfg)
}

func TestValidate4(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	newCfg := *cfg
	newCfg.Difficulty = 8

	proof, err := NewInitializer(&newCfg, logger).Initialize(id)
	r.NoError(err)

	err = NewValidator(&newCfg).Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, &newCfg)
}

func TestValidateBadDifficulty(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	newCfg := *cfg
	newCfg.Difficulty = 4

	err := NewValidator(&newCfg).Validate(new(proving.Proof))
	r.EqualError(err, fmt.Sprintf("difficulty must be between 5 and 8 (received %d)", newCfg.Difficulty))
}

func testGenerateProof(r *require.Assertions, id []byte, cfg *Config) {
	proof, err := NewProver(cfg, logger).GenerateProof(id, challenge)
	r.NoError(err)

	err = NewValidator(cfg).Validate(proof)
	r.Nil(err)
}

func TestGenerateProofFailure(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	newCfg := *cfg
	newCfg.Difficulty = 4

	_, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)
	proof, err := NewProver(&newCfg, logger).GenerateProof(id, challenge)
	r.EqualError(err, fmt.Sprintf("proof generation failed: difficulty must be between 5 and 8 (received %d)", newCfg.Difficulty))
	r.Empty(proof)
}

func TestValidateFail(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	proof.Identity = append([]byte{0}, proof.Identity[1:]...)

	err = NewValidator(cfg).Validate(proof)
	r.EqualError(err, "validation failed: label at index 91 should be 01101111, but found 00011101")
}

func TestValidateFail2(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	proof.Challenge = []byte{1}

	err = NewValidator(cfg).Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail3(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	proof.ProvenLeaves[0] = append([]byte{}, proof.ProvenLeaves[0]...)
	proof.ProvenLeaves[0][0] += 1

	err = NewValidator(cfg).Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail4(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	proof.ProvenLeaves = proof.ProvenLeaves[1:]

	err = NewValidator(cfg).Validate(proof)
	r.EqualError(err, "validation failed: number of derived leaf indices (4) doesn't match number of included proven leaves (3)")
}

func TestValidateFail5(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	proof.ProofNodes[0] = append([]byte{}, proof.ProofNodes[0]...)
	proof.ProofNodes[0][0] += 1

	err = NewValidator(cfg).Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail6(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	proof.ProofNodes = proof.ProofNodes[1:]

	err = NewValidator(cfg).Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")
}

func TestValidateFail7(t *testing.T) {
	r := require.New(t)
	defer cleanup()

	proof, err := NewInitializer(cfg, logger).Initialize(id)
	r.NoError(err)

	proof.MerkleRoot = append([]byte{}, proof.MerkleRoot...)
	proof.MerkleRoot[0] += 1

	err = NewValidator(cfg).Validate(proof)
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
	_ = os.RemoveAll(tempdir)
}
