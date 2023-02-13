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

var batchDataPool = sync.Pool{
	New: func() any {
		buf := make([]byte, batchSize)
		return &buf
	},
}

type batch struct {
	Data  []byte
	Index uint64
}

type solution struct {
	Nonce uint
	Index uint64
}

// ioWorker is a worker that reads labels from disk and writes them to a batch channel to be processed by the
// labelWorkers.
//
// TODO(mafa): use this as base to replace GranSpecificReader / GranSpecificWriter and the persistence package.
func ioWorker(ctx context.Context, batchQueue chan<- *batch, reader io.Reader) error {
	defer close(batchQueue)
	index := uint64(0)

	for {
		data := *batchDataPool.Get().(*[]byte)
		n, err := reader.Read(data)
		switch {
		case err == io.EOF || err == nil:
			n -= n % aes.BlockSize // make sure we don't send partial blocks to the label workers.
			b := &batch{
				Data:  data[:n],
				Index: index,
			}
			select {
			case batchQueue <- b:
				if err == io.EOF {
					return nil
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		default:
			data = data[:n]
			batchDataPool.Put(&data)
			return err
		}
		index += uint64(n)
	}
}

// labelWorker is a worker that receives batches from ioWorker and looks for indices to be included in the proof.
func labelWorker(ctx context.Context, batchChan <-chan *batch, solutionChan chan<- *solution, ch Challenge, numOuts uint8, d uint, difficulty uint64) error {
	// use two slices with different types that point to the same memory location.
	// this is done to speed up the conversation from bytes to uint64.
	u64s := make([]uint64, numOuts*aes.BlockSize/8)
	out := unsafe.Slice((*byte)(unsafe.Pointer(&u64s[0])), len(u64s)*8)
	mask := ((uint64(1) << d) - 1)

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

			for len(labels) > 0 {
				block := labels[:aes.BlockSize]
				labels = labels[aes.BlockSize:]

				select {
				case <-ctx.Done():
					batchDataPool.Put(&batch.Data)
					return ctx.Err()
				default:
				}

				for i, cipher := range ciphers {
					cipher.Encrypt(out[i*aes.BlockSize:(i+1)*aes.BlockSize], block)
				}

				for nonce := uint(0); nonce < uint(numOuts); nonce++ {
					// Extract the hash output for this nonce
					offset := nonce * d
					low_idx := offset / 64
					high_idx := (offset + d - 1) / 64

					val := u64s[low_idx]
					val >>= offset % 64
					if low_idx != high_idx {
						high := u64s[high_idx]
						val |= (high << (64 - offset%64))
					}

					val &= mask

					// check against difficulty threshold
					if val < difficulty {
						s := &solution{
							Index: index,
							Nonce: nonce,
						}
						select {
						case solutionChan <- s: // send solution to proof generator
						case <-ctx.Done():
							batchDataPool.Put(&batch.Data)
							return ctx.Err()
						}
					}
				}
				index++
			}
			batchDataPool.Put(&batch.Data)
		}
	}
}

func solutionWorker(ctx context.Context, solutionChan <-chan *solution, numLabels uint64, K2 uint32, logger Logger) (*nonceResult, error) {
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

			logger.Debug("Found enough label indices for proof with nonce %d", solution.Nonce)
			sort.Slice(passed[solution.Nonce], func(i, j int) bool { return i < j })
			logger.Debug("indices are %v", passed[solution.Nonce])

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

func createAesCiphers(ch Challenge, count uint8) (ciphers []cipher.Block, err error) {
	for i := uint8(0); i < count; i++ {
		c, err := oracle.CreateBlockCipher(ch, i)
		if err != nil {
			return nil, err
		}
		ciphers = append(ciphers, c)
	}
	return ciphers, nil
}
