package initialization

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
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

func TestInitializeLong(t *testing.T) {
	if testing.Short() {
		t.Skip("This is a long test")
	}
	id, _ := hex.DecodeString("deadbeef")
	difficulty, _ := hex.DecodeString("00001000000000000000000000000000")
	expectedMerkleRoot, _ := hex.DecodeString("3e65e6dc939d3a31c921069219d1eb8fbcdcc5876c6413c786773a18147f95b3")

	resChan := Initialize(id, 32, difficulty)

	done := make(chan bool)
	go func() {
		merkleRoot := <-resChan
		assert.Equal(t, expectedMerkleRoot, merkleRoot)
		done<-true
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		panic("timeout")
	}
}
