package initialization

import (
	"encoding/hex"
	"flag"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"path/filepath"
	"post-private/persistence"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("01000000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("3bcf70f0d75aeb2d10eef08c9df63bb618bbdda6f97418884939d6a69538d7ec")

	resChan := Initialize(id, 16, difficulty)

	done := make(chan bool)
	go func() {
		merkleRoot := <-resChan
		assert.Equal(t, expectedMerkleRoot, merkleRoot)
		done<-true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		panic("timeout")
	}
}

func _TestInitializeLong(t *testing.T) {
	if testing.Short() {
		t.Skip("This is a long test (4+ minutes)")
	}
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("10000000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("f7f5cb643a23c5bcd79bc3950dca9c0535381f549876a72f3f6628a3251d27fe")

	resChan := Initialize(id, uint64(math.Pow(2,25)), difficulty)

	done := make(chan bool)
	go func() {
		merkleRoot := <-resChan
		assert.Equal(t, expectedMerkleRoot, merkleRoot)
		done<-true
	}()

	select {
	case <-done:
	case <-time.After(5 * 60 * time.Second):
		panic("timeout")
	}
	/*
	=== RUN   TestInitializeLong

	🔹  Constructed list of 33554432 PoST labels.
	🔹  Number of random oracle calls: 536922911
	🔹  Merkle root: f7f5cb643a23c5bcd79bc3950dca9c0535381f549876a72f3f6628a3251d27fe

	--- PASS: TestInitializeLong (247.98s)
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
