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
	// How many nonces to try in parallel.
	nonces uint
	// How many threads to use to generate a proof.
	// 0 - automatically detect
	threads uint
}

func (o *option) validate() error {
	if o.datadir == "" {
		return errors.New("`datadir` is required")
	}
	if o.nonces == 0 {
		return errors.New("`nonces` must be greater than 0")
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

		if ok, err := initCompleted(datadir, m.NumUnits, cfg.LabelsPerUnit); err != nil {
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

func WithPowScryptParams(params config.ScryptParams) OptionFunc {
	return func(o *option) error {
		o.powScrypt = params
		return nil
	}
}

func WithNonces(nonces uint) OptionFunc {
	return func(o *option) error {
		if nonces == 0 {
			return errors.New("`nonces` must be greater than 0")
		}
		o.nonces = nonces
		return nil
	}
}

func WithThreads(threads uint) OptionFunc {
	return func(o *option) error {
		o.threads = threads
		return nil
	}
}
