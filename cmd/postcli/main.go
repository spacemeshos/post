package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/verifying"
)

var (
	cfg  = config.MainnetConfig()
	opts = config.MainnetInitOpts()

	searchForNonce bool
	printProviders bool
	printNumFiles  bool
	printConfig    bool
	genProof       bool

	verifyPos bool
	fraction  float64

	idHex              string
	commitmentAtxIdHex string
	reset              bool
	numUnits           uint64

	yes      bool
	logLevel zapcore.Level

	flagSet = make(map[string]bool)

	ErrKeyFileExists = errors.New("key file already exists")
)

func parseFlags() {
	flag.BoolVar(&verifyPos, "verify", false, "verify initialized data")
	flag.Float64Var(&fraction, "fraction", 0.2, "how much % of POS data to verify. Sane values are < 1.0")

	flag.BoolVar(&yes, "yes", false, "confirm potentially dangerous actions")
	flag.TextVar(&logLevel, "logLevel", zapcore.InfoLevel, "log level (debug, info, warn, error, dpanic, panic, fatal)")

	flag.BoolVar(&searchForNonce, "searchForNonce", false, "search for VRF nonce in already initialized files")
	flag.BoolVar(&printProviders, "printProviders", false, "print the list of compute providers")
	flag.BoolVar(&printNumFiles, "printNumFiles", false, "print the total number of files that would be initialized")
	flag.BoolVar(&printConfig, "printConfig", false, "print the used config and options")
	flag.BoolVar(&genProof, "genproof", false, "generate proof as a sanity test, after initialization")

	flag.StringVar(&opts.DataDir, "datadir", opts.DataDir, "filesystem datadir path")
	flag.Uint64Var(&opts.MaxFileSize, "maxFileSize", opts.MaxFileSize, "max file size")
	var providerID uint64
	flag.Uint64Var(&providerID, "provider", 0, "compute provider id (required)")
	flag.Uint64Var(&cfg.LabelsPerUnit, "labelsPerUnit", cfg.LabelsPerUnit, "the number of labels per unit")
	flag.UintVar(&opts.Scrypt.N, "scryptN", opts.Scrypt.N, "scrypt N parameter")
	flag.BoolVar(&reset, "reset", false, "whether to reset the datadir before starting")
	flag.StringVar(&idHex, "id", "", "miner's id (public key), in hex (will be auto-generated if not provided)")
	flag.StringVar(&commitmentAtxIdHex, "commitmentAtxId", "", "commitment atx id, in hex (required)")
	flag.Uint64Var(&numUnits, "numUnits", 0, "number of units (required)")

	flag.IntVar(&opts.FromFileIdx, "fromFile", 0, "index of the first file to init (inclusive)")
	var to int
	flag.IntVar(&to, "toFile", 0, "index of the last file to init (inclusive). Will init to the end of declared space if not provided.")
	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		flagSet[f.Name] = true
	})

	if flagSet["toFile"] {
		opts.ToFileIdx = &to
	}

	opts.ProviderID = new(uint32)
	*opts.ProviderID = uint32(providerID)
	opts.NumUnits = uint32(numUnits)
}

func askForConfirmation() {
	if yes {
		return
	}

	log.Println("Are you sure you want to continue (y/N)?")

	var answer string
	_, err := fmt.Scanln(&answer)
	if err != nil || !(answer == "y" || answer == "Y") {
		log.Fatal("Aborting")
	}
}

