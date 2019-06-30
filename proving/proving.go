package proving

import (
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"io"
	"runtime"
)

const (
	LabelGroupSize = shared.LabelGroupSize
)

var (
	VerifyInitialized = shared.VerifyInitialized
)

type (
	Config           = shared.Config
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
	logger Logger
}

func NewProver(cfg *Config, logger Logger) *Prover { return &Prover{cfg, logger} }

func (p *Prover) GenerateProof(id []byte, challenge Challenge) (proof *Proof,
	err error) {
	proof, err = p.generateProof(id, challenge)
	if err != nil {
		err = fmt.Errorf("proof generation failed: %v", err)
		p.logger.Error(err.Error())
	}
	return proof, err
}

func (p *Prover) generateProof(id []byte, challenge Challenge) (*Proof, error) {
	if err := VerifyInitialized(p.cfg, id); err != nil {
		return nil, err
	}

	difficulty := Difficulty(p.cfg.Difficulty)
	if err := difficulty.Validate(); err != nil {
		return nil, err
	}

	proof := new(Proof)
	proof.Challenge = challenge
	proof.Identity = id

	dir := shared.GetInitDir(p.cfg.DataDir, id)
	readers, err := persistence.GetReaders(dir)
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

	provenLeafIndices := CalcProvenLeafIndices(
		proof.MerkleRoot, width<<difficulty, uint8(p.cfg.NumOfProvenLabels), difficulty)

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

	return &MTreeOutput{
		Reader: cacheReader,
		Root:   tree.Root(),
	}, nil
}

func (p *Prover) generateMTrees(readers []LayerReadWriter, challenge Challenge) ([]*MTreeOutput, error) {
	numOfFiles := len(readers)
	numOfWorkers := p.CalcParallelism(numOfFiles)
	jobsChan := make(chan int, numOfFiles)
	resultsChan := make(chan *MTreeOutputEntry, numOfFiles)
	errChan := make(chan error, 0)

	p.logger.Info("execution: start executing %v files, parallelism degree: %v\n", numOfFiles, numOfWorkers)

	for i := 0; i < numOfFiles; i++ {
		jobsChan <- i
	}
	close(jobsChan)

	for i := 0; i < numOfWorkers; i++ {
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

	results := make([]*MTreeOutput, numOfFiles)
	for i := 0; i < numOfFiles; i++ {
		select {
		case res := <-resultsChan:
			results[res.Index] = res.MTreeOutput
		case err := <-errChan:
			return nil, err
		}
	}

	return results, nil
}

func (p *Prover) CalcParallelism(numOfFiles int) int {
	max := shared.Max(int(p.cfg.MaxReadFilesParallelism), 1)
	max = shared.Min(max, runtime.NumCPU())
	max = shared.Min(max, numOfFiles)

	return max
}
