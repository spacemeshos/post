package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

const edKeyFileName = "key.bin"

var (
	cfg                = config.DefaultConfig()
	opts               = config.DefaultInitOpts()
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
	flag.IntVar(&opts.ProviderID, "provider", opts.ProviderID, "compute provider id (required)")
	flag.Uint64Var(&cfg.LabelsPerUnit, "labelsPerUnit", cfg.LabelsPerUnit, "the number of labels per unit")
	flag.BoolVar(&reset, "reset", false, "whether to reset the datadir before starting")
	flag.StringVar(&idHex, "id", "", "miner's id (public key), in hex (will be auto-generated if not provided)")
	flag.StringVar(&commitmentAtxIdHex, "commitmentAtxId", "", "commitment atx id, in hex (required)")
	numUnits := flag.Uint64("numUnits", uint64(opts.NumUnits), "number of units")
	flag.Parse()

	opts.NumUnits = uint32(*numUnits) // workaround the missing type support for uint32
}

func processFlags() error {
	if opts.ProviderID < 0 {
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
		log.Printf("cli: generated id %x\n", id)
		return saveKey(priv)
	}
	id, err = hex.DecodeString(idHex)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return nil
}

func main() {
	parseFlags()

	if printProviders {
		providers, err := postrs.OpenCLProviders()
		if err != nil {
			log.Fatalln("failed to get OpenCL providers", err)
		}
		spew.Dump(providers)
		return
	}

	if printConfig {
		spew.Dump(cfg)
		spew.Dump(opts)
		return
	}

	if err := processFlags(); err != nil {
		log.Fatalln("failed to process flags", err)
	}

	zapLog, err := zap.NewProduction()
	if err != nil {
		log.Fatalln("failed to initialize zap logger:", err)
	}

	init, err := initialization.NewInitializer(
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithNodeId(id),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithLogger(zapLog),
	)
	if err != nil {
		log.Panic(err.Error())
	}

	if reset {
		if err := init.Reset(); err != nil {
			log.Fatalln("reset error", err)
		}
		log.Println("cli: reset completed")
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
		log.Println("cli: initialization interrupted")
		return
	case err != nil:
		log.Println("cli: initialization error", err)
		return
	}

	log.Println("cli: initialization completed")

	if genProof {
		log.Println("cli: generating proof as a sanity test")

		proof, proofMetadata, err := proving.Generate(ctx, shared.ZeroChallenge, cfg, zapLog, proving.WithDataSource(cfg, id, commitmentAtxId, opts.DataDir))
		if err != nil {
			log.Fatalln("proof generation error", err)
		}
		verifier, err := verifying.NewProofVerifier()
		if err != nil {
			log.Fatalln("failed to create verifier", err)
		}
		defer verifier.Close()
		if err := verifier.Verify(proof, proofMetadata, cfg, zapLog); err != nil {
			log.Fatalln("failed to verify test proof", err)
		}

		log.Println("cli: proof is valid")
	}
}

func saveKey(key ed25519.PrivateKey) error {
	if err := os.MkdirAll(opts.DataDir, 0o700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdir error: %w", err)
	}

	filename := filepath.Join(opts.DataDir, edKeyFileName)
	if err := os.WriteFile(filename, []byte(hex.EncodeToString(key)), 0o600); err != nil {
		return fmt.Errorf("key write to disk error: %w", err)
	}
	return nil
}
