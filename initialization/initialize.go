package initialization

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/post-private/persistence"
	"github.com/spacemeshos/post-private/util"
	"math"
)

const maxWidth = 1 << 50 // at 1 byte per label, this would be 1 peta-byte of storage

// Initialize takes an id (public key), width (number of labels) and difficulty (hash to use as upper bound for labels).
// The merkle root is passed on the results channel.
func Initialize(id []byte, width uint64, difficulty []byte) ([]byte, error) {
	merkleRoot, err := initialize(id, width, difficulty)
	if err != nil {
		log.Error(err.Error())
	}
	return merkleRoot, err
}

func initialize(id []byte, width uint64, difficulty []byte) ([]byte, error) {
	if width > maxWidth {
		return nil, fmt.Errorf("requested width (%d) is larger than supported width (%d)", width, maxWidth)
	}
	labelsWriter, err := persistence.NewPostLabelsFileWriter(id)
	if err != nil {
		return nil, err
	}
	merkleTree := merkle.NewTree()
	var cnt, labelsFound uint64 = 0, 0
	for {
		l := util.NewLabel(cnt)
		if util.CalcHash(id, l).IsLessThan(difficulty) {
			err := labelsWriter.Write(l)
			if err != nil {
				return nil, err
			}
			merkleTree.AddLeaf(l)
			labelsFound++
			if labelsFound == width {
				break
			}
			if labelsFound%5000000 == 0 {
				log.Info("found %v labels", labelsFound)
			}
		}
		if cnt == math.MaxUint64 {
			return nil, errors.New("out of counter space")
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
