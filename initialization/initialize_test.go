package initialization

import (
	"encoding/hex"
	"flag"
	"github.com/spacemeshos/post-private/persistence"
	"github.com/spacemeshos/post-private/proving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

const difficulty = 5

func TestInitialize(t *testing.T) {
	r := require.New(t)
	id := hexDecode("deadbeef")
	expectedMerkleRoot := hexDecode("fb00ac6f6b50a1433a7691d2e079b0dc5b221b4f3fd5ace3dc00c0db792518bb")

	proof, err := Initialize(id, 16, 4, difficulty)
	r.NoError(err)
	r.Equal(expectedMerkleRoot, proof.MerkleRoot)

	expectedProvenLeaves := nodes{
		hexDecode("d46655d5d089024be1e0cdaf9a581e6841adec80490d4fd83350a1960069a4cc"),
		hexDecode("1efdf0ac1d019422061d4751ae3b21eff98285a38186890944b86c06711f38eb"),
		hexDecode("cb0ff939e3b3a1dd0d36bc14f7f05f8dce599a158ab39e125c175d23a958d79b"),
		hexDecode("a7f925a9734acaea7a87ce76b1314274c82068605e9ca882c545b6de45769c7a"),
	}

	r.EqualValues(expectedProvenLeaves, nodes(proof.ProvenLeaves))

	expectedProofNodes := nodes{
		hexDecode("9c75020dd3efc92f8181e17283063982f4d36a657949bcf094312310847fc6fc"),
		hexDecode("5b3393e2c969ed646d261cdecc5506a177f5d0afa586531c53489be77265030b"),
		hexDecode("a8b27534bec9db1512b8bcb5ea30bd9bda9f5588d8a7b5cbe62580d7fa5d61d5"),
		hexDecode("13f955ccbc82c0c67d473a0e83826f90144e5d43c0410a8002f7d81d25a9da69"),
		hexDecode("39e6a1117dfcea18367ea949b49c46f65c31d1c0fac9afae0fcab83f3c944114"),
		hexDecode("a68af3373456a8c6cbc1de201c7bed0d47ded851e15ddde4dde64aa697c8c0d3"),
		hexDecode("48577ac18b61dbac03650ec87b733e82e37c8eb44c26ba72cf8640f9dd26bf2a"),
		hexDecode("af6e5d56fdc7c77b29b4d42d62ce96eca35823b3e14fe3b64a1272fb3f95c9fa"),
	}

	r.EqualValues(expectedProofNodes, nodes(proof.ProofNodes))

	r.EqualValues([]uint64{1, 6, 9, 12}, proof.ProvenIndices)
}

func TestInitializeErrors(t *testing.T) {
	r := require.New(t)
	id := hexDecode("deadbeef")

	proof, err := Initialize(id, 16, 4, 4)
	r.EqualError(err, "difficulty must be between 5 and 8 (received 4)")
	r.EqualValues(proving.Proof{}, proof)

	proof, err = Initialize(id, 16, 4, 9)
	r.EqualError(err, "difficulty must be between 5 and 8 (received 9)")
	r.EqualValues(proving.Proof{}, proof)

	proof, err = Initialize(id, (1<<50)+1, 100, difficulty)
	r.EqualError(err, "failed to initialize post: requested width (1125899906842625) is greater than "+
		"supported width (1125899906842624)")
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
		s += hex.EncodeToString(v[:2]) + " "
	}
	return s
}

func BenchmarkInitialize(b *testing.B) {
	id, _ := hex.DecodeString("deadbeef")
	expectedMerkleRoot, _ := hex.DecodeString("af052351d359ce4a3041ce1992d659f68d30f6c1e5c5d229c389c2912a373c70")

	proof, err := Initialize(id, 1<<25, 100, difficulty)
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
		pkg: github.com/spacemeshos/post-private/initialization
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
