package proving

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
)

const (
	NumNoncesPerIteration = 10 // TODO(moshababo): update the recommended value
	MaxNumIterations      = 10 // TODO(moshababo): update the recommended value
)

type (
	Config              = config.Config
	Proof               = shared.Proof
	ProofMetadata       = shared.ProofMetadata
	Logger              = shared.Logger
	Challenge           = shared.Challenge
	ConfigMismatchError = shared.ConfigMismatchError

	Metadata  = initialization.Metadata
	DiskState = initialization.DiskState
)

var (
	FastOracle = oracle.FastOracle
	UInt64LE   = shared.UInt64LE
)

type Prover struct {
	cfg     Config
	datadir string
	id      []byte

	diskState *DiskState

	logger Logger
}

func NewProver(cfg Config, datadir string, id []byte) (*Prover, error) {
	return &Prover{
		cfg:       cfg,
		datadir:   datadir,
		id:        id,
		diskState: initialization.NewDiskState(datadir, cfg.BitsPerLabel),
		logger:    shared.DisabledLogger{},
	}, nil
}

// GenerateProof (analogous to the PoST protocol Execution phase) receives a challenge that cannot be predicted,
// and reads the entire PoST data to generate a proof in response to the challenge to prove that the prover data exists at the time of invocation.
// Generating a proof can be repeated arbitrarily many times without repeating the PoST protocol Initialization phase;
// thus despite the initialization essentially serving as a PoW, the amortized computational complexity can be made arbitrarily small.
func (p *Prover) GenerateProof(challenge Challenge) (*Proof, *ProofMetadata, error) {
	m, err := p.loadMetadata()
	if err != nil {
		return nil, nil, err
	}

	if err := p.verifyGenerateProofAllowed(m); err != nil {
		return nil, nil, err
	}

	numLabels := uint64(m.NumUnits * p.cfg.LabelsPerUnit)

	for i := 0; i < MaxNumIterations; i++ {
		startNonce := uint32(i) * NumNoncesPerIteration
		endNonce := startNonce + NumNoncesPerIteration - 1

		p.logger.Debug("proving: starting iteration %d; startNonce: %v, endNonce: %v, challenge: %x", i+1, startNonce, endNonce, challenge)

		solutionNonceResult, err := p.tryNonces(numLabels, challenge, startNonce, endNonce)
		if err != nil {
			return nil, nil, err
		}

		if solutionNonceResult != nil {
			p.logger.Info("proving: generated proof after %d iteration(s)", i+1)

			proof := &Proof{
				Nonce:   solutionNonceResult.nonce,
				Indices: solutionNonceResult.indices,
			}
			proofMetadata := &ProofMetadata{
				ID:            p.id,
				Challenge:     challenge,
				BitsPerLabel:  p.cfg.BitsPerLabel,
				LabelsPerUnit: p.cfg.LabelsPerUnit,
				NumUnits:      m.NumUnits,
				K1:            p.cfg.K1,
				K2:            p.cfg.K2,
			}
			return proof, proofMetadata, nil
		}
	}

	return nil, nil, fmt.Errorf("failed to generate proof; tried %v iterations, %v nonces each", MaxNumIterations, NumNoncesPerIteration)
}

func (p *Prover) SetLogger(logger Logger) {
	p.logger = logger
}

func (p *Prover) verifyGenerateProofAllowed(m *Metadata) error {
	if err := p.verifyMetadata(m); err != nil {
		return err
	}

	if err := p.verifyInitCompleted(m.NumUnits); err != nil {
		return err
	}

	return nil
}

func (p *Prover) verifyInitCompleted(numUnits uint) error {
	ok, err := p.initCompleted(numUnits)
	if err != nil {
		return err
	}
	if ok == false {
		return shared.ErrInitNotCompleted
	}

	return nil
}

func (p *Prover) initCompleted(numUnits uint) (bool, error) {
	numLabelsWritten, err := p.diskState.NumLabelsWritten()
	if err != nil {
		return false, err
	}

	target := uint64(numUnits) * uint64(p.cfg.LabelsPerUnit)
	return numLabelsWritten == target, nil
}

func (p *Prover) loadMetadata() (*initialization.Metadata, error) {
	return initialization.LoadMetadata(p.datadir)
}

