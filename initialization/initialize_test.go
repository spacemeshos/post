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
	difficulty, _ := hex.DecodeString("01000000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("d39a77684b387f90792706ce49945c73849c6e5cdca31f220d5045e9fa21086b")

	merkleRoot, err := Initialize(id, 16, difficulty)
	require.NoError(t, err)
	assert.Equal(t, expectedMerkleRoot, merkleRoot)
}

func TestInitialize2(t *testing.T) {
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("01000000000000000000000000000000")

	merkleRoot, err := Initialize(id, (1<<50)+1, difficulty)
	require.Error(t, err)
	require.Nil(t, merkleRoot)
}

func BenchmarkInitialize(b *testing.B) {
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("10000000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("c0f742adce5c9fed7289c0a1664d71f08280f7084dcff24df916a6da56f8a88c")

	merkleRoot, err := Initialize(id, 1<<25, difficulty)
	require.NoError(b, err)
	assert.Equal(b, expectedMerkleRoot, merkleRoot)
	/*
		2019-02-21T11:54:22.649+0200	INFO	Spacemesh	creating directory: "/Users/noamnelke/.spacemesh-data/post-data/deadbeef"
		2019-02-21T11:54:47.512+0200	INFO	Spacemesh	found 5000000 labels
		2019-02-21T11:55:12.373+0200	INFO	Spacemesh	found 10000000 labels
		2019-02-21T11:55:37.346+0200	INFO	Spacemesh	found 15000000 labels
		2019-02-21T11:56:02.292+0200	INFO	Spacemesh	found 20000000 labels
		2019-02-21T11:56:27.203+0200	INFO	Spacemesh	found 25000000 labels
		2019-02-21T11:56:52.184+0200	INFO	Spacemesh	found 30000000 labels
		2019-02-21T11:57:10.325+0200	INFO	Spacemesh	completed PoST label list construction	{"number_of_labels": 33554432, "number_of_oracle_calls": 536922910}
		2019-02-21T11:57:10.325+0200	INFO	Spacemesh	closing file	{"filename": "all.labels", "size_in_bytes": 268435456}
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/post-private/initialization
		BenchmarkInitialize-8   	       1	167676495711 ns/op
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
