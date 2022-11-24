package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

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
	cfg   = config.DefaultConfig()
	opts  = config.DefaultInitOpts()
	id    []byte
	reset bool
)

func parseFlags() error {
	flag.StringVar(&opts.DataDir, "datadir", opts.DataDir, "filesystem datadir path")

	var numUnits uint64
	flag.Uint64Var(&numUnits, "numUnits", uint64(opts.NumUnits), "number of units") // workaround the missing type support for uint32
	opts.NumUnits = uint32(numUnits)

	flag.BoolVar(&reset, "reset", false, "whether to reset the datadir before starting")
	idHex := flag.String("id", "", "id (public key) in hex")

	// TODO: expose more cfg/opts to cmd flags
	flag.Parse()

	if *idHex != "" {
		var err error
		id, err = hex.DecodeString(*idHex)
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		return nil
	}

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate identity: %w", err)
	}

	id = pub
	log.Printf("generated id: %x\n", id)

	return saveKey(priv) // The key will need to be loaded in clients for the data to be usable.
}

// TODO(mafa): add "WithId" and "WithCommitmentATX" options to the initializer and do this within the initializer.
func GetCommitmentBytes(id []byte, commitmentAtxId []byte) []byte {
	h := sha256.Sum256(append(id, commitmentAtxId...))
	return h[:]
}

func main() {
	if err := parseFlags(); err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	atxId := make([]byte, 32) // TODO(mafa): get this as a flag like the id.
	commitment := GetCommitmentBytes(id, atxId)

	opts.ComputeProviderID = gpu.CPUProviderID() // TODO(mafa): select best provider.

	init, err := initialization.NewInitializer(
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithCommitment(commitment),
		//initialization.WithLogger() // TODO: add wrapper for zap logger.
	)
	if err != nil {
		log.Fatal(err)
	}

	if reset {
		if err := init.Reset(); err != nil {
			log.Fatalf("reset error: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := init.Initialize(ctx); err != nil {
		if err == shared.ErrInitCompleted {
			log.Println(err)
			return
		}
		if err == context.Canceled {
			log.Println("initialization interrupted")
			return
		}
		log.Fatalf("initialization error: %v", err)
	}

	// Initialization is done. Try to generate a valid proof as a sanity check.
	prover, err := proving.NewProver(cfg, opts.DataDir, commitment)
	if err != nil {
		log.Fatal(err)
	}
	// prover.SetLogger() // TODO: add wrapper for zap logger.
	proof, proofMetadata, err := prover.GenerateProof(shared.ZeroChallenge)
	if err != nil {
		log.Fatalf("proof generation error: %v", err)
	}
	if err := verifying.Verify(proof, proofMetadata); err != nil {
		log.Fatalf("failed to verify test proof: %v", err)
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
