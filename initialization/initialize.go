package initialization

import (
	"encoding/hex"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/post-private/persistence"
	"github.com/spacemeshos/post-private/util"
	"math"
)

// Initialize takes an id (public key), width (number of labels) and difficulty (hash to use as upper bound for labels).
// The merkle root is passed on the results channel.
func Initialize(id []byte, width uint64, difficulty []byte) <-chan []byte {
	// TODO @noam: tune performance (parallel PoW, but don't overuse the machine)
	ch := make(chan []byte)
	go func() {
		res, _ := initializeSync(id, width, difficulty) // TODO @noam: handle error
		ch <- res
	}()
	return ch
}

func initializeSync(id []byte, width uint64, difficulty []byte) ([]byte, error) {
	labelsWriter, err := persistence.NewPostLabelsWriter(id)
	if err != nil {
		return nil, err
	}
	merkleTree := merkle.NewTree()
	var cnt, labelsFound uint64 = 0, 0
	for labelsFound < width {
		l := util.NewLabel(cnt)
		if util.CalcHash(id, l).IsLessThan(difficulty) {
			err := labelsWriter.Write(l)
			if err != nil {
				return nil, err
			}
			merkleTree.AddLeaf(l)
			labelsFound++
		}
		if cnt == math.MaxUint64 && labelsFound < width {
			panic("Out of counter space!") // TODO @noam: handle gracefully?
		}
		cnt++
	}
	err = labelsWriter.Close()
	if err != nil {
		return nil, err
	}
	root := merkleTree.Root()
	fmt.Printf("\n"+
		"🔹  Constructed list of %v PoST labels.\n"+
		"🔹  Number of random oracle calls: %v\n"+
		"🔹  Merkle root: %v\n"+
		"\n", labelsFound, cnt, hex.EncodeToString(root))
	return root, nil
}
