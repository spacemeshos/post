package main

import (
	"code.cloudfoundry.org/bytefmt"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/validation"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Config = shared.Config

var (
	id, _         = hex.DecodeString("deadbeef")
	challenge, _  = hex.DecodeString("this is a challenge")
	defaultConfig = shared.DefaultConfig()
)

type testMode int

const (
	single testMode = iota
	mid
	full
)

func main() {
	datadir := flag.String("datadir", defaultConfig.DataDir, "filesystem datadir path")
	space := flag.Uint64("space", 1<<23, "space per unit, in bytes")
	single := flag.Bool("single", false, "whether to execute a single test instead of the complete set")
	flag.Parse()

	log.Printf("bench config: datadir: %v, space: %v", *datadir, *space)

	cases := genTestCases(*datadir, *space, *single)
	data := make([][]string, 0)
	for i, cfg := range cases {

		log.Printf("test %v/%v starting...", i+1, len(cases))
		tStart := time.Now()

		init := initialization.NewInitializer(&cfg, shared.DisabledLogger{})
		prover := proving.NewProver(&cfg, shared.DisabledLogger{})
		validator := validation.NewValidator(&cfg)

		t := time.Now()
		proof, err := init.Initialize(id)
		if err != nil {
			panic(err)
		}
		eInit := time.Since(t)

		t = time.Now()
		err = validator.Validate(proof)
		if err != nil {
			panic(err)
		}
		eInitV := time.Since(t)

		t = time.Now()
		proof, err = prover.GenerateProof(id, challenge)
		if err != nil {
			panic(err)
		}
		eExec := time.Since(t)

		t = time.Now()
		err = validator.Validate(proof)
		if err != nil {
			panic(err)
		}
		eExecV := time.Since(t)

		err = init.Reset(id)
		if err != nil {
			panic(err)
		}

		log.Printf("test %v/%v completed, %v", i+1, len(cases), time.Since(tStart))

		pfiles, pinfile := init.CalcParallelism(runtime.NumCPU())
		data = append(data, []string{
			bytefmt.ByteSize(cfg.SpacePerUnit),
			bytefmt.ByteSize(cfg.FileSize),
			strconv.Itoa(pfiles),
			strconv.Itoa(pinfile),
			eInit.Round(time.Duration(time.Millisecond)).String(),
			eInitV.Round(time.Duration(time.Millisecond)).String(),
			eExec.Round(time.Duration(time.Millisecond)).String(),
			eExecV.Round(time.Duration(time.Millisecond)).String(),
		})
	}

	header := []string{"space", "filesize", "p-files", "p-infile", "init", "init-v", "exec", "exec-v"}
	report(*datadir, header, data)
}

func report(datadir string, header []string, data [][]string) {
	fmt.Printf("\n\nBENCHMARKS: datadir=%v\n", datadir)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorder(true)
	table.AppendBulk(data)
	table.Render()
}

func genTestCases(datadir string, space uint64, single bool) []Config {
	def := *defaultConfig
	cases := make([]Config, 0)

	def.DataDir = datadir
	def.SpacePerUnit = space
	def.FileSize = space

	if single {
		cases = append(cases, def)
		return cases
	}

	def.MaxFilesParallelism = 1
	def.MaxInFileParallelism = 1

	// Various in-file parallelism degrees.
	for i := 1; i <= runtime.NumCPU(); i++ {
		cfg := def
		cfg.MaxInFileParallelism = uint(i)
		cases = append(cases, cfg)
	}

	// Split to files without files parallelism.
	for i := 1; i <= 6; i++ {
		cfg := def
		cfg.FileSize >>= uint(i)
		cases = append(cases, cfg)
	}

	// Split to files with max files parallelism degrees.
	for i := uint(1); i <= 6; i++ {
		cfg := def
		cfg.FileSize >>= i
		cfg.MaxFilesParallelism = uint(math.Pow(2, float64(i)))
		cases = append(cases, cfg)
	}

	// Split to files with max files and in-file parallelism degrees.
	for i := uint(1); i <= 4; i++ {
		cfg := def
		cfg.FileSize >>= i
		cfg.MaxFilesParallelism = uint(math.Pow(2, float64(i)))
		cfg.MaxInFileParallelism = uint(math.Pow(2, float64(i)))
		cases = append(cases, cfg)
	}

	return cases
}
