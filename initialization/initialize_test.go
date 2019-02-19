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

func BenchmarkInitialize(b *testing.B) {
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("10000000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("c0f742adce5c9fed7289c0a1664d71f08280f7084dcff24df916a6da56f8a88c")

	merkleRoot, err := Initialize(id, 1<<25, difficulty)
	require.NoError(b, err)
	assert.Equal(b, expectedMerkleRoot, merkleRoot)
	/*
		2019-02-19T15:21:48.505+0200	INFO	Spacemesh	creating directory: "/Users/noamnelke/.spacemesh/post-data/deadbeef"
		2019-02-19T15:22:13.913+0200	INFO	Spacemesh	found 5000000 labels
		2019-02-19T15:22:39.265+0200	INFO	Spacemesh	found 10000000 labels
		2019-02-19T15:23:05.030+0200	INFO	Spacemesh	found 15000000 labels
		2019-02-19T15:23:30.480+0200	INFO	Spacemesh	found 20000000 labels
		2019-02-19T15:23:55.900+0200	INFO	Spacemesh	found 25000000 labels
		2019-02-19T15:24:21.338+0200	INFO	Spacemesh	found 30000000 labels
		2019-02-19T15:24:39.349+0200	INFO	Spacemesh	closing file	{"filename": "all.labels", "size_in_bytes": 268435456}
		2019-02-19T15:24:39.349+0200	INFO	Spacemesh	completed PoST label list construction	{"number_of_labels": 33554432, "number_of_oracle_calls": 536922910}
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/post-private/initialization
		BenchmarkInitialize-8   	       1	170844397489 ns/op
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
