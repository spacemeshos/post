package initialization

import (
	"github.com/spacemeshos/go-spacemesh/log"
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
		res, err := initializeSync(id, width, difficulty)
		if err != nil {
			log.Error(err.Error())
			close(ch)
		}
		ch <- res
	}()
	return ch
}

func initializeSync(id []byte, width uint64, difficulty []byte) ([]byte, error) {
	labelsWriter, err := persistence.NewPostLabelsFileWriter(id)
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
	log.With().Info("completed PoST label list construction",
		log.Uint64("number_of_labels", labelsFound),
		log.Uint64("number_of_oracle_calls", cnt),
	)
	return merkleTree.Root(), nil
}
