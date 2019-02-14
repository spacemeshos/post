package initialization

import (
	"encoding/hex"
	"flag"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"post-private/persistence"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("01000000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("d39a77684b387f90792706ce49945c73849c6e5cdca31f220d5045e9fa21086b")

	resChan := Initialize(id, 16, difficulty)

	done := make(chan bool)
	go func() {
		merkleRoot := <-resChan
		assert.Equal(t, expectedMerkleRoot, merkleRoot)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		panic("timeout")
	}
}

func BenchmarkInitialize(b *testing.B) {
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("10000000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("c0f742adce5c9fed7289c0a1664d71f08280f7084dcff24df916a6da56f8a88c")

	resChan := Initialize(id, 1<<25, difficulty)

	done := make(chan bool)
	go func() {
		merkleRoot := <-resChan
		assert.Equal(b, expectedMerkleRoot, merkleRoot)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * 60 * time.Second):
		panic("timeout")
	}
	/*
		creating directory: /Users/noamnelke/.spacemesh/post-data/deadbeef
		closing file: 'all.labels' (268435456 bytes)

		🔹  Constructed list of 33554432 PoST labels.
		🔹  Number of random oracle calls: 536922911
		🔹  Merkle root: c0f742adce5c9fed7289c0a1664d71f08280f7084dcff24df916a6da56f8a88c

		goos: darwin
		goarch: amd64
		pkg: post-private/initialization
		BenchmarkInitialize-8   	       1	170890054153 ns/op
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
