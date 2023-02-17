package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	baseLog "log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/spacemeshos/ed25519"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

var (
	cfg                = config.DefaultConfig()
	opts               = config.DefaultInitOpts()
	log                = logger{}
	printProviders     bool
	printConfig        bool
	genProof           bool
	idHex              string
	id                 []byte
	commitmentAtxIdHex string
	commitmentAtxId    []byte
	reset              bool
)

func parseFlags() {
	flag.BoolVar(&printProviders, "printProviders", false, "print the list of compute providers")
	flag.BoolVar(&printConfig, "printConfig", false, "print the used config and options")
	flag.BoolVar(&genProof, "genproof", false, "generate proof as a sanity test, after initialization")
	flag.StringVar(&opts.DataDir, "datadir", opts.DataDir, "filesystem datadir path")
	flag.Uint64Var(&opts.MaxFileSize, "maxFileSize", opts.MaxFileSize, "max file size")
	flag.IntVar(&opts.ComputeProviderID, "provider", opts.ComputeProviderID, "compute provider id (required)")
	flag.Uint64Var(&cfg.LabelsPerUnit, "labelsPerUnit", cfg.LabelsPerUnit, "the number of labels per unit")
	flag.BoolVar(&reset, "reset", false, "whether to reset the datadir before starting")
	flag.StringVar(&idHex, "id", "", "miner's id (public key), in hex (will be auto-generated if not provided)")
	flag.StringVar(&commitmentAtxIdHex, "commitmentAtxId", "", "commitment atx id, in hex (required)")
	numUnits := flag.Uint64("numUnits", uint64(opts.NumUnits), "number of units")
	flag.Parse()

	opts.NumUnits = uint32(*numUnits) // workaround the missing type support for uint32
}

func processFlags() error {
	if opts.ComputeProviderID < 0 {
		return errors.New("-provider flag is required")
	}

	if commitmentAtxIdHex == "" {
		return errors.New("-commitmentAtxId flag is required")
	}
	var err error
	commitmentAtxId, err = hex.DecodeString(commitmentAtxIdHex)
	if err != nil {
		return fmt.Errorf("invalid commitmentAtxId: %w", err)
	}

	if idHex == "" {
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			return fmt.Errorf("failed to generate identity: %w", err)
		}
		id = pub
		log.Info("cli: generated id: %x", id)
		saveKey(priv) // The key will need to be loaded in clients for the PoST data to be usable.
	} else {
		var err error
		id, err = hex.DecodeString(idHex)
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
	}

	return nil
}

func main() {
	parseFlags()

	if printProviders {
		spew.Dump(gpu.Providers())
		return
	}

	if printConfig {
		spew.Dump(cfg)
		spew.Dump(opts)
		return
	}

	if err := processFlags(); err != nil {
		log.Panic("cli: %v", err)
	}

	init, err := initialization.NewInitializer(
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithNodeId(id),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithLogger(log),
	)
	if err != nil {
		log.Panic(err.Error())
	}

	if reset {
		if err := init.Reset(); err != nil {
			log.Panic("reset error: %v", err)
		}
		log.Info("cli: reset completed")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err = init.Initialize(ctx)
	switch {
	case errors.Is(err, shared.ErrInitCompleted):
		log.Panic(err.Error())
		return
	case errors.Is(err, context.Canceled):
		log.Info("cli: initialization interrupted")
		return
	case err != nil:
		log.Error("cli: initialization error: %v", err)
		return
	}

	log.Info("cli: initialization completed")

	if genProof {
		log.Info("cli: generating proof as a sanity test")

		proof, proofMetadata, err := proving.Generate(ctx, shared.ZeroChallenge, cfg, log, proving.WithDataSource(cfg, id, commitmentAtxId, opts.DataDir))
		if err != nil {
			log.Panic("proof generation error: %v", err)
		}
		if err := verifying.VerifyNew(proof, proofMetadata); err != nil {
			log.Panic("failed to verify test proof: %v", err)
		}

		log.Info("cli: proof is valid")
	}
}

func saveKey(key []byte) error {
	if err := os.MkdirAll(opts.DataDir, shared.OwnerReadWriteExec); err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdir error: %w", err)
	}

	if err := os.WriteFile(filepath.Join(opts.DataDir, "key.bin"), key, shared.OwnerReadWrite); err != nil {
		return fmt.Errorf("key write to disk error: %w", err)
	}
	return nil
}

type logger struct{}

func (l logger) Info(msg string, args ...any)    { baseLog.Printf("\tINFO\t"+msg, args...) }
func (l logger) Debug(msg string, args ...any)   { baseLog.Printf("\tDEBUG\t"+msg, args...) }
func (l logger) Warning(msg string, args ...any) { baseLog.Printf("\tWARN\t"+msg, args...) }
func (l logger) Error(msg string, args ...any)   { baseLog.Printf("\tERROR\t"+msg, args...) }
func (l logger) Panic(msg string, args ...any)   { baseLog.Fatalf(msg, args...) }
