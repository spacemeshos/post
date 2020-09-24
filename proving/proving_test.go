package proving

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math"
	"testing"
)

var (
	cfg            *Config
	id             = make([]byte, 32)
	challenge      = shared.ZeroChallenge
	NewInitializer = initialization.NewInitializer
)

func init() {
	cfg = config.DefaultConfig()
	cfg.NumLabels = 1 << 15
	cfg.LabelSize = 8
	cfg.DataDir, _ = ioutil.TempDir("", "post-test")
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
}

func TestProver(t *testing.T) {
	r := require.New(t)
	init, err := NewInitializer(cfg, id)
	r.NoError(err)

	err = init.Initialize(initialization.CPUProviderID())
	r.NoError(err)
	defer func() {
		err := init.Reset()
		r.NoError(err)
	}()

	p, err := NewProver(cfg, id)
	r.NoError(err)

	//p.SetLogger(log.AppLog)

	proof, proofMetaData, err := p.GenerateProof(id)
	r.NoError(err)
	r.NotNil(proof)
	r.NotNil(proofMetaData)
}

func TestCalcProvingDifficulty(t *testing.T) {
	t.Skip()

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
