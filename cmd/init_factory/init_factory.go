package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	cfg   = config.DefaultConfig()
	opts  = config.DefaultInitOpts()
	id    []byte
	reset bool
)

func init() {
	flag.StringVar(&opts.DataDir, "datadir", opts.DataDir, "filesystem datadir path")
	//flag.Uint64(&opts.NumUnits, "numunits", opts.NumUnits, "number of units") // TODO: workaround the missing type support for uint32
	// TODO: expose more cfg/opts to cmd flags
	flag.BoolVar(&reset, "reset", false, "whether to reset the datadir before starting")
	idHex := flag.String("id", "", "id (public key) in hex")
	flag.Parse()

	if *idHex == "" {
		pub, priv, err := ed25519.GenerateKey(nil) // TODO: verify whether this is the current key generator.
		if err != nil {
			log.Fatalf("generate key error: %v", err)
		}
		log.Printf("generated id: %x", id)
		saveKey(priv) // The key will need to be loaded in clients for the data to be usable.
		id = pub
	} else {
		var err error
		id, err = hex.DecodeString(*idHex)
		if err != nil {
			log.Fatalf("id hex decode error: %v", err)
		}
	}
}

func main() {
	commitment := id // TODO: expose commitmentATX or commitment hash (commitmentATX ++ id) to cmd flags.

	init, err := initialization.NewInitializer(
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithCommitment(commitment),
		//initialization.WithLogger() // TODO: set a custom logger
	)
	if err != nil {
		log.Fatal(err)
	}

	if reset {
		if err := init.Reset(); err != nil {
			log.Fatalf("reset error: %v", err)
		}
	}

	if err := init.Initialize(context.TODO()); err != nil {
		if err == shared.ErrInitCompleted {
			log.Print(err)
			return
		}
		log.Fatalf("initialization error: %v", err)
	}

	// Initialization is done. Try to generate a valid proof as a sanity check.
	prover, err := proving.NewProver(cfg, opts.DataDir, commitment)
	if err != nil {
		log.Fatal(err)
	}
	// prover.SetLogger() // TODO: set a custom logger
	proof, proofMetadata, err := prover.GenerateProof(shared.ZeroChallenge)
	if err != nil {
		log.Fatalf("proof generation error: %v", err)
	}
	if err := verifying.Verify(proof, proofMetadata); err != nil {
		log.Fatal(err)
	}
}

func saveKey(key []byte) {
	err := os.MkdirAll(opts.DataDir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("mkdir error: %v", err)
	}

	err = ioutil.WriteFile(filepath.Join(opts.DataDir, "key.bin"), key, shared.OwnerReadWrite)
	if err != nil {
		log.Fatalf("key write to disk error: %v", err)
	}
}
