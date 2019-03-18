package initialization

import (
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/post-private/config"
	"github.com/spacemeshos/post-private/labels"
	"github.com/spacemeshos/post-private/persistence"
	"github.com/spacemeshos/post-private/util"
)

const maxWidth = 1 << 50 // at 1 byte per label, this would be 1 peta-byte of storage

// Initialize takes an id (public key), width (number of labels) and difficulty (hash to use as upper bound for labels).
// The merkle root is passed on the results channel.
func Initialize(id []byte, width uint64) ([]byte, error) {
	labelsWriter, err := persistence.NewPostLabelsFileWriter(id)
	if err != nil {
		return nil, err
	}
	merkleRoot, err := initialize(id, width, &labelsWriter)
	if err2 := labelsWriter.Close(); err2 != nil {
		if err != nil {
			err = fmt.Errorf("%v, %v", err, err2)
		} else {
			err = err2
		}
	}
	if err != nil {
		err = fmt.Errorf("failed to initialize post: %v", err)
		log.Error(err.Error())
	}
	return merkleRoot, err
}

type postLabelsWriter interface {
	Write(label util.Label) error
}

func initialize(id []byte, width uint64, labelsWriter postLabelsWriter) ([]byte, error) {
	if width > maxWidth {
		return nil, fmt.Errorf("requested width (%d) is larger than supported width (%d)", width, maxWidth)
	}
	// TODO @noam: save cache
	merkleTree := merkle.NewTree(merkle.GetSha256Parent)
	for position := uint64(0); position < width; position++ {
		lg := labels.CalcLabelGroup(id, position)
		err := labelsWriter.Write(lg)
		if err != nil {
			return nil, err
		}
		err = merkleTree.AddLeaf(lg)
		if err != nil {
			return nil, err
		}
		if (position+1)%config.Post.LogEveryXLabels == 0 {
			log.Info("found %v labels", position+1)
		}
	}

	log.With().Info("completed PoST label list construction")
	return merkleTree.Root(), nil
}