func processFlags() {
	meta, err := initialization.LoadMetadata(opts.DataDir)
	switch {
	case errors.Is(err, initialization.ErrStateMetadataFileMissing):
	case err != nil:
		log.Fatalln("failed to load metadata:", err)
	default:
		if idHex == "" {
			idHex = hex.EncodeToString(meta.NodeId)
		} else if idHex != hex.EncodeToString(meta.NodeId) {
			log.Println("WARNING: it appears that", opts.DataDir, "was previously initialized with a different `id` value.")
			log.Println("\tCurrent value:", hex.EncodeToString(meta.NodeId))
			log.Println("\tValue passed to postcli:", idHex)
			log.Fatalln("aborting")
		}
	}

	edKeyFileName, err := findKeyFile(opts.DataDir)
	if err != nil {
		log.Fatalln("failed to find identity file:", err)
	}
	key, err := loadKey(edKeyFileName)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		log.Fatalln("failed to load key:", err)
	case meta != nil && !bytes.Equal(meta.NodeId, key):
		log.Fatalln("WARNING: inconsistent state:", edKeyFileName, "file does not match metadata in", opts.DataDir)
	default:
		if idHex == "" {
			idHex = hex.EncodeToString(key)
		} else if idHex != hex.EncodeToString(key) {
			log.Println("WARNING: it appears that", opts.DataDir, "was previously initialized with a generated key.")
			log.Println("The provided id does not match the generated key.")
			log.Println("\tCurrent value:", hex.EncodeToString(key))
			log.Println("\tValue passed to postcli:", idHex)
			log.Fatalln("aborting")
		}
	}

	// we require the user to explicitly pass numUnits to avoid erasing existing data
	if !flagSet["numUnits"] && meta != nil {
		log.Fatalln("-numUnits must be specified to perform initialization.")
	}

	if flagSet["numUnits"] && meta != nil && numUnits != uint64(meta.NumUnits) {
		log.Println("WARNING: it appears that", opts.DataDir, "was previously initialized with a different `numUnits` value.")
		log.Println("\tCurrent value:", meta.NumUnits)
		log.Println("\tValue passed to postcli:", numUnits)
		if (numUnits < uint64(meta.NumUnits)) && !yes {
			log.Println("CONTINUING MIGHT ERASE EXISTING DATA. MAKE ABSOLUTELY SURE YOU SPECIFY THE CORRECT VALUE.")
		}
		askForConfirmation()
	}

	if flagSet["numUnits"] && (numUnits < uint64(cfg.MinNumUnits) || numUnits > uint64(cfg.MaxNumUnits)) {
		log.Println("WARNING: numUnits is outside of range valid for mainnet (min:",
			cfg.MinNumUnits, "max:", cfg.MaxNumUnits, ")")
		if !yes {
			log.Println("CONTINUING WILL INITIALIZE DATA INCOMPATIBLE WITH MAINNET. MAKE ABSOLUTELY SURE YOU WANT TO DO THIS.")
		}
		askForConfirmation()
		cfg.MinNumUnits = uint32(numUnits)
		cfg.MaxNumUnits = uint32(numUnits)
	}

	if !flagSet["commitmentAtxId"] && meta != nil {
		log.Fatalln("-commitmentAtxId must be specified to perform initialization.")
	}

	if flagSet["commitmentAtxId"] {
		commitmentAtxId, err := hex.DecodeString(commitmentAtxIdHex)
		if err != nil {
			log.Println("invalid commitmentAtxId:", err)
		}
		if meta != nil && !bytes.Equal(commitmentAtxId, meta.CommitmentAtxId) {
			log.Println("WARNING: it appears that", opts.DataDir, "was previously initialized with a different `commitmentAtxId` value.")
			log.Println("\tCurrent value:", hex.EncodeToString(meta.CommitmentAtxId))
			log.Println("\tValue passed to postcli:", commitmentAtxIdHex)
			log.Fatalln("aborting")
		}
	}

	if !flagSet["provider"] {
		log.Fatalln("-provider flag is required")
	}

	if flagSet["labelsPerUnit"] && (cfg.LabelsPerUnit != config.MainnetConfig().LabelsPerUnit) {
		log.Println("WARNING: labelsPerUnit is set to a non-default value.")
		log.Println("If you're trying to initialize for mainnet, please remove the -labelsPerUnit flag")
		if !yes {
			log.Println("CONTINUING WILL INITIALIZE DATA INCOMPATIBLE WITH MAINNET. MAKE ABSOLUTELY SURE YOU WANT TO DO THIS.")
		}
		askForConfirmation()
	}

	if flagSet["scryptN"] && (opts.Scrypt.N != config.MainnetInitOpts().Scrypt.N) {
		log.Println("WARNING: scryptN is set to a non-default value.")
		log.Println("If you're trying to initialize for mainnet, please remove the -scryptN flag")
		if !yes {
			log.Println("CONTINUING WILL INITIALIZE DATA INCOMPATIBLE WITH MAINNET. MAKE ABSOLUTELY SURE YOU WANT TO DO THIS.")
		}
		askForConfirmation()
	}

	if (opts.FromFileIdx != 0 || opts.ToFileIdx != nil) && idHex == "" {
		log.Fatalln("-id flag is required when using -fromFile or -toFile")
	}

	if idHex == "" {
		log.Println("cli: generating new identity")
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatalln("failed to generate identity:", err)
		}
		edKeyFileName, err := saveKey(priv)
		if err != nil {
			log.Fatalln("failed to save identity:", err)
		}
		idHex = hex.EncodeToString(pub)
		log.Println("generated key in", edKeyFileName)
		log.Println("copy this file to the `data/identities` directory of your node")
		return
	}
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

	if printNumFiles {
		totalFiles := opts.TotalFiles(cfg.LabelsPerUnit)
		log.Println(totalFiles)
		return
	}

	if printConfig {
		spew.Dump(cfg)
		spew.Dump(opts)
		return
	}

	zapCfg := zap.Config{
		Level:    zap.NewAtomicLevelAt(logLevel),
		Encoding: "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			MessageKey:     "M",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapCfg.Build()
	if err != nil {
		log.Fatalln("failed to initialize zap logger:", err)
	}

	if verifyPos {
		cmdVerifyPos(opts, fraction, logger)
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if searchForNonce {
		nonce, label, err := initialization.SearchForNonce(
			ctx,
			cfg,
			opts,
			initialization.SearchWithLogger(logger),
		)
		switch {
		case errors.Is(err, context.Canceled):
			log.Println("cli: search for nonce interrupted")
			if label != nil {
				log.Printf("cli: nonce found so far: Nonce: %d | Label: %X\n", nonce, label)
			}
		case err != nil:
			log.Fatalf("cli: search for nonce failed: %v", err)
		default:
			log.Printf("cli: search for nonce completed. Nonce: %d | Label: %X\n", nonce, label)
		}
		return
	}

	processFlags()

	id, err := hex.DecodeString(idHex)
	if err != nil {
		log.Fatalf("failed to decode id %s: %s\n", idHex, err)
	}

	commitmentAtxId, err := hex.DecodeString(commitmentAtxIdHex)
	if err != nil {
		log.Fatalf("failed to decode commitmentAtxId %s: %s\n", commitmentAtxIdHex, err)
	}

	init, err := initialization.NewInitializer(
		initialization.WithConfig(cfg),
		initialization.WithInitOpts(opts),
		initialization.WithNodeId(id),
		initialization.WithCommitmentAtxId(commitmentAtxId),
		initialization.WithLogger(logger),
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

	err = init.Initialize(ctx)
	switch {
	case errors.Is(err, context.Canceled):
		log.Fatalln("cli: initialization interrupted")
		return
	case err != nil:
		log.Println("cli: initialization error", err)
		return
	}

	log.Println("cli: initialization completed")

	if genProof {
		log.Println("cli: generating proof as a sanity test")

		proof, proofMetadata, err := proving.Generate(ctx, shared.ZeroChallenge, cfg, logger, proving.WithDataSource(cfg, id, commitmentAtxId, opts.DataDir))
		if err != nil {
			log.Fatalln("proof generation error", err)
		}
		verifier, err := verifying.NewProofVerifier()
		if err != nil {
			log.Fatalln("failed to create verifier", err)
		}
		defer verifier.Close()
		err = verifier.Verify(proof, proofMetadata, cfg, logger, verifying.WithLabelScryptParams(opts.Scrypt), verifying.AllIndices())
		if err != nil {
			log.Fatalln("failed to verify test proof", err)
		}

		log.Println("cli: proof is valid")
	}
}

func saveKey(key ed25519.PrivateKey) (string, error) {
	err := os.MkdirAll(opts.DataDir, 0o700)
	switch {
	case errors.Is(err, os.ErrExist):
	case err != nil:
		return "", fmt.Errorf("mkdir error: %w", err)
	}

	pub := key.Public().(ed25519.PublicKey)
	filename := filepath.Join(opts.DataDir, fmt.Sprintf("%s.key", hex.EncodeToString(pub[:3])))
	if _, err := os.Stat(filename); err == nil {
		return "", ErrKeyFileExists
	}

	if err := os.WriteFile(filename, []byte(hex.EncodeToString(key)), 0o600); err != nil {
		return "", fmt.Errorf("key write to disk error: %w", err)
	}
	return filename, nil
}

func findKeyFile(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create directory at %s: %w", dataDir, err)
	}
	edKeyFileName := ""
	err := filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory at %s: %w", path, err)
		}

		// skip subdirectories and files in them
		if d.IsDir() && path != dataDir {
			return fs.SkipDir
		}

		// skip files that are not identity files
		if filepath.Ext(path) != ".key" {
			return nil
		}

		if edKeyFileName != "" {
			return fmt.Errorf("multiple identity files found: %w", fs.ErrExist)
		}
		edKeyFileName = d.Name()
		return nil
	})
	return edKeyFileName, err
}