func (p *Prover) verifyMetadata(m *Metadata) error {
	if bytes.Compare(p.id, m.ID) != 0 {
		return ConfigMismatchError{
			Param:    "ID",
			Expected: fmt.Sprintf("%x", p.id),
			Found:    fmt.Sprintf("%x", m.ID),
			DataDir:  p.datadir,
		}
	}

	if p.cfg.BitsPerLabel != m.BitsPerLabel {
		return ConfigMismatchError{
			Param:    "BitsPerLabel",
			Expected: fmt.Sprintf("%d", p.cfg.BitsPerLabel),
			Found:    fmt.Sprintf("%d", m.BitsPerLabel),
			DataDir:  p.datadir,
		}
	}

	if p.cfg.LabelsPerUnit != m.LabelsPerUnit {
		return ConfigMismatchError{
			Param:    "LabelsPerUnit",
			Expected: fmt.Sprintf("%d", p.cfg.LabelsPerUnit),
			Found:    fmt.Sprintf("%d", m.LabelsPerUnit),
			DataDir:  p.datadir,
		}
	}

	return nil
}

func (p *Prover) tryNonce(ctx context.Context, numLabels uint64, ch Challenge, nonce uint32, readerChan <-chan []byte, difficulty uint64) ([]byte, error) {
	var bitsPerIndex = uint(shared.BinaryRepresentationMinBits(numLabels))
	var buf = bytes.NewBuffer(make([]byte, shared.Size(bitsPerIndex, p.cfg.K2))[0:0])
	var gsWriter = shared.NewGranSpecificWriter(buf, bitsPerIndex)
	var index uint64
	var passed uint
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("cancelled: tried: %v, passed: %v, needed: %v", index, passed, p.cfg.K2)
		case label, more := <-readerChan:
			if !more {
				return nil, fmt.Errorf("exhausted all labels; tried: %v, passed: %v, needed: %v", index, passed, p.cfg.K2)
			}

			hash := FastOracle(ch, nonce, label)

			// Convert the fast oracle output's leading 64 bits to a number,
			// so that it could be used to perform math comparisons.
			hashNum := UInt64LE(hash[:])

			// Check the difficulty requirement.
			if hashNum <= difficulty {
				if err := gsWriter.WriteUintBE(index); err != nil {
					return nil, err
				}
				passed++

				if passed >= p.cfg.K2 {
					if err := gsWriter.Flush(); err != nil {
						return nil, err
					}
					return buf.Bytes(), nil
				}
			}

			index++
		}
	}
}

type nonceResult struct {
	nonce   uint32
	indices []byte
	err     error
}

func (p *Prover) tryNonces(numLabels uint64, challenge Challenge, startNonce, endNonce uint32) (*nonceResult, error) {
	var difficulty = shared.ProvingDifficulty(numLabels, uint64(p.cfg.K1))

	reader, err := persistence.NewLabelsReader(p.datadir, p.cfg.BitsPerLabel)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	gsReader := shared.NewGranSpecificReader(reader, p.cfg.BitsPerLabel)

	numWorkers := endNonce - startNonce + 1
	workersChans := make([]chan []byte, numWorkers)
	// workersComplete channel will be closed when worker stops listening for appropriate workersChan
	workersComplete := make([]chan struct{}, numWorkers)
	for i := range workersChans {
		workersChans[i] = make(chan []byte, 1)
		workersComplete[i] = make(chan struct{})
	}
	resultsChan := make(chan *nonceResult, numWorkers)
	errChan := make(chan error, 1)

	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start IO worker.
	// Feed all labels into each worker chan.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			label, err := gsReader.ReadNext()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errChan <- err
				}
				for i := range workersChans {
					close(workersChans[i])
				}
				return
			}

			for i := range workersChans {
				select {
				case workersChans[i] <- label:
				case <-workersComplete[i]:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Start a worker for each nonce.
	// TODO(dshulyak) it would be more efficient to start a worker per CPU and distribute work among
	// them but it is not trivial
	for i := uint32(0); i < numWorkers; i++ {
		i := i
		wg.Add(1)
		go func() {
			nonce := startNonce + i
			indices, err := p.tryNonce(ctx, numLabels, challenge, nonce, workersChans[i], difficulty)
			close(workersComplete[i])
			resultsChan <- &nonceResult{nonce, indices, err}
			wg.Done()
		}()
	}

	// return last observed error if all workers failed, otherwise return first found result
	for i := uint32(0); i < numWorkers; i++ {
		select {
		case result := <-resultsChan:
			if result.err != nil {
				p.logger.Debug("proving: nonce %v failed: %v", result.nonce, result.err)
			} else {
				p.logger.Debug("proving: nonce %v succeeded", result.nonce)
				return result, nil
			}
		case err := <-errChan:
			p.logger.Debug("proving: error: %v", err)
			return nil, err
		}
	}
	return nil, nil
}
