package initialization_test

import (
	"encoding/hex"
	"flag"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

const (
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
	challenge = shared.ZeroChallenge
	id        = hexDecode("deadbeef")
	cfg       *Config
)

func TestMain(m *testing.M) {
	cfg = config.DefaultConfig()
	cfg.DataDir, _ = ioutil.TempDir("", "post-test")
	cfg.LabelsLogRate = uint64(math.MaxUint64)

	flag.StringVar(&cfg.DataDir, "datadir", cfg.DataDir, "")
	flag.Uint64Var(&cfg.NumLabels, "numlabels", cfg.NumLabels, "")
	flag.UintVar(&cfg.LabelSize, "labelsize", cfg.LabelSize, "")
	flag.UintVar(&cfg.NumFiles, "numfiles", cfg.NumFiles, "")
	flag.Parse()

	res := m.Run()
	os.Exit(res)
}

func TestInitializer(t *testing.T) {
	r := require.New(t)

	init, err := NewInitializer(cfg, id)
	r.NoError(err)
	err = init.Initialize()
	defer cleanup(init)
	r.NoError(err)
}

//
//func TestInitializerErrors(t *testing.T) {
//	r := require.New(t)
//
//	newCfg := *cfg
//	newCfg.Difficulty = 4
//	init, err := NewInitializer(&newCfg, id)
//	r.Nil(init)
//	r.EqualError(err, "difficulty must be between 5 and 8 (received 4)")
//
//	newCfg = *cfg
//	newCfg.Difficulty = 9
//	init, err = NewInitializer(&newCfg, id)
//	r.Nil(init)
//	r.EqualError(err, "difficulty must be between 5 and 8 (received 9)")
//
//	newCfg = *cfg
//	newCfg.SpacePerUnit = MaxSpace + 1
//	init, err = NewInitializer(&newCfg, id)
//	r.Nil(init)
//	r.EqualError(err, fmt.Sprintf("numLabels (%d) is greater than the supported max (%d)", MaxSpace+1, MaxSpace))
//}

func TestInitializerMultipleFiles(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 15
	cfg.NumFiles = 1

	init, err := NewInitializer(&cfg, id)
	r.NoError(err)
	err = init.Initialize()
	r.NoError(err)

	cleanup(init)

	for numFiles := uint(2); numFiles <= 16; numFiles <<= 1 {
		newCfg := cfg
		newCfg.NumFiles = numFiles
		newCfg.MaxWriteFilesParallelism = uint(numFiles)
		newCfg.MaxWriteInFileParallelism = uint(numFiles)
		newCfg.MaxReadFilesParallelism = uint(numFiles)

		init, err := NewInitializer(&newCfg, id)
		r.NoError(err)
		err = init.Initialize()
		r.NoError(err)

		cleanup(init)

		// TODO(moshababo): compare the init data. the proofs are random, so there's no point to compare.

		//r.Equal(initProof.MerkleRoot, multiFilesInitProof.MerkleRoot)
		//r.EqualValues(initProof.ProvenLeaves, multiFilesInitProof.ProvenLeaves)
		//r.EqualValues(initProof.ProofNodes, multiFilesInitProof.ProofNodes)
		//
		//r.Equal(execProof.MerkleRoot, multiFilesExecProof.MerkleRoot)
		//r.EqualValues(execProof.ProvenLeaves, multiFilesExecProof.ProvenLeaves)
		//r.EqualValues(execProof.ProofNodes, multiFilesExecProof.ProofNodes)
	}
}

func TestInitializer_State(t *testing.T) {
	r := require.New(t)

	cfg := *cfg
	cfg.NumLabels = 1 << 15
	cfg.NumFiles = 1

	init, err := NewInitializer(&cfg, id)
	r.NoError(err)

	state, requiredSpace, err := init.State()
	r.Equal(StateNotStarted, state)
	r.Equal(cfg.Space(), requiredSpace)
	r.NoError(err)

	err = init.Initialize()
	r.NoError(err)

	state, requiredSpace, err = init.State()
	r.Equal(StateCompleted, state)
	r.Equal(uint64(0), requiredSpace)
	r.NoError(err)

	err = init.Initialize()
	r.Equal(err, shared.ErrInitCompleted)

	// Initialize using a new instance.

	init, err = NewInitializer(&cfg, id)
	r.NoError(err)

	state, requiredSpace, err = init.State()
	r.Equal(StateCompleted, state)
	r.Equal(uint64(0), requiredSpace)
	r.NoError(err)

	err = init.Initialize()
	r.Equal(err, shared.ErrInitCompleted)

	// Use a new instance with a different config.

	newCfg := cfg
	newCfg.NumLabels = 1 << 14
	newCfg.NumFiles = 1

	init, err = NewInitializer(&newCfg, id)
	r.NoError(err)

	_, _, err = init.State()
	r.Equal(err, initialization.ErrStateConfigMismatch)

	err = init.Initialize()
	r.Equal(err, initialization.ErrStateConfigMismatch)

	err = init.Reset()
	r.Equal(err, initialization.ErrStateConfigMismatch)

	// Reset with the correct config.

	init, err = NewInitializer(&cfg, id)
	r.NoError(err)
	err = init.Reset()
	r.NoError(err)
}

func hexDecode(hexStr string) []byte {
	node, _ := hex.DecodeString(hexStr)
	return node
}

func BenchmarkInitialize30(b *testing.B) {
	newCfg := *cfg
	newCfg.NumFiles = 1
	// 1 GB.
	newCfg.NumLabels = 1 << 27
	newCfg.LabelSize = 8

	init, err := NewInitializer(&newCfg, id)
	require.NoError(b, err)
	err = init.Initialize()
	cleanup(init)
	require.NoError(b, err)

	// TODO(moshababo): print new version
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
	init, err := NewInitializer(cfg, id)
	require.NoError(b, err)
	init.SetLogger(log.AppLog)
	err = init.Initialize()
	cleanup(init)
	require.NoError(b, err)
}

func cleanup(init *Initializer) {
	err := init.Reset()
	if err != nil {
		panic(err)
	}
}
