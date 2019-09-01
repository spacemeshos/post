package initialization_test

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

const (
	LabelGroupSize  = config.LabelGroupSize
	MaxSpace        = config.MaxSpace
	StateNotStarted = initialization.StateNotStarted
	StateCompleted  = initialization.StateCompleted
)

type (
	Config      = config.Config
	Initializer = initialization.Initializer
)

var (
	NewInitializer = initialization.NewInitializer
)

// Test vars.
var (
	id                   = hexDecode("deadbeef")
	challenge            = hexDecode("this is a challenge")
	datadir, _           = ioutil.TempDir("", "post-test")
	space                = uint64(16 * LabelGroupSize)
	numFiles             = 1
	maxFilesParallelism  = uint(1)
	maxInfileParallelism = uint(1)

	cfg *Config
)

func TestMain(m *testing.M) {
	flag.StringVar(&datadir, "datadir", datadir, "")
	flag.Uint64Var(&space, "space", space, "")
	flag.IntVar(&numFiles, "numfiles", numFiles, "")
	flag.UintVar(&maxFilesParallelism, "parallel-files", maxFilesParallelism, "")
	flag.UintVar(&maxInfileParallelism, "parallel-infile", maxInfileParallelism, "")
	flag.Parse()

	cfg = &Config{
		SpacePerUnit:                            space,
		NumFiles:                                numFiles,
		Difficulty:                              5,
		NumProvenLabels:                         4,
		LowestLayerToCacheDuringProofGeneration: 0,
		DataDir:                                 datadir,
		MaxWriteFilesParallelism:                maxFilesParallelism,
		MaxWriteInFileParallelism:               maxInfileParallelism,
		LabelsLogRate:                           uint64(math.MaxUint64),
	}

	res := m.Run()
	os.Exit(res)
}

func TestInitializer(t *testing.T) {
	r := require.New(t)

	init := NewInitializer(cfg, id)
	proof, err := init.Initialize()
	defer cleanup(init)
	r.NoError(err)

	expectedMerkleRoot := hexDecode("2292f95c87626f5a281fa811ba825ffce79442f8999e1ddc8e8c9bbac15e3fcb")
	r.Equal(expectedMerkleRoot, proof.MerkleRoot)

	expectedProvenLeaves := nodes{
		hexDecode("1507851a83f1b8644dbbc09c4cb66d28397ed7f5cecce3d5dbce4b6f0b7cd5b3"),
		hexDecode("04e98f15e487573d38609f0cb50e4d66107d2aef126dd52f4833f24200e099ff"),
		hexDecode("f0e25e059be7c13a2af257568f7ea386ccbf9f175b7af3c978e3376c48ba20ff"),
		hexDecode("d876529601cf04b6acc7ee1ac2b33f052e58d0dce58859e3a4a6a029ded70ee0"),
	}

	r.EqualValues(expectedProvenLeaves, nodes(proof.ProvenLeaves))

	expectedProofNodes := nodes{
		hexDecode("94686b27f3ef2ab9415f95aeafba42da6f4036872dffcc5475e9749980e8e4b3"),
		hexDecode("750ba998411ef4d1357fead36c2b080c53bef7fa8a9bd3ff02cae1aef08fce7d"),
		hexDecode("9847d3adad39f5c2a8c2f9e7d8d3001caf6b65c9a544e537c55f630949d6c440"),
		hexDecode("6695ccdf6ff22dc17c7cdd3217b7d49405824266d35bda1eeae610335a2247bd"),
		hexDecode("8bed2cae59accd2c817c4d82a11c610d5590d96e98607cbc1bc4c7040d9ade8b"),
		hexDecode("09db8e0d03b3786a4cd05dd1dce42d7d6dfbfabd63575734b531ab80c05ff41d"),
	}

	r.EqualValues(expectedProofNodes, nodes(proof.ProofNodes))
}

func TestInitializerErrors(t *testing.T) {
	r := require.New(t)

	newCfg := *cfg
	newCfg.Difficulty = 4
	init := NewInitializer(&newCfg, id)
	proof, err := init.Initialize()
	r.EqualError(err, "difficulty must be between 5 and 8 (received 4)")
	r.Nil(proof)

	newCfg = *cfg
	newCfg.Difficulty = 9
	init = NewInitializer(&newCfg, id)
	proof, err = init.Initialize()
	r.EqualError(err, "difficulty must be between 5 and 8 (received 9)")
	r.Nil(proof)

	newCfg = *cfg
	newCfg.SpacePerUnit = MaxSpace + 1
	init = NewInitializer(&newCfg, id)
	proof, err = init.Initialize()
	r.EqualError(err, fmt.Sprintf("space (%d) is greater than the supported max (%d)", MaxSpace+1, MaxSpace))
	r.Nil(proof)
}

