package proving

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"sort"
	"sync"
	"unsafe"

	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

type batch struct {
	Data    []byte
	Index   uint64
	Release func()
}

type solution struct {
	Nonce uint
	Index uint64
}

// ioWorker is a worker that reads labels from disk and writes them to a batch channel to be processed by the
// labelWorkers.
//
// TODO(mafa): use this as base to replace GranSpecificReader / GranSpecificWriter and the persistence package.
func ioWorker(ctx context.Context, batchChan chan<- *batch, b uint32, source io.ReadCloser) error {
	defer close(batchChan)
	defer source.Close()
	index := uint64(0)

	batchDataPool := sync.Pool{
		New: func() any {
			buf := make([]byte, BlocksPerWorker*b)
			return &buf
		},
	}

	for {
		data := *(batchDataPool.Get().(*[]byte))
		n, err := source.Read(data)
		switch {
		case err == io.EOF || err == nil:
			// make sure we don't send partial blocks to the label workers.
			// TODO(mafa): this silently drops partial blocks and should be handled better.
			n -= n % int(b)
			batch := &batch{
				Data:  data[:n],
				Index: index,
				Release: func() {
					batchDataPool.Put(&data)
				},
			}
			select {
			case batchChan <- batch:
				if err == io.EOF {
					return nil
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		default:
			batchDataPool.Put(&data)
			return err
		}
		index += uint64(n) / aes.BlockSize
	}
}

// labelWorker is a worker that receives batches from ioWorker and looks for indices to be included in the proof.
func labelWorker(ctx context.Context, batchChan <-chan *batch, solutionChan chan<- *solution, ch shared.Challenge, numOuts uint8, numNonces uint32, b uint32, d uint, difficulty uint64) error {
	// use two slices with different types that point to the same memory location.
	// this is done to speed up the conversion from bytes to uint64.
	out := make([]byte, numOuts*aes.BlockSize+8)
	u64s := make([][]uint64, 8)
	for i := range u64s {
		size := (len(out) - i) / 8
		u64s[i] = unsafe.Slice((*uint64)(unsafe.Pointer(&out[i])), size)
	}
	mask := (uint64(1) << (d * 8)) - 1

	ciphers, err := createAesCiphers(ch, numOuts)
	if err != nil {
		return fmt.Errorf("failed to create aes ciphers: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch, ok := <-batchChan:
			if !ok {
				return nil
			}
			index := batch.Index
			labels := batch.Data

			block := make([]byte, aes.BlockSize)

			for len(labels) > 0 {
				copy(block, labels[:b])
				labels = labels[b:]

				for i := range ciphers {
					ciphers[i].Encrypt(out[i*aes.BlockSize:(i+1)*aes.BlockSize], block)
				}

				for nonce := uint(0); nonce < uint(numNonces); nonce++ {
					offset := nonce * d
					val := u64s[offset%8][offset/8] & mask // mask out the bits we don't care about

					// check against difficulty threshold
					if val < difficulty {
						s := &solution{
							Index: index,
							Nonce: nonce,
						}
						select {
						case solutionChan <- s: // send solution to proof generator
						case <-ctx.Done():
							batch.Release()
							return ctx.Err()
						}
					}
				}
				index++
			}
			batch.Release()
		}
	}
}

func solutionWorker(ctx context.Context, solutionChan <-chan *solution, numLabels uint64, K2 uint32, logger shared.Logger) (*nonceResult, error) {
	passed := make(map[uint][]uint64)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case solution, ok := <-solutionChan:
			if !ok {
				return nil, fmt.Errorf("no solution found")
			}

			passed[solution.Nonce] = append(passed[solution.Nonce], solution.Index)
			if len(passed[solution.Nonce]) < int(K2) {
				continue
			}

			logger.Debug("found enough label indices for proof with nonce %d", solution.Nonce)
			sort.Slice(passed[solution.Nonce], func(i, j int) bool { return i < j })
			for _, p := range passed[solution.Nonce] {
				logger.Debug("\tlabel index %d", p)
			}
			// logger.Debug("highest index found is %d", passed[solution.Nonce][len(passed[solution.Nonce])-1])

			bitsPerIndex := uint(shared.BinaryRepresentationMinBits(numLabels))
			buf := bytes.NewBuffer(make([]byte, 0, shared.Size(bitsPerIndex, uint(K2))))
			gsWriter := shared.NewGranSpecificWriter(buf, bitsPerIndex)
			for _, p := range passed[solution.Nonce] {
				if err := gsWriter.WriteUintBE(p); err != nil {
					return nil, err
				}
			}

			if err := gsWriter.Flush(); err != nil {
				return nil, err
			}

			return &nonceResult{
				nonce:   uint32(solution.Nonce),
				indices: buf.Bytes(),
			}, nil
		}
	}
}

func createAesCiphers(ch shared.Challenge, count uint8) (ciphers []cipher.Block, err error) {
	for i := uint8(0); i < count; i++ {
		c, err := oracle.CreateBlockCipher(ch, i)
		if err != nil {
			return nil, err
		}
		ciphers = append(ciphers, c)
	}
	return ciphers, nil
}
