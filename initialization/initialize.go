package initialization

import (
	"encoding/hex"
	"fmt"
	"math"
	"post-private/datatypes"
	"post-private/merkle"
	"post-private/persistence"
)

// Initialize takes an id (public key), width (number of labels) and difficulty (hash to use as upper bound for labels).
// The merkle root is passed on the results channel.
func Initialize(id []byte, width uint64, difficulty []byte) <-chan []byte {
	// TODO @noam: tune performance (parallel PoW, but don't overuse the machine)
	ch := make(chan []byte)
	go func() {
		res := initializeSync(id, width, difficulty)
		ch <- res
	}()
	return ch
}

func initializeSync(id []byte, width uint64, difficulty []byte) []byte {
	labels := make([]datatypes.Label, 0, width)
	var cnt uint64 = 0
	for len(labels) < int(width) { // TODO @noam: handle larger than int
		l := datatypes.NewLabel(cnt)
		if datatypes.CalcHash(id, l).IsLessThan(difficulty) {
			labels = append(labels, l)
		}
		if cnt == math.MaxUint64 && len(labels) < int(width) {
			panic("Out of counter space!") // TODO @noam: handle gracefully?
		}
		cnt++
	}
	persistence.PersistPostLabels(id, labels)
	root := merkle.CalcMerkleRoot(labels)
	fmt.Printf("\n===\nConstructed list of %v PoST labels.\n"+
		"Number of random oracle calls: %v\n"+
		"Merkel root: %v\n"+
		"Actual labels: %v\n===\n\n", len(labels), cnt, hex.EncodeToString(root), labels)
	return root
}
