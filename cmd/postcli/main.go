package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	baseLog "log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/minio/sha256-simd"

	"github.com/spacemeshos/ed25519"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/gpu"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

var (
	cfg             = config.DefaultConfig()
	opts            = config.DefaultInitOpts()
	log             = logger{}
	printProviders  bool
	printConfig     bool
	id              []byte
	commitmentAtxId []byte
	reset           bool
)

func parseFlags() error {
	flag.BoolVar(&printProviders, "printProviders", false, "print the list of compute providers")
	flag.BoolVar(&printProviders, "printConfig", false, "print the used config and options")
	flag.StringVar(&opts.DataDir, "datadir", opts.DataDir, "filesystem datadir path")
	flag.IntVar(&opts.ComputeProviderID, "provider", opts.ComputeProviderID, "compute provider id (required)")
	flag.Uint64Var(&cfg.LabelsPerUnit, "labelsPerUnit", cfg.LabelsPerUnit, "the number of labels per unit")
	flag.BoolVar(&reset, "reset", false, "whether to reset the datadir before starting")
	idHex := flag.String("id", "", "miner's id (public key), in hex (will be auto-generated if not provided)")
	commitmentAtxIdHex := flag.String("commitmentAtxId", "", "commitment atx id, in hex (required)")

	var numUnits uint64
	flag.Uint64Var(&numUnits, "numUnits", uint64(opts.NumUnits), "number of units") // workaround the missing type support for uint32
	opts.NumUnits = uint32(numUnits)

	flag.Parse()

	if opts.ComputeProviderID < 0 {
		baseLog.Fatal("-provider flag is required")
	}

	if *commitmentAtxIdHex == "" {
		baseLog.Fatalf("-commitmentAtxId flag is required")
	}
	var err error
	commitmentAtxId, err = hex.DecodeString(*commitmentAtxIdHex)
	if err != nil {
		return fmt.Errorf("invalid commitmentAtxId: %w", err)
	}

	if *idHex == "" {
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			return fmt.Errorf("failed to generate identity: %w", err)
		}
		id = pub
		log.Info("cli: generated id: %x", id)
		saveKey(priv) // The key will need to be loaded in clients for the PoST data to be usable.
	} else {
		var err error
		id, err = hex.DecodeString(*idHex)
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
	}

	return nil
}

func main() {
	if err := parseFlags(); err != nil {
		log.Panic("cli: failed to parse flags: %v", err)
	}

	if printProviders {
		spew.Dump(gpu.Providers())
		return
	}

	if printConfig {
		spew.Dump(cfg)
		spew.Dump(opts)
		return
	}

	commitment := GetCommitmentBytes(id, commitmentAtxId)
	log.Info("cli: commitment: %x", commitment)
	init, err := initialization.NewInitializer(
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithCommitment(commitment),
		initialization.WithLogger(log),
	)
	if err != nil {
		log.Panic(err.Error())
	}

	if reset {
		if err := init.Reset(); err != nil {
			log.Panic("reset error: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := init.Initialize(ctx); err != nil {
		if err == shared.ErrInitCompleted {
			log.Panic(err.Error())
			return
		}
		if err == context.Canceled {
			log.Info("cli: initialization interrupted")
			return
		}
		log.Error("cli: initialization error: %v", err)
		return
	}

	log.Info("cli: initialization completed, generating a proof as a sanity test")
	prover, err := proving.NewProver(cfg, opts.DataDir, commitment)
	if err != nil {
		log.Panic(err.Error())
	}
	prover.SetLogger(log)
	proof, proofMetadata, err := prover.GenerateProof(shared.ZeroChallenge)
	if err != nil {
		log.Panic("proof generation error: %v", err)
	}
	if err := verifying.Verify(proof, proofMetadata); err != nil {
		log.Panic("failed to verify test proof: %v", err)
	}

	log.Info("cli: proof is valid")
}

// TODO(mafa): add "WithId" and "WithCommitmentATX" options to the initializer and do this within the initializer.
func GetCommitmentBytes(id []byte, commitmentAtxId []byte) []byte {
	h := sha256.Sum256(append(id, commitmentAtxId...))
	return h[:]
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

func (l logger) Info(msg string, args ...interface{})    { baseLog.Printf("\tINFO\t"+msg, args...) }
func (l logger) Debug(msg string, args ...interface{})   { baseLog.Printf("\tDEBUG\t"+msg, args...) }
func (l logger) Warning(msg string, args ...interface{}) { baseLog.Printf("\tWARN\t"+msg, args...) }
func (l logger) Error(msg string, args ...interface{})   { baseLog.Printf("\tERROR\t"+msg, args...) }
func (l logger) Panic(msg string, args ...interface{})   { baseLog.Fatalf("\tPANIC\t"+msg, args...) }