func TestInitializerMultipleFiles(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.SpacePerUnit = 1 << 15
	cfg.NumFiles = 1

	init := NewInitializer(&cfg, id)
	initProof, err := init.Initialize()
	r.NoError(err)

	execProof, err := proving.NewProver(&cfg, id).GenerateProof(challenge)
	r.NoError(err)

	cleanup(init)

	for numFiles := 2; numFiles <= 16; numFiles <<= 1 {
		newCfg := cfg
		newCfg.NumFiles = numFiles
		newCfg.MaxWriteFilesParallelism = uint(numFiles)
		newCfg.MaxWriteInFileParallelism = uint(numFiles)
		newCfg.MaxReadFilesParallelism = uint(numFiles)

		init := NewInitializer(&newCfg, id)
		multiFilesInitProof, err := init.Initialize()
		r.NoError(err)

		multiFilesExecProof, err := proving.NewProver(&newCfg, id).GenerateProof(challenge)
		r.NoError(err)

		cleanup(init)

		r.Equal(initProof.MerkleRoot, multiFilesInitProof.MerkleRoot)
		r.EqualValues(initProof.ProvenLeaves, multiFilesInitProof.ProvenLeaves)
		r.EqualValues(initProof.ProofNodes, multiFilesInitProof.ProofNodes)

		r.Equal(execProof.MerkleRoot, multiFilesExecProof.MerkleRoot)
		r.EqualValues(execProof.ProvenLeaves, multiFilesExecProof.ProvenLeaves)
		r.EqualValues(execProof.ProofNodes, multiFilesExecProof.ProofNodes)
	}
}

func TestInitializer_State(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.SpacePerUnit = 1 << 15
	cfg.NumFiles = 1

	init := NewInitializer(&cfg, id)

	state, requiredSpace, err := init.State()
	r.Equal(StateNotStarted, state)
	r.Equal(cfg.SpacePerUnit, requiredSpace)
	r.NoError(err)

	_, err = init.Initialize()
	r.NoError(err)

	state, requiredSpace, err = init.State()
	r.Equal(StateCompleted, state)
	r.Equal(uint64(0), requiredSpace)
	r.NoError(err)

	_, err = init.Initialize()
	r.Equal(err, shared.ErrInitCompleted)

	// Initialize using a new instance.

	init = NewInitializer(&cfg, id)

	state, requiredSpace, err = init.State()
	r.Equal(StateCompleted, state)
	r.Equal(uint64(0), requiredSpace)
	r.NoError(err)

	_, err = init.Initialize()
	r.Equal(err, shared.ErrInitCompleted)

	// Use a new instance with a different config.

	newCfg := cfg
	newCfg.SpacePerUnit = 1 << 14
	newCfg.NumFiles = 1

	init = NewInitializer(&newCfg, id)

	_, _, err = init.State()
	r.Equal(err, initialization.ErrStateConfigMismatch)

	_, err = init.Initialize()
	r.Equal(err, initialization.ErrStateConfigMismatch)

	err = init.Reset()
	r.Equal(err, initialization.ErrStateConfigMismatch)

	// Reset with the correct config.

	init = NewInitializer(&cfg, id)
	err = init.Reset()
	r.NoError(err)
}

func hexDecode(hexStr string) []byte {
	node, _ := hex.DecodeString(hexStr)
	return node
}

type nodes [][]byte

func (n nodes) String() string {
	s := ""
	for _, v := range n {
		s += "\n" + hex.EncodeToString(v[:]) + " "
	}
	return s
}

func BenchmarkInitialize30(b *testing.B) {
	space := uint64(1) << 30 // 1 GB.

	newCfg := *cfg
	newCfg.SpacePerUnit = space
	newCfg.NumFiles = 1

	init := NewInitializer(&newCfg, id)
	proof, err := init.Initialize()
	cleanup(init)
	require.NoError(b, err)

	expectedMerkleRoot, _ := hex.DecodeString("42dd3ed26e6f30f8098ec0b5093147551b32573ef9ed6670076248b4fd0fac30")
	assert.Equal(b, expectedMerkleRoot, proof.MerkleRoot)
	/*
		2019-04-30T11:49:10.271+0300    INFO    Spacemesh       creating directory: "/Users/moshababo/.spacemesh-data/post-data/deadbeef"
		2019-04-30T11:49:45.168+0300    INFO    Spacemesh       found 5000000 labels
		2019-04-30T11:50:21.192+0300    INFO    Spacemesh       found 10000000 labels
		2019-04-30T11:50:56.607+0300    INFO    Spacemesh       found 15000000 labels
		2019-04-30T11:51:32.103+0300    INFO    Spacemesh       found 20000000 labels
		2019-04-30T11:52:07.283+0300    INFO    Spacemesh       found 25000000 labels
		2019-04-30T11:52:42.195+0300    INFO    Spacemesh       found 30000000 labels
		2019-04-30T11:53:07.074+0300    INFO    Spacemesh       completed PoST label list construction
		2019-04-30T11:53:07.074+0300    INFO    Spacemesh       closing file    {"filename": "all.labels", "size_in_bytes": 1073741824}

		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/post/initialization
		BenchmarkInitialize-12                 1        236916764781 ns/op
		PASS
	*/
}

func BenchmarkInitializeGeneric(b *testing.B) {
	// Use cli flags (TestMain) to utilize this test.
	init := NewInitializer(cfg, id)
	init.SetLogger(log.AppLog)
	_, err := init.Initialize()
	cleanup(init)
	require.NoError(b, err)
}

func cleanup(init *Initializer) {
	err := init.Reset()
	if err != nil {
		panic(err)
	}
}
