package initialization

import (
	"encoding/hex"
	"flag"
	"github.com/spacemeshos/post-private/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestInitialize(t *testing.T) {
	id, _ := hex.DecodeString("deadbeef")
	expectedMerkleRoot, _ := hex.DecodeString("fb00ac6f6b50a1433a7691d2e079b0dc5b221b4f3fd5ace3dc00c0db792518bb")

	merkleRoot, err := Initialize(id, 16)
	require.NoError(t, err)
	println(hex.EncodeToString(merkleRoot))
	assert.Equal(t, expectedMerkleRoot, merkleRoot)
}

func TestInitialize2(t *testing.T) {
	id, _ := hex.DecodeString("deadbeef")

	merkleRoot, err := Initialize(id, (1<<50)+1)
	require.Error(t, err)
	require.Nil(t, merkleRoot)
}

func BenchmarkInitialize(b *testing.B) {
	id, _ := hex.DecodeString("deadbeef")
	expectedMerkleRoot, _ := hex.DecodeString("af052351d359ce4a3041ce1992d659f68d30f6c1e5c5d229c389c2912a373c70")

	merkleRoot, err := Initialize(id, 1<<25)
	require.NoError(b, err)
	println(hex.EncodeToString(merkleRoot))
	assert.Equal(b, expectedMerkleRoot, merkleRoot)
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
