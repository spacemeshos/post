package proving

import (
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"io"
	"runtime"
)

const (
	LabelGroupSize = config.LabelGroupSize
)

type (
	Config           = config.Config
	Proof            = shared.Proof
	Logger           = shared.Logger
	Difficulty       = shared.Difficulty
	Challenge        = shared.Challenge
	MTreeOutput      = shared.MTreeOutput
	MTreeOutputEntry = shared.MTreeOutputEntry
	CacheReader      = cache.CacheReader
	LayerReadWriter  = cache.LayerReadWriter
)

type Prover struct {
	cfg    *Config
	id     []byte
	logger Logger
}

func NewProver(cfg *Config, id []byte) (*Prover, error) {
	if err := shared.ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return &Prover{cfg, id, shared.DisabledLogger{}}, nil
}

func (p *Prover) SetLogger(logger Logger) {
	p.logger = logger
}

func (p *Prover) GenerateProof(challenge Challenge) (proof *Proof, err error) {
	proof, err = p.generateProof(challenge)
	if err != nil {
		err = fmt.Errorf("proof generation failed: %v", err)
		p.logger.Error(err.Error())
	}
	return proof, err
}

func (p *Prover) generateProof(challenge Challenge) (*Proof, error) {
	init, err := initialization.NewInitializer(p.cfg, p.id)
	if err != nil {
		return nil, err
	}
	if err := init.VerifyCompleted(); err != nil {
		return nil, err
	}

	difficulty := Difficulty(p.cfg.Difficulty)
	if err := difficulty.Validate(); err != nil {
		return nil, err
	}

	proof := new(Proof)
	proof.Challenge = challenge
	proof.Identity = p.id

	readers, err := persistence.GetReaders(p.cfg.DataDir, p.id)
	if err != nil {
		return nil, err
	}

	outputs, err := p.generateMTrees(readers, challenge)
	if err != nil {
		return nil, err
	}

	output, err := shared.Merge(outputs)
	if err != nil {
		return nil, err
	}

	proof.MerkleRoot = output.Root

	leafReader := output.Reader.GetLayerReader(0)
	width, err := leafReader.Width()
	if err != nil {
		return nil, err
	}

	provenLeafIndices := shared.CalcProvenLeafIndices(
		proof.MerkleRoot, width<<difficulty, uint8(p.cfg.NumProvenLabels), difficulty)

	_, proof.ProvenLeaves, proof.ProofNodes, err = merkle.GenerateProof(provenLeafIndices, output.Reader)
	if err != nil {
		return nil, err
	}

	err = leafReader.Close()
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func (p *Prover) generateMTree(reader LayerReadWriter, challenge Challenge) (*MTreeOutput, error) {
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(p.cfg.LowestLayerToCacheDuringProofGeneration),
		cache.MakeSliceReadWriterFactory())

	tree, err := merkle.NewTreeBuilder().WithHashFunc(challenge.GetSha256Parent).WithCacheWriter(cacheWriter).Build()
	if err != nil {
		return nil, err
	}

	for {
		leaf, err := reader.ReadNext()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		err = tree.AddLeaf(leaf)
		if err != nil {
			return nil, err
		}
	}

	cacheWriter.SetLayer(0, reader)
	cacheReader, err := cacheWriter.GetReader()
	if err != nil {
		return nil, err
	}

	return &MTreeOutput{
		Reader: cacheReader,
		Root:   tree.Root(),
	}, nil
}

func (p *Prover) generateMTrees(readers []LayerReadWriter, challenge Challenge) ([]*MTreeOutput, error) {
	numFiles := len(readers)
	numWorkers := p.CalcParallelism(numFiles)
	jobsChan := make(chan int, numFiles)
	resultsChan := make(chan *MTreeOutputEntry, numFiles)
	errChan := make(chan error, 0)

	p.logger.Info("execution: start executing %v files, parallelism degree: %v\n", numFiles, numWorkers)

	for i := 0; i < numFiles; i++ {
		jobsChan <- i
	}
	close(jobsChan)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				index, more := <-jobsChan
				if !more {
					return
				}

				output, err := p.generateMTree(readers[index], challenge)
				if err != nil {
					errChan <- err
					return
				}

				resultsChan <- &MTreeOutputEntry{Index: index, MTreeOutput: output}
			}
		}()
	}

	results := make([]*MTreeOutput, numFiles)
	for i := 0; i < numFiles; i++ {
		select {
		case res := <-resultsChan:
			results[res.Index] = res.MTreeOutput
		case err := <-errChan:
			return nil, err
		}
	}

	return results, nil
}

func (p *Prover) CalcParallelism(numFiles int) int {
	max := shared.Max(int(p.cfg.MaxReadFilesParallelism), 1)
	max = shared.Min(max, runtime.NumCPU())
	max = shared.Min(max, numFiles)

	return max
}
