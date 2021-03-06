package proving

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
	"io"
	"runtime"
)

const (
	MaxIterations          = 10
	NumNoncesPerIterations = 10
)

type (
	Config    = config.Config
	Proof     = shared.Proof
	Logger    = shared.Logger
	Challenge = shared.Challenge
)

var (
	FastOracle = oracle.FastOracle
)

type Prover struct {
	cfg    *Config
	id     []byte
	logger Logger
}

func NewProver(cfg *Config, id []byte) (*Prover, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Prover{cfg, id, shared.DisabledLogger{}}, nil
}

func (p *Prover) SetLogger(logger Logger) {
	p.logger = logger
}

func (p *Prover) GenerateProof(challenge Challenge) (proof *Proof, err error) {
	if err := p.ValidateProofGeneration(); err != nil {
		return nil, err
	}

	return p.generateProof(challenge)
}

func (p *Prover) ValidateProofGeneration() error {
	init, err := initialization.NewInitializer(p.cfg, p.id)
	if err != nil {
		return err
	}
	if err := init.VerifyCompleted(); err != nil {
		return err
	}

	return nil
}

func (p *Prover) tryNonce(ctx context.Context, ch Challenge, nonce uint32, readerChan <-chan []byte, difficulty uint64) ([]byte, error) {
	var indices = bytes.NewBuffer(make([]byte, p.cfg.K2*8)[0:0])
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
			hashNum := binary.LittleEndian.Uint64(hash[:])

			// check the difficulty requirement.
			if hashNum <= difficulty {
				indexBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(indexBytes, index)
				indices.Write(indexBytes)
				passed++

				if passed >= p.cfg.K2 {
					return indices.Bytes(), nil
				}
			}

			index++
		}
	}

	panic("unreachable")
}

type nonceResult struct {
	nonce   uint32
	indices []byte
	err     error
}

func (p *Prover) generateProof(challenge Challenge) (*Proof, error) {
	for i := 0; i < MaxIterations; i++ {
		startNonce := uint32(i) * NumNoncesPerIterations
		endNonce := startNonce + NumNoncesPerIterations - 1

		p.logger.Info("proving: starting iteration %d; startNonce: %v, endNonce: %v, challenge: %x", i+1, startNonce, endNonce, challenge)

		goodNonceResult, err := p.tryNonces(challenge, startNonce, endNonce)
		if err != nil {
			return nil, err
		}

		if goodNonceResult != nil {
			p.logger.Info("proving: generated proof after %d iteration(s)", i+1)

			return &Proof{
				Challenge: challenge,
				Nonce:     goodNonceResult.nonce,
				Indices:   goodNonceResult.indices,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to generate proof; tried %v iterations, %v nonces each", MaxIterations, NumNoncesPerIterations)
}

func (p *Prover) tryNonces(challenge Challenge, startNonce, endNonce uint32) (*nonceResult, error) {
	difficulty := p.cfg.ProvingDifficulty()
	//
	readers, err := persistence.GetReaders(p.cfg.DataDir, p.id, p.cfg.LabelSize)
	if err != nil {
		return nil, err
	}

	reader, err := persistence.Merge(readers)
	if err != nil {
		return nil, err
	}

	numWorkers := endNonce - startNonce + 1
	var indices []byte

	workersChans := make([]chan []byte, numWorkers)
	for i := range workersChans {
		workersChans[i] = make(chan []byte, 1000) // TODO(moshababo): use numLabels/2 size instead? need just enough buffer to circulate between the two routines
	}
	resultsChan := make(chan nonceResult, numWorkers)
	errChan := make(chan error, numWorkers)

	// Start IO worker.
	// Feed all labels into each worker chan.
	go func() {
		for {
			label, err := reader.ReadNext()
			if err != nil {
				for i := range workersChans {
					close(workersChans[i])
				}

				if err != io.EOF {
					errChan <- err
				}
				break
			}

			for i := range workersChans {
				workersChans[i] <- label
			}
		}
	}()

	// Start a worker for each nonce.
	ctx, cancel := context.WithCancel(context.Background())
	for i := uint32(0); i < numWorkers; i++ {
		i := i
		go func() {
			nonce := startNonce + i
			indices, err = p.tryNonce(ctx, challenge, nonce, workersChans[i], difficulty)
			resultsChan <- nonceResult{nonce, indices, err}
		}()
	}

	// Drain the workers results chan.
	var goodNonce *nonceResult
	for i := uint32(0); i < numWorkers; i++ {
		res := <-resultsChan
		if res.err != nil {
			p.logger.Info("proving: nonce %v failed: %v", res.nonce, res.err)
		} else {
			p.logger.Info("proving: nonce %v succeeded", res.nonce)
			cancel()

			// There might be multiple successful nonces due to race condition with the cancellation,
			// but this is not a problem. We'll use the last one to arrive.
			goodNonce = &res
		}
	}

	// Check for an error from the IO worker.
	select {
	case err := <-errChan:
		p.logger.Info("proving: error: %v", err)
		return nil, err
	default:
	}

	return goodNonce, nil
}

func (p *Prover) CalcParallelism(numFiles int) int {
	max := shared.Max(int(p.cfg.MaxReadFilesParallelism), 1)
	max = shared.Min(max, runtime.NumCPU())
	max = shared.Min(max, numFiles)

	return max
}
