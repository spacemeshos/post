package initialization

import (
	"encoding/hex"
	"flag"
	"github.com/spacemeshos/post-private/persistence"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
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
		2019-02-19T13:53:02.749+0200	INFO	Spacemesh	creating directory: "/Users/noamnelke/.spacemesh/post-data/deadbeef"
		2019-02-19T13:55:56.848+0200	INFO	Spacemesh	closing file	{"filename": "all.labels", "size_in_bytes": 268435456}
		2019-02-19T13:55:56.849+0200	INFO	Spacemesh	completed PoST label list construction	{"number_of_labels": 33554432, "number_of_oracle_calls": 536922911}
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/post-private/initialization
		BenchmarkInitialize-8   	       1	174098803159 ns/op
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
