package main

import (
	"encoding/hex"
	"flag"
	"github.com/spacemeshos/ed25519"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/validation"
	smlog "github.com/spacemeshos/smutil/log"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	cfg   = config.DefaultConfig()
	id    []byte
	reset bool
)

func init() {
	flag.StringVar(&cfg.DataDir, "datadir", cfg.DataDir, "filesystem datadir path")
	flag.Uint64Var(&cfg.NumLabels, "numlabels", cfg.NumLabels, "number of labels")
	flag.UintVar(&cfg.LabelSize, "labelsize", cfg.LabelSize, "label size")
	flag.UintVar(&cfg.NumFiles, "numfiles", cfg.NumFiles, "number of files")
	flag.BoolVar(&reset, "reset", false, "whether to reset the given id initialization folder before start initializing")
	idHex := flag.String("id", "", "id (public key) in hex")

	flag.Parse()

	if *idHex == "" {
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatalf("generate key failure: %v", err)
		}

		id = pub
		saveKey(priv)

		smlog.Info("generated id: %x", id)
	} else {
		var err error
		id, err = hex.DecodeString(*idHex)
		if err != nil {
			log.Fatalf("id hex decode failure: %v", err)
		}
	}
}

func main() {
	init, err := initialization.NewInitializer(cfg, id)
	if err != nil {
		log.Fatal(err)
	}
	init.SetLogger(smlog.AppLog)

	if reset {
		if err := init.Reset(); err != nil {
			log.Fatalf("reset failure: %v", err)
		}
	}

	if err := init.Initialize(); err != nil {
		if err == shared.ErrInitCompleted {
			log.Print(err)
			return
		}
		log.Fatalf("failed to initialize: %v", err)
	}

	prover, err := proving.NewProver(cfg, id)
	if err != nil {
		log.Fatal(err)
	}
	prover.SetLogger(smlog.AppLog)
	proof, err := prover.GenerateProof(shared.ZeroChallenge)
	if err != nil {
		log.Fatalf("failed to generate proof: %v", err)
	}

	validator, err := validation.NewValidator(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := validator.Validate(id, proof); err != nil {
		log.Fatal(err)
	}

	if err := shared.PersistProof(cfg.DataDir, id, proof.Challenge, proof.Encode()); err != nil {
		log.Fatalf("failed to persist proof: %v", err)
	}
}

func saveKey(key []byte) {
	dir := shared.GetInitDir(cfg.DataDir, id)
	err := os.MkdirAll(dir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("dir creation failure: %v", err)
	}

	err = ioutil.WriteFile(filepath.Join(dir, "key.bin"), key, shared.OwnerReadWrite)
	if err != nil {
		log.Fatalf("write to disk failure: %v", err)
	}
}
