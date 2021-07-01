package proving

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
	smlog "github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

var (
	id = make([]byte, 32)
	ch = make(Challenge, 32)

	cfg  config.Config
	opts config.InitOpts

	log   = flag.Bool("log", false, "")
	debug = flag.Bool("debug", false, "")

	NewInitializer = initialization.NewInitializer
	CPUProviderID  = initialization.CPUProviderID
)

func TestMain(m *testing.M) {
	cfg = config.DefaultConfig()
	cfg.LabelsPerUnit = 1 << 12

	opts = config.DefaultInitOpts()
	opts.DataDir, _ = ioutil.TempDir("", "post-test")
	opts.NumUnits = cfg.MinNumUnits
	opts.NumFiles = 2
	opts.ComputeProviderID = CPUProviderID()

	res := m.Run()
	os.Exit(res)
}

// TODO: tests should range through `cfg.BitsPerLabel` as well.
func TestProver_GenerateProof(t *testing.T) {
	r := require.New(t)

	for numUnits := uint(cfg.MinNumUnits); numUnits < 6; numUnits++ {
		newOpts := opts
		newOpts.NumUnits = numUnits

		init, err := NewInitializer(cfg, newOpts, id)
		r.NoError(err)
		err = init.Initialize()
		r.NoError(err)

		p, err := NewProver(cfg, opts.DataDir, id)
		r.NoError(err)
		if *log {
			p.SetLogger(smlog.AppLog)
		}

		binary.BigEndian.PutUint64(ch, uint64(numUnits))
		proof, proofMetaData, err := p.GenerateProof(ch)
		r.NoError(err, fmt.Sprintf("numUnits: %d", numUnits))
		r.NotNil(proof)
		r.NotNil(proofMetaData)

		r.Equal(id, proofMetaData.ID)
		r.Equal(ch, proofMetaData.Challenge)
		r.Equal(cfg.BitsPerLabel, proofMetaData.BitsPerLabel)
		r.Equal(cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
		r.Equal(numUnits, proofMetaData.NumUnits)
		r.Equal(cfg.K1, proofMetaData.K1)
		r.Equal(cfg.K2, proofMetaData.K2)

		numLabels := uint64(cfg.LabelsPerUnit * numUnits)
		indexBitSize := uint(shared.NumBits(numLabels))
		r.Equal(shared.Size(indexBitSize, p.cfg.K2), uint(len(proof.Indices)))

		if *debug {
			fmt.Printf("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
		}

		err = verifying.Verify(proof, proofMetaData)
		r.NoError(err)

		// Cleanup.
		err = init.Reset()
		r.NoError(err)
	}
}

func TestProver_GenerateProof_NotAllowed(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, opts, id)
	r.NoError(err)
	err = init.Initialize()
	r.NoError(err)

	// Attempt to generate proof with different `ID`.
	newID := make([]byte, 32)
	copy(newID, id)
	newID[0] = newID[0] + 1
	p, err := NewProver(cfg, opts.DataDir, newID)
	r.NoError(err)
	_, _, err = p.GenerateProof(ch)
	r.Error(err)
	errConfigMismatch, ok := err.(initialization.ConfigMismatchError)
	r.True(ok)
	r.Equal("ID", errConfigMismatch.Param)

	// Attempt to generate proof with different `BitsPerLabel`.
	newCfg := cfg
	newCfg.BitsPerLabel++
	p, err = NewProver(newCfg, opts.DataDir, id)
	r.NoError(err)
	_, _, err = p.GenerateProof(ch)
	r.Error(err)
	errConfigMismatch, ok = err.(initialization.ConfigMismatchError)
	r.True(ok)
	r.Equal("BitsPerLabel", errConfigMismatch.Param)

	// Attempt to generate proof with different `LabelsPerUnint`.
	newCfg = cfg
	newCfg.LabelsPerUnit++
	p, err = NewProver(newCfg, opts.DataDir, id)
	r.NoError(err)
	_, _, err = p.GenerateProof(ch)
	r.Error(err)
	errConfigMismatch, ok = err.(initialization.ConfigMismatchError)
	r.True(ok)
	r.Equal("LabelsPerUnit", errConfigMismatch.Param)

	// Cleanup.
	err = init.Reset()
	r.NoError(err)
}

func TestCalcProvingDifficulty(t *testing.T) {
	t.Skip("poc")

	// Implementation of:
	// SUCCESS = msb64(HASH_OUTPUT) <= MAX_TARGET * (K1/NumLabels)

	NumLabels := uint64(4294967296)
	K1 := uint64(2000000)

	fmt.Printf("NumLabels: %v\n", NumLabels)
	fmt.Printf("K1: %v\n", K1)
	fmt.Println()

	maxTarget := uint64(math.MaxUint64)
	fmt.Printf("max target: %d\n", maxTarget)

	if ok := shared.Uint64MulOverflow(NumLabels, K1); ok {
		panic("NumLabels*K1 overflow")
	}

	x := maxTarget / NumLabels
	y := maxTarget % NumLabels
	difficulty := x*K1 + (y*K1)/NumLabels
	fmt.Printf("difficulty: %v\n", difficulty)

	fmt.Println()
	fmt.Printf("calculating various values...\n")
	for i := 129540; i < 129545; i++ { // value 129544 pass
		// Generate a preimage.
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(i))
		fmt.Printf("%v: preimage: 0x%x\n", i, b)

		// Derive the hash output.
		hash := sha256.Sum256(b[:])
		fmt.Printf("%v: hash: Ox%x\n", i, hash)

		// Convert the hash output leading 64 bits to an integer
		// so that it could be used to perform math comparisons.
		hashNum := binary.BigEndian.Uint64(hash[:])
		fmt.Printf("%v: hashNum: %v\n", i, hashNum)

		// Test the difficulty requirement.
		if hashNum > difficulty {
			fmt.Printf("%v: Not passed. hashNum > difficulty\n", i)
		} else {
			fmt.Printf("%v: Great success! hashNum <= difficulty\n", i)
			break
		}

		fmt.Println()
	}
}
