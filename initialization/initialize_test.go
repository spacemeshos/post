package initialization

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/proving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

const (
	defaultDifficulty           = 5
	defaultSpace                = proving.Space(16 * LabelGroupSize)
	defaultNumberOfProvenLabels = 4
)

var (
	defaultId = hexDecode("deadbeef")
)

func TestInitialize(t *testing.T) {
	r := require.New(t)

	proof, err := Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, defaultDifficulty)
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

	proof, err := Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, 4)
	r.EqualError(err, "difficulty must be between 5 and 8 (received 4)")
	r.EqualValues(proving.Proof{}, proof)

	proof, err = Initialize(defaultId, defaultSpace, defaultNumberOfProvenLabels, 9)
	r.EqualError(err, "difficulty must be between 5 and 8 (received 9)")
	r.EqualValues(proving.Proof{}, proof)

	proof, err = Initialize(defaultId, proving.MaxSpace+1, 100, defaultDifficulty)
	r.EqualError(err, fmt.Sprintf("space (%d) is greater than the supported max (%d)", proving.MaxSpace+1, proving.MaxSpace))
	r.EqualValues(proving.Proof{}, proof)
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
	id, _ := hex.DecodeString("deadbeef")
	expectedMerkleRoot, _ := hex.DecodeString("af052351d359ce4a3041ce1992d659f68d30f6c1e5c5d229c389c2912a373c70")

	proof, err := Initialize(id, 1<<25, 100, defaultDifficulty)
	require.NoError(b, err)
	println(hex.EncodeToString(proof.MerkleRoot))
	assert.Equal(b, expectedMerkleRoot, proof.MerkleRoot)
	/*
		2019-03-18T17:38:42.336+0200	INFO	Spacemesh	creating directory: "/Users/noamnelke/.spacemesh-data/post-data/deadbeef"
		2019-03-18T17:39:23.608+0200	INFO	Spacemesh	found 5000000 labels
		2019-03-18T17:40:04.247+0200	INFO	Spacemesh	found 10000000 labels
		2019-03-18T17:40:44.546+0200	INFO	Spacemesh	found 15000000 labels
		2019-03-18T17:41:25.565+0200	INFO	Spacemesh	found 20000000 labels
		2019-03-18T17:42:05.958+0200	INFO	Spacemesh	found 25000000 labels
		2019-03-18T17:42:46.402+0200	INFO	Spacemesh	found 30000000 labels
		2019-03-18T17:43:14.990+0200	INFO	Spacemesh	completed PoST label list construction
		2019-03-18T17:43:14.990+0200	INFO	Spacemesh	closing file	{"filename": "all.labels", "size_in_bytes": 1073741824}
		goos: darwin

		af052351d359ce4a3041ce1992d659f68d30f6c1e5c5d229c389c2912a373c70
		goarch: amd64
		pkg: github.com/spacemeshos/post/initialization
		BenchmarkInitialize-8   	       1	272653006697 ns/op
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
	_ = os.RemoveAll(filepath.Join(persistence.GetPostDataPath(), "deadbeef"))
}