func loadKey(edKeyFileName string) (ed25519.PublicKey, error) {
	if edKeyFileName == "" {
		return nil, fs.ErrNotExist
	}
	keyPath := filepath.Join(opts.DataDir, edKeyFileName)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("could not read private key from %s: %w", keyPath, err)
	}

	dst := make([]byte, ed25519.PrivateKeySize)
	n, err := hex.Decode(dst, data)
	if err != nil {
		log.Fatalf("failed to decode private key from %s: %s\n", keyPath, err)
	}
	if n != ed25519.PrivateKeySize {
		log.Fatalf("size of key (%d) not expected size %d\n", n, ed25519.PrivateKeySize)
	}
	return ed25519.NewKeyFromSeed(dst[:ed25519.SeedSize]).Public().(ed25519.PublicKey), nil
}

func cmdVerifyPos(opts config.InitOpts, fraction float64, logger *zap.Logger) {
	edKeyFileName, err := findKeyFile(opts.DataDir)
	if err != nil {
		log.Fatalf("failed to find identity file: %s\n", err)
	}
	if edKeyFileName != "" {
		log.Println("cli: verifying", edKeyFileName)
		pub, err := loadKey(edKeyFileName)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			log.Fatalf("failed to load public key from %s: %s\n", edKeyFileName, err)
		default:
			metaFile := filepath.Join(opts.DataDir, initialization.MetadataFileName)
			meta, err := initialization.LoadMetadata(opts.DataDir)
			if err != nil {
				log.Fatalf("failed to load metadata from %s: %s\n", opts.DataDir, err)
			}

			if !bytes.Equal(meta.NodeId, pub) {
				log.Fatalf("NodeID in %s (%x) does not match public key from %s (%x)", metaFile, meta.NodeId, edKeyFileName, pub)
			}
			log.Println("cli:", edKeyFileName, "is valid")
		}
	}

	log.Println("cli: verifying POS data")
	params := postrs.NewScryptParams(opts.Scrypt.N, opts.Scrypt.R, opts.Scrypt.P)
	verifyOpts := []postrs.VerifyPosOptionsFunc{
		postrs.WithFraction(fraction),
		postrs.FromFile(uint32(opts.FromFileIdx)),
		postrs.VerifyPosWithLogger(logger),
	}
	if opts.ToFileIdx != nil {
		verifyOpts = append(verifyOpts, postrs.ToFile(uint32(*opts.ToFileIdx)))
	}

	err = postrs.VerifyPos(opts.DataDir, params, verifyOpts...)
	switch {
	case err == nil:
		log.Println("cli: POS data is valid")
	case errors.Is(err, postrs.ErrInvalidPos):
		log.Fatalf("cli: %v\n", err)
	default:
		log.Fatalf("cli: failed (%v)\n", err)
	}
}
