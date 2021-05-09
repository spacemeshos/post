package verifying

import (
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

var (
	id = make([]byte, 32)
	ch = make(proving.Challenge, 32)

	cfg           = config.DefaultConfig()
	CPUProviderID = initialization.CPUProviderID()

	debug = flag.Bool("debug", false, "")

	NewInitializer = initialization.NewInitializer
	NewProver      = proving.NewProver
)

func TestMain(m *testing.M) {
	cfg.DataDir, _ = ioutil.TempDir("", "post-test")
	cfg.NumLabels = 1 << 12

	res := m.Run()
	os.Exit(res)
}

func TestVerify(t *testing.T) {
	req := require.New(t)

	init, err := NewInitializer(cfg, id)
	req.NoError(err)
	err = init.Initialize(CPUProviderID)
	req.NoError(err)

	p, err := NewProver(cfg, id)
	req.NoError(err)
	proof, proofMetadata, err := p.GenerateProof(ch)
	req.NoError(err)

	err = Verify(proof, proofMetadata)
	req.NoError(err)

	// Cleanup.
	err = init.Reset()
	req.NoError(err)
}

// TestLabelsCorrectness tests, for variation of label sizes, the correctness of
// reading labels from disk (written in multiple files) when compared to a single label compute.
// It is covers the following components: labels compute lib (oracle pkg), labels writer (persistence pkg),
// labels reader (persistence pkg), and the granularity-specific reader (shared pkg).
// it proceeds as follows:
// 1. Compute labels, in batches, and write them into multiple files (prover).
// 2. Read the sequence of labels from the files according to the specified label size (prover),
//    and ensure that each one equals a single label compute (verifier).
func TestLabelsCorrectness(t *testing.T) {
	req := require.New(t)
	if testing.Short() {
		t.Skip()
	}

	numFiles := 2
	numFileBatches := 2
	batchSize := 256
	id := make([]byte, 32)
	datadir, _ := ioutil.TempDir("", "post-test")

	for labelSize := uint32(config.MinBitsPerLabel); labelSize <= config.MaxBitsPerLabel; labelSize++ {
		if *debug {
			fmt.Printf("label size: %v\n", labelSize)
		}

		// Write.
		for i := 0; i < numFiles; i++ {
			writer, err := persistence.NewLabelsWriter(datadir, i, uint(labelSize))
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
		reader, err := persistence.NewLabelsReader(datadir, uint(labelSize))
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

		// Cleanup.
		_ = os.RemoveAll(datadir)
	}
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(wrongIdentity, proof)
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(id, proof)
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
//	err = v.Verify(id, proof)
//	r.EqualError(err, "validation failed: merkle root mismatch")
//
//	err = init.Reset()
//	r.NoError(err)
//}
