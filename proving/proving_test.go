package proving

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
	smlog "github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

var (
	id = make([]byte, 32)
	ch = make(Challenge, 32)

	cfg = config.DefaultConfig()

	log   = flag.Bool("log", false, "")
	debug = flag.Bool("debug", false, "")

	NewInitializer = initialization.NewInitializer
)

func TestMain(m *testing.M) {
	cfg.DataDir, _ = ioutil.TempDir("", "post-test")
	cfg.BitsPerLabel = 8
	cfg.LabelsPerUnit = 1 << 15

	res := m.Run()
	os.Exit(res)
}

// TODO: verifier tests should range through labelSizes

func TestProver_GenerateProof(t *testing.T) {
	req := require.New(t)

	// Test one numLabel value for every index size, up to 16,
	// which should result in a different size of the list of indices.
	// TODO: update documentation
	for numUnits := uint(config.MinNumUnits); numUnits < 10; numUnits++ {
		cfg := *cfg
		//cfg. = numLabels
		//cfg.K1 = uint(numLabels)
		//cfg.K2 = uint(numLabels)

		init, err := NewInitializer(&cfg, id)
		req.NoError(err)
		err = init.Initialize(initialization.CPUProviderID(), numUnits)
		req.NoError(err)

		p, err := NewProver(&cfg, id)
		req.NoError(err)
		if *log {
			p.SetLogger(smlog.AppLog)
		}

		binary.BigEndian.PutUint64(ch, uint64(numUnits))
		proof, proofMetaData, err := p.GenerateProof(ch)
		req.NoError(err, fmt.Sprintf("numUnits: %d", numUnits))
		req.NotNil(proof)
		req.NotNil(proofMetaData)

		req.Equal(id, proofMetaData.ID)
		req.Equal(ch, proofMetaData.Challenge)
		req.Equal(cfg.BitsPerLabel, proofMetaData.BitsPerLabel)
		req.Equal(cfg.LabelsPerUnit, proofMetaData.LabelsPerUnit)
		req.Equal(numUnits, proofMetaData.NumUnits)
		req.Equal(cfg.K1, proofMetaData.K1)
		req.Equal(cfg.K2, proofMetaData.K2)

		numLabels := uint64(cfg.LabelsPerUnit * numUnits)
		indexBitSize := uint(shared.NumBits(numLabels))
		req.Equal(shared.Size(indexBitSize, p.cfg.K2), uint(len(proof.Indices)))

		if *debug {
			fmt.Printf("numLabels: %v, indices size: %v\n", numLabels, len(proof.Indices))
		}

		// Cleanup.
		err = init.Reset()
		req.NoError(err)
	}
}

//
//func TestProver_GenerateProof_NotAllowed(t *testing.T) {
//	req := require.New(t)
//
//	cfg := *cfg
//	cfg.NumLabels = 1 << 12
//	cfg.K1 = uint(cfg.NumLabels)
//	cfg.K2 = uint(cfg.NumLabels)
//
//	init, err := NewInitializer(&cfg, id)
//	req.NoError(err)
//	err = init.Initialize(initialization.CPUProviderID())
//	req.NoError(err)
//
//	// Attempt to generate proof with different `id`.
//	newID := make([]byte, 32)
//	copy(newID, id)
//	newID[0] = newID[0] + 1
//	p, err := NewProver(&cfg, newID)
//	req.NoError(err)
//	_, _, err = p.GenerateProof(ch)
//	req.Error(err)
//	errConfigMismatch, ok := err.(initialization.ConfigMismatchError)
//	req.True(ok)
//	req.Equal("ID", errConfigMismatch.Param)
//
//	// Attempt to generate proof with different `labelSize`.
//	newCfg := cfg
//	newCfg.LabelSize = newCfg.LabelSize + 1
//	p, err = NewProver(&newCfg, id)
//	req.NoError(err)
//	_, _, err = p.GenerateProof(ch)
//	req.Error(err)
//	errConfigMismatch, ok = err.(initialization.ConfigMismatchError)
//	req.True(ok)
//	req.Equal("LabelSize", errConfigMismatch.Param)
//
//	// Attempt to generate proof with different `numLabels`.
//	newCfg = cfg
//	newCfg.NumLabels = newCfg.NumLabels << 1
//	p, err = NewProver(&newCfg, id)
//	req.NoError(err)
//	_, _, err = p.GenerateProof(ch)
//	req.Equal(shared.ErrInitNotCompleted, err)
//
//	// Cleanup.
//	err = init.Reset()
//	req.NoError(err)
//}
//
//func TestCalcProvingDifficulty(t *testing.T) {
//	t.Skip("playground")
//
//	// Implementation of:
//	// SUCCESS = msb64(HASH_OUTPUT) <= MAX_TARGET * (K1/NumLabels)
//
//	NumLabels := uint64(4294967296)
//	K1 := uint64(2000000)
//
//	fmt.Printf("NumLabels: %v\n", NumLabels)
//	fmt.Printf("K1: %v\n", K1)
//	fmt.Println()
//
//	maxTarget := uint64(math.MaxUint64)
//	fmt.Printf("max target: %d\n", maxTarget)
//
//	if ok := shared.Uint64MulOverflow(NumLabels, K1); ok {
//		panic("NumLabels*K1 overflow")
//	}
//
//	x := maxTarget / NumLabels
//	y := maxTarget % NumLabels
//	difficulty := x*K1 + (y*K1)/NumLabels
//	fmt.Printf("difficulty: %v\n", difficulty)
//
//	fmt.Println()
//	fmt.Printf("calculating various values...\n")
//	for i := 129540; i < 129545; i++ { // value 129544 pass
//		// Generate a preimage.
//		var b [4]byte
//		binary.BigEndian.PutUint32(b[:], uint32(i))
//		fmt.Printf("%v: preimage: 0x%x\n", i, b)
//
//		// Derive the hash output.
//		hash := sha256.Sum256(b[:])
//		fmt.Printf("%v: hash: Ox%x\n", i, hash)
//
//		// Convert the hash output leading 64 bits to an integer
//		// so that it could be used to perform math comparisons.
//		hashNum := binary.BigEndian.Uint64(hash[:])
//		fmt.Printf("%v: hashNum: %v\n", i, hashNum)
//
//		// Test the difficulty requirement.
//		if hashNum > difficulty {
//			fmt.Printf("%v: Not passed. hashNum > difficulty\n", i)
//		} else {
//			fmt.Printf("%v: Great success! hashNum <= difficulty\n", i)
//			break
//		}
//
//		fmt.Println()
//	}
//}
