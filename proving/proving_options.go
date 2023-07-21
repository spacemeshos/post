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
	powFlags        config.PowFlags
	// How many nonces to try in parallel.
	nonces uint
	// How many threads to use to generate a proof.
	// 0 - automatically detect
	threads uint

	powCreatorId []byte
}

func (o *option) validate() error {
	if o.datadir == "" {
		return errors.New("`datadir` is required")
	}
	if o.nonces == 0 {
		return errors.New("`nonces` must be greater than 0")
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

func WithPowFlags(flags config.PowFlags) OptionFunc {
	return func(o *option) error {
		o.powFlags = flags
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

func WithPowCreator(id []byte) OptionFunc {
	return func(o *option) error {
		if len(id) != 32 {
			return errors.New("pow creator id must be 32 bytes")
		}
		o.powCreatorId = id
		return nil
	}
}
