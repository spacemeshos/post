package initialization

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/proving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

const (
	defaultDifficulty        = 5
	defaultSpace             = 16 * LabelGroupSize
	defaultNumOfProvenLabels = 4
)

var (
	defaultId        = hexDecode("deadbeef")
	defaultChallenge = hexDecode("this is a challenge")
	tempdir, _       = ioutil.TempDir("", "post-test")
	lograte          = uint64(math.MaxUint64)
)

func TestInitialize(t *testing.T) {
	r := require.New(t)

	proof, err := initialize(defaultId, defaultSpace, defaultSpace, defaultNumOfProvenLabels, defaultDifficulty, false, tempdir, lograte)
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

func TestInitializeErrors(t *testing.T) {
	r := require.New(t)

	proof, err := initialize(defaultId, defaultSpace, defaultSpace, defaultNumOfProvenLabels, 4, false, tempdir, lograte)
	r.EqualError(err, "difficulty must be between 5 and 8 (received 4)")
	r.Nil(proof)

	proof, err = initialize(defaultId, defaultSpace, defaultSpace, defaultNumOfProvenLabels, 9, false, tempdir, lograte)
	r.EqualError(err, "difficulty must be between 5 and 8 (received 9)")
	r.Nil(proof)

	proof, err = initialize(defaultId, MaxSpace+1, defaultSpace, NumOfProvenLabels, defaultDifficulty, false, tempdir, lograte)
	r.EqualError(err, fmt.Sprintf("space (%d) is greater than the supported max (%d)", MaxSpace+1, MaxSpace))
	r.Nil(proof)
}

func TestInitializeMultipleFiles(t *testing.T) {
	r := require.New(t)
	space := uint64(defaultSpace)

	proof, err := initialize(defaultId, space, space, defaultNumOfProvenLabels, defaultDifficulty, false, tempdir, lograte)
	r.NoError(err)
	execProof, err := proving.GenerateProof(defaultId, defaultChallenge, defaultNumOfProvenLabels, defaultDifficulty, tempdir)
	r.NoError(err)

	for numOfFiles := uint64(2); numOfFiles <= 16; numOfFiles *= 2 {
		cleanup()
		multiFilesProof, err := initialize(defaultId, space, space/numOfFiles, defaultNumOfProvenLabels, defaultDifficulty, true, tempdir, lograte)
		r.NoError(err)
		multiFilesExecProof, err := proving.GenerateProof(defaultId, defaultChallenge, defaultNumOfProvenLabels, defaultDifficulty, tempdir)
		r.NoError(err)

		r.Equal(proof.MerkleRoot, multiFilesProof.MerkleRoot)
		r.EqualValues(proof.ProvenLeaves, multiFilesProof.ProvenLeaves)
		r.EqualValues(proof.ProofNodes, multiFilesProof.ProofNodes)

		r.Equal(execProof.MerkleRoot, multiFilesExecProof.MerkleRoot)
		r.EqualValues(execProof.ProvenLeaves, multiFilesExecProof.ProvenLeaves)
		r.EqualValues(execProof.ProofNodes, multiFilesExecProof.ProofNodes)
	}
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

func BenchmarkInitialize(b *testing.B) {
	space := uint64(1) << 30 // 1 GB.
	proof, err := initialize(defaultId, space, space, NumOfProvenLabels, defaultDifficulty, false, tempdir, lograte)
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

func TestMain(m *testing.M) {
	flag.Parse()
	res := m.Run()
	cleanup()
	os.Exit(res)
}

func cleanup() {
	_ = os.RemoveAll(tempdir)
}
