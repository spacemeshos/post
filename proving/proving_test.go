package proving

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/minio/sha256-simd"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

var (
	NewInitializer = initialization.NewInitializer
	CPUProviderID  = initialization.CPUProviderID
)

func getTestConfig(t *testing.T) (config.Config, config.InitOpts) {
	cfg := config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts := config.DefaultInitOpts()
	opts.DataDir = t.TempDir()
	opts.NumUnits = cfg.MinNumUnits
	opts.ComputeProviderID = int(CPUProviderID())

	return cfg, opts
}

type testLogger struct {
	shared.Logger

	t *testing.T
}

func (l testLogger) Info(msg string, args ...any)  { l.t.Logf("\tINFO\t"+msg, args...) }
func (l testLogger) Debug(msg string, args ...any) { l.t.Logf("\tDEBUG\t"+msg, args...) }

func TestProver_GenerateProof(t *testing.T) {
	// TODO(moshababo): tests should range through `cfg.BitsPerLabel` as well.
	r := require.New(t)
	log := testLogger{t: t}

	for numUnits := uint32(config.DefaultMinNumUnits); numUnits < 6; numUnits++ {
		t.Run(fmt.Sprintf("numUnits=%d", numUnits), func(t *testing.T) {
			t.Parallel()

			nodeId := make([]byte, 32)
			commitmentAtxId := make([]byte, 32)
			ch := make(Challenge, 32)
			cfg := config.DefaultConfig()
			cfg.LabelsPerUnit = 1 << 12

			opts := config.DefaultInitOpts()
			opts.ComputeProviderID = int(CPUProviderID())
			opts.NumUnits = numUnits
			opts.DataDir = t.TempDir()

			init, err := NewInitializer(
				initialization.WithNodeId(nodeId),
				initialization.WithCommitmentAtxId(commitmentAtxId),
				initialization.WithConfig(cfg),
				initialization.WithInitOpts(opts),
				initialization.WithLogger(log),
			)
			r.NoError(err)
			r.NoError(init.Initialize(context.Background()))

			p, err := NewProver(cfg, opts.DataDir, nodeId, commitmentAtxId)
			r.NoError(err)
			p.SetLogger(log)

			binary.BigEndian.PutUint64(ch, uint64(opts.NumUnits))
			proof, proofMetaData, err := p.GenerateProof(ch)
			r.NoError(err, "numUnits: %d", opts.NumUnits)
			r.NotNil(proof)
			r.NotNil(proofMetaData)

			r.Equal(nodeId, proofMetaData.NodeId)
			r.Equal(commitmentAtxId, proofMetaData.CommitmentAtxId)
			r.Equal(ch, proofMetaData.Challenge)
			r.Equal(cfg.BitsPerLabel, proofMetaData.BitsPerLabel)
			r.Equal(cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
			r.Equal(opts.NumUnits, proofMetaData.NumUnits)
			r.Equal(cfg.K1, proofMetaData.K1)
			r.Equal(cfg.K2, proofMetaData.K2)

			numLabels := cfg.LabelsPerUnit * uint64(numUnits)
			indexBitSize := uint(shared.BinaryRepresentationMinBits(numLabels))
			r.Equal(shared.Size(indexBitSize, uint(p.cfg.K2)), uint(len(proof.Indices)))

			log.Info("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))

			r.NoError(verifying.Verify(proof, proofMetaData))
		})
	}
}

func TestProver_GenerateProof_NotAllowed(t *testing.T) {
	r := require.New(t)

	nodeId := make([]byte, 32)
	commitmentAtxId := make([]byte, 32)

	ch := make(Challenge, 32)
	cfg, opts := getTestConfig(t)
	init, err := NewInitializer(
		initialization.WithNodeId(nodeId),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithLogger(testLogger{t: t}),
	)
	r.NoError(err)
	r.NoError(init.Initialize(context.Background()))

	// Attempt to generate proof with different `nodeId`.
	newNodeId := make([]byte, 32)
	copy(newNodeId, nodeId)
	newNodeId[0] = newNodeId[0] + 1
	p, err := NewProver(cfg, opts.DataDir, newNodeId, commitmentAtxId)
	r.NoError(err)
	_, _, err = p.GenerateProof(ch)
	var errConfigMismatch initialization.ConfigMismatchError
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("NodeId", errConfigMismatch.Param)

	// Attempt to generate proof with different `atxId`.
	newAtxId := make([]byte, 32)
	copy(newAtxId, commitmentAtxId)
	newAtxId[0] = newAtxId[0] + 1
	p, err = NewProver(cfg, opts.DataDir, nodeId, newAtxId)
	r.NoError(err)
	_, _, err = p.GenerateProof(ch)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("CommitmentAtxId", errConfigMismatch.Param)

	// Attempt to generate proof with different `BitsPerLabel`.
	newCfg := cfg
	newCfg.BitsPerLabel++
	p, err = NewProver(newCfg, opts.DataDir, nodeId, commitmentAtxId)
	r.NoError(err)
	_, _, err = p.GenerateProof(ch)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("BitsPerLabel", errConfigMismatch.Param)

	// Attempt to generate proof with different `LabelsPerUnint`.
	newCfg = cfg
	newCfg.LabelsPerUnit++
	p, err = NewProver(newCfg, opts.DataDir, nodeId, commitmentAtxId)
	r.NoError(err)
	_, _, err = p.GenerateProof(ch)
	r.ErrorAs(err, &errConfigMismatch)
	r.Equal("LabelsPerUnit", errConfigMismatch.Param)
}

func TestCalcProvingDifficulty(t *testing.T) {
	t.Skip("poc")

	// Implementation of:
	// SUCCESS = msb64(HASH_OUTPUT) <= MAX_TARGET * (K1/NumLabels)

	NumLabels := uint64(4294967296)
	K1 := uint64(2000000)

	t.Logf("NumLabels: %v\n", NumLabels)
	t.Logf("K1: %v\n", K1)

	maxTarget := uint64(math.MaxUint64)
	t.Logf("\nmax target: %d\n", maxTarget)

	if ok := shared.Uint64MulOverflow(NumLabels, K1); ok {
		panic("NumLabels*K1 overflow")
	}

	x := maxTarget / NumLabels
	y := maxTarget % NumLabels
	difficulty := x*K1 + (y*K1)/NumLabels
	t.Logf("difficulty: %v\n", difficulty)

	t.Log("\ncalculating various values...\n")
	for i := 129540; i < 129545; i++ { // value 129544 pass
		// Generate a preimage.
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(i))
		t.Logf("%v: preimage: 0x%x\n", i, b)

		// Derive the hash output.
		hash := sha256.Sum256(b[:])
		t.Logf("%v: hash: Ox%x\n", i, hash)

		// Convert the hash output leading 64 bits to an integer
		// so that it could be used to perform math comparisons.
		hashNum := binary.BigEndian.Uint64(hash[:])
		t.Logf("%v: hashNum: %v\n", i, hashNum)

		// Test the difficulty requirement.
		if hashNum > difficulty {
			t.Logf("%v: Not passed. hashNum > difficulty\n", i)
		} else {
			t.Logf("%v: Great success! hashNum <= difficulty\n", i)
			break
		}

		t.Log("\n")
	}
}
