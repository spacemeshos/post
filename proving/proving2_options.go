package proving

import (
	"errors"
	"io"

	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
)

type option struct {
	// reader is the source of labels to be used for generating a proof.
	reader io.Reader

	nodeId          []byte
	commitmentAtxId []byte

	numUnits uint32
}

func (o *option) validate() error {
	if o.reader == nil {
		return errors.New("`reader` is required")
	}
	return nil
}

type OptionFunc func(*option) error

// WithDataSource sets the data source to use for the proof.
func WithDataSource(cfg Config, nodeId, commitmentAtxId []byte, datadir string) OptionFunc {
	return func(o *option) error {
		m, err := initialization.LoadMetadata(datadir)
		if err != nil {
			return err
		}

		if err := verifyMetadata(m, cfg, datadir, nodeId, commitmentAtxId); err != nil {
			return err
		}

		if ok, err := initCompleted(datadir, m.NumUnits, cfg.BitsPerLabel, cfg.LabelsPerUnit); err != nil {
			return err
		} else if !ok {
			return shared.ErrInitNotCompleted
		}

		reader, err := persistence.NewLabelsReader(datadir, uint(cfg.BitsPerLabel))
		if err != nil {
			return err
		}

		o.reader = reader
		o.nodeId = nodeId
		o.commitmentAtxId = commitmentAtxId
		o.numUnits = m.NumUnits
		return nil
	}
}

// withLabelsReader is an option that allows the caller to provide a reader for labels.
// TODO(mafa): at the moment this is intended for testing purposes only, but will eventually replace `WithDataSource`.
func withLabelsReader(reader io.Reader, nodeId, commitmentAtxId []byte, numUnits uint32) OptionFunc {
	return func(o *option) error {
		o.reader = reader
		o.nodeId = nodeId
		o.commitmentAtxId = commitmentAtxId
		o.numUnits = numUnits
		return nil
	}
}
