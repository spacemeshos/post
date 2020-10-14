package validation

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

var (
	id             = make([]byte, 32)
	cfg            *Config
	NewInitializer = initialization.NewInitializer
	CPUProviderID  = initialization.CPUProviderID()
	NewProver      = proving.NewProver
)

// TestLabelsCorrectness tests, for variation of label sizes, the correctness of
// reading labels from disk (written in multiple files) when compared to a single label compute.
// It is covers the following components: labels compute lib (oracle pkg), labels writer (persistence pkg),
// labels reader (persistence pkg), and the granularity-specific reader (shared pkg).
// it proceeds as follows:
// 1. Compute labels, in batches, and write them into multiple files (prover).
// 2. Read the sequence of labels from the files according to the specified label size (prover),
//    and ensure that each one equals a single label compute (verifier).
func TestLabelsCorrectness(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	req := require.New(t)

	numFiles := 2
	numFileBatches := 2
	batchSize := 256
	id := make([]byte, 32)
	datadir, _ := ioutil.TempDir("", "post-test")

	for labelSize := uint32(config.MinLabelSize); labelSize <= config.MaxLabelSize; labelSize++ {
		// Write.
		for i := 0; i < numFiles; i++ {
			writer, err := persistence.NewLabelsWriter(datadir, id, i, uint(labelSize))
			req.NoError(err)
			for j := 0; j < numFileBatches; j++ {
				numBatch := i*numFileBatches + j
				startPosition := uint64(numBatch * batchSize)
				endPosition := startPosition + uint64(batchSize) - 1

				labels, err := oracle.WorkOracle(2, id, startPosition, endPosition, labelSize)
				req.NoError(err)
				err = writer.Write(labels)
				req.NoError(err)

			}
			_, err = writer.Close()
			req.NoError(err)
		}

		// Read.
		reader, err := persistence.NewLabelsReader(datadir, id, uint(labelSize))
		gsReader := shared.NewGranSpecificReader(reader, uint(labelSize))
		req.NoError(err)
		var position uint64
		for {
			label, err := gsReader.ReadNext()
			if err != nil {
				if err == io.EOF {
					req.Equal(uint64(numFiles*numFileBatches*batchSize), position)
					break
				}
				req.Fail(err.Error())
			}

			// Verify correctness.
			labelCompute := oracle.WorkOracleOne(CPUProviderID, id, position, labelSize)
			req.Equal(labelCompute, label, fmt.Sprintf("position: %v, labelSize: %v", position, labelSize))

			position++
		}
		_ = os.RemoveAll(datadir)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	cfg = config.DefaultConfig()
	cfg.NumLabels = 1 << 15
	cfg.LabelSize = 8
	cfg.DataDir, _ = ioutil.TempDir("", "post-test")

	res := m.Run()
	os.Exit(res)
}

func TestValidate(t *testing.T) {
	req := require.New(t)

	init, err := NewInitializer(cfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID)
	req.NoError(err)

	p, err := NewProver(cfg, id)
	req.NoError(err)
	proof, proofMetadata, err := p.GenerateProof(id)
	req.NoError(err)
	err = Validate(id, proof, proofMetadata)
	req.NoError(err)

	err = init.Reset()
	req.NoError(err)
}

func testGenerateProof(r *require.Assertions, id []byte, cfg *Config) {
	p, err := NewProver(cfg, id)
	r.NoError(err)
	p.SetLogger(log.AppLog)

	proof, proofMetadata, err := p.GenerateProof(id)
	r.NoError(err)

	err = Validate(id, proof, proofMetadata)
	r.NoError(err)
}

//func TestValidate2(t *testing.T) {
//	r := require.New(t)
//
//	newCfg := *cfg
//	newCfg.Difficulty = 6
//
//	init, err := NewInitializer(&newCfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	v, err := NewValidator(&newCfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.Nil(err)
//
//	testGenerateProof(r, id, &newCfg)
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidate3(t *testing.T) {
//	r := require.New(t)
//
//	newCfg := *cfg
//	newCfg.Difficulty = 7
//
//	init, err := NewInitializer(&newCfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	v, err := NewValidator(&newCfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.Nil(err)
//
//	testGenerateProof(r, id, &newCfg)
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidate4(t *testing.T) {
//	r := require.New(t)
//
//	newCfg := *cfg
//	newCfg.Difficulty = 8
//
//	init, err := NewInitializer(&newCfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	v, err := NewValidator(&newCfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.Nil(err)
//
//	testGenerateProof(r, id, &newCfg)
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateBadDifficulty(t *testing.T) {
//	r := require.New(t)
//
//	newCfg := *cfg
//	newCfg.Difficulty = 4
//
//	v, err := NewValidator(&newCfg)
//	r.Nil(v)
//	r.EqualError(err, fmt.Sprintf("difficulty must be between 5 and 8 (received %d)", newCfg.Difficulty))
//}
//

//
//func TestGenerateProofFailure(t *testing.T) {
//	r := require.New(t)
//
//	newCfg := *cfg
//	newCfg.Difficulty = 6
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	_, err = init.Initialize()
//	r.NoError(err)
//
//	p, err := NewProver(&newCfg, id)
//	r.NoError(err)
//	proof, err := p.GenerateProof(challenge)
//	r.EqualError(err, "proof generation failed: initialization state error: config mismatch")
//	r.Empty(proof)
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateFail(t *testing.T) {
//	r := require.New(t)
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	wrongIdentity := append([]byte{0}, id[1:]...)
//
//	v, err := NewValidator(cfg)
//	r.NoError(err)
//	err = v.Validate(wrongIdentity, proof)
//	r.EqualError(err, "validation failed: label at index 91 should be 01101111, but found 00011101")
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateFail2(t *testing.T) {
//	r := require.New(t)
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	proof.Challenge = []byte{1}
//
//	v, err := NewValidator(cfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.EqualError(err, "validation failed: merkle root mismatch")
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateFail3(t *testing.T) {
//	r := require.New(t)
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	proof.ProvenLeaves[0] = append([]byte{}, proof.ProvenLeaves[0]...)
//	proof.ProvenLeaves[0][0] += 1
//
//	v, err := NewValidator(cfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.EqualError(err, "validation failed: merkle root mismatch")
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateFail4(t *testing.T) {
//	r := require.New(t)
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	proof.ProvenLeaves = proof.ProvenLeaves[1:]
//
//	v, err := NewValidator(cfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.EqualError(err, "validation failed: number of derived leaf indices (4) doesn't match number of included proven leaves (3)")
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateFail5(t *testing.T) {
//	r := require.New(t)
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	proof.ProofNodes[0] = append([]byte{}, proof.ProofNodes[0]...)
//	proof.ProofNodes[0][0] += 1
//
//	v, err := NewValidator(cfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.EqualError(err, "validation failed: merkle root mismatch")
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateFail6(t *testing.T) {
//	r := require.New(t)
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	proof.ProofNodes = proof.ProofNodes[1:]
//
//	v, err := NewValidator(cfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.EqualError(err, "validation failed: merkle root mismatch")
//
//	err = init.Reset()
//	r.NoError(err)
//}
//
//func TestValidateFail7(t *testing.T) {
//	r := require.New(t)
//
//	init, err := NewInitializer(cfg, id)
//	r.NoError(err)
//	proof, err := init.Initialize()
//	r.NoError(err)
//
//	proof.MerkleRoot = append([]byte{}, proof.MerkleRoot...)
//	proof.MerkleRoot[0] += 1
//
//	v, err := NewValidator(cfg)
//	r.NoError(err)
//	err = v.Validate(id, proof)
//	r.EqualError(err, "validation failed: merkle root mismatch")
//
//	err = init.Reset()
//	r.NoError(err)
//}

func hexDecode(hexStr string) []byte {
	node, _ := hex.DecodeString(hexStr)
	return node
}
