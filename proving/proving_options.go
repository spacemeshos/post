package proving

import (
	"errors"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/shared"
)

type option struct {
	datadir         string
	nodeId          []byte
	commitmentAtxId []byte
	numUnits        uint32
	powScrypt       config.ScryptParams
}

func (o *option) validate() error {
	if o.datadir == "" {
		return errors.New("`datadir` is required")
	}
	if err := o.powScrypt.Validate(); err != nil {
		return err
	}
	return nil
}

type OptionFunc func(*option) error

// WithDataSource sets the data source to use for the proof.
func WithDataSource(cfg config.Config, nodeId, commitmentAtxId []byte, datadir string) OptionFunc {
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

		o.datadir = datadir
		o.nodeId = nodeId
		o.commitmentAtxId = commitmentAtxId
		o.numUnits = m.NumUnits
		return nil
	}
}

func defaultOpts() *option {
	return &option{
		powScrypt: config.DefaultPowScryptParams(),
	}
}

func WithPowScryptParams(params config.ScryptParams) OptionFunc {
	return func(o *option) error {
		o.powScrypt = params
		return nil
	}
}
