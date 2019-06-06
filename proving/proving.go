package proving

import (
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"io"
	"math"
)

const (
	LabelGroupSize = shared.LabelGroupSize
)

var (
	VerifyInitialized = shared.VerifyInitialized
)

type (
	Config     = shared.Config
	Logger     = shared.Logger
	Difficulty = shared.Difficulty
	Challenge  = shared.Challenge
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
	reader, err := persistence.NewLabelsReader(dir)
	if err != nil {
		return nil, err
	}
	width, err := reader.Width()
	if err != nil {
		return nil, err
	}
	if width*difficulty.LabelsPerGroup() >= math.MaxUint64 {
		return nil, fmt.Errorf("leaf reader too big, number of label groups (%d) * labels per group (%d) "+
			"overflows uint64", width, difficulty.LabelsPerGroup())
	}
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
	proof.MerkleRoot = tree.Root()

	cacheWriter.SetLayer(0, reader)
	cacheReader, err := cacheWriter.GetReader()

	numOfLabels := width * difficulty.LabelsPerGroup()
	provenLeafIndices := CalcProvenLeafIndices(
		proof.MerkleRoot, numOfLabels, uint8(p.cfg.NumOfProvenLabels), difficulty)

	_, proof.ProvenLeaves, proof.ProofNodes, err = merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return nil, err
	}

	return proof, nil
}
