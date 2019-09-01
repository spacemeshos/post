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
	cfg        = &Config{
		SpacePerUnit:                            16 * shared.LabelGroupSize,
		NumFiles:                                1,
		Difficulty:                              5,
		NumProvenLabels:                         4,
		LowestLayerToCacheDuringProofGeneration: 0,
		DataDir:                                 tempdir,
		LabelsLogRate:                           uint64(math.MaxUint64),
	}
	NewInitializer = initialization.NewInitializer
	NewProver      = proving.NewProver
)

func TestMain(m *testing.M) {
	flag.Parse()
	res := m.Run()
	os.Exit(res)
}

func TestValidate(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, cfg)

	err = init.Reset()
	r.NoError(err)
}

func TestValidate2(t *testing.T) {
	r := require.New(t)

	newCfg := *cfg
	newCfg.Difficulty = 6

	init, err := NewInitializer(&newCfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	v, err := NewValidator(&newCfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, &newCfg)

	err = init.Reset()
	r.NoError(err)
}

func TestValidate3(t *testing.T) {
	r := require.New(t)

	newCfg := *cfg
	newCfg.Difficulty = 7

	init, err := NewInitializer(&newCfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	v, err := NewValidator(&newCfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, &newCfg)

	err = init.Reset()
	r.NoError(err)
}

func TestValidate4(t *testing.T) {
	r := require.New(t)

	newCfg := *cfg
	newCfg.Difficulty = 8

	init, err := NewInitializer(&newCfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	v, err := NewValidator(&newCfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.Nil(err)

	testGenerateProof(r, id, &newCfg)

	err = init.Reset()
	r.NoError(err)
}

func TestValidateBadDifficulty(t *testing.T) {
	r := require.New(t)

	newCfg := *cfg
	newCfg.Difficulty = 4

	v, err := NewValidator(&newCfg)
	r.Nil(v)
	r.EqualError(err, fmt.Sprintf("difficulty must be between 5 and 8 (received %d)", newCfg.Difficulty))
}

func testGenerateProof(r *require.Assertions, id []byte, cfg *Config) {
	p, err := NewProver(cfg, id)
	r.NoError(err)
	proof, err := p.GenerateProof(id)
	r.NoError(err)

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.Nil(err)
}

func TestGenerateProofFailure(t *testing.T) {
	r := require.New(t)

	newCfg := *cfg
	newCfg.Difficulty = 6

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	_, err = init.Initialize()
	r.NoError(err)

	p, err := NewProver(&newCfg, id)
	r.NoError(err)
	proof, err := p.GenerateProof(challenge)
	r.EqualError(err, "proof generation failed: initialization state error: config mismatch")
	r.Empty(proof)

	err = init.Reset()
	r.NoError(err)
}

func TestValidateFail(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	proof.Identity = append([]byte{0}, proof.Identity[1:]...)

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.EqualError(err, "validation failed: label at index 91 should be 01101111, but found 00011101")

	err = init.Reset()
	r.NoError(err)
}

func TestValidateFail2(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	proof.Challenge = []byte{1}

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")

	err = init.Reset()
	r.NoError(err)
}

func TestValidateFail3(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	proof.ProvenLeaves[0] = append([]byte{}, proof.ProvenLeaves[0]...)
	proof.ProvenLeaves[0][0] += 1

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")

	err = init.Reset()
	r.NoError(err)
}

func TestValidateFail4(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	proof.ProvenLeaves = proof.ProvenLeaves[1:]

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.EqualError(err, "validation failed: number of derived leaf indices (4) doesn't match number of included proven leaves (3)")

	err = init.Reset()
	r.NoError(err)
}

func TestValidateFail5(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	proof.ProofNodes[0] = append([]byte{}, proof.ProofNodes[0]...)
	proof.ProofNodes[0][0] += 1

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")

	err = init.Reset()
	r.NoError(err)
}

func TestValidateFail6(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	proof.ProofNodes = proof.ProofNodes[1:]

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")

	err = init.Reset()
	r.NoError(err)
}

func TestValidateFail7(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	proof, err := init.Initialize()
	r.NoError(err)

	proof.MerkleRoot = append([]byte{}, proof.MerkleRoot...)
	proof.MerkleRoot[0] += 1

	v, err := NewValidator(cfg)
	r.NoError(err)
	err = v.Validate(proof)
	r.EqualError(err, "validation failed: merkle root mismatch")

	err = init.Reset()
	r.NoError(err)
}

func hexDecode(hexStr string) []byte {
	node, _ := hex.DecodeString(hexStr)
	return node
}
