package proving

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/spacemeshos/go-spacemesh/log"
)

const (
	blockSize       = aes.BlockSize
	blocksPerWorker = 2 << 20
	batchSize       = blocksPerWorker * blockSize

	m = blockSize * 8
	d = 34

	numNonces = 20
)

var batchDataPool = sync.Pool{
	New: func() any { return make([]byte, batchSize) },
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
func ioWorker(ctx context.Context, batchQueue chan<- *batch, datadir string) error {
	defer close(batchQueue)
	index := uint64(0)

	for {
		data := batchDataPool.Get().([]byte)
		n, err := reader.Read(data)
		switch {
		case err == io.EOF || err == nil:
			b := &batch{
				Data:  data[:n],
				Index: index,
			}
			select {
			case batchQueue <- b:
			case <-ctx.Done():
				return ctx.Err()
			}
		default:
			batchDataPool.Put(data)
			log.Error("error reading labels from disk: %s", err)
			return err
		}
		index += uint64(n)
	}

	return nil
}

// labelWorker is a worker that receives batches from ioWorker and looks for indices to be included in the proof.
func labelWorker(ctx context.Context, batchQueue <-chan *batch, proofChan chan<- *solution, ch Challenge, difficulty []byte) error {
	numOuts := uint8(math.Ceil(float64(numNonces*d) / m))
	difficultyVal := le34(difficulty, 0)

	ciphers, err := createAesCiphers(ch, numOuts)
	if err != nil {
		return fmt.Errorf("failed to create aes ciphers: %v", err)
	}
	out := make([]byte, numOuts*blockSize)

	for batch := range batchQueue {
		index := batch.Index
		labels := batch.Data

		for len(labels) > 0 {
			block := labels[:aes.BlockSize]
			labels = labels[aes.BlockSize:]

			for i := uint8(0); i < numOuts; i++ {
				ciphers[i].Encrypt(out[i*blockSize:(i+1)*blockSize], block)
			}

			for j := uint(0); j < numNonces; j++ {
				val := le34Faster(out, j*d)
				if val < difficultyVal {
					select {
					case <-ctx.Done():
						batchDataPool.Put(batch.Data[:batchSize])
						return ctx.Err()
					case proofChan <- &solution{
						Index: index,
						Nonce: j,
					}:
					}
				}
			}
			index++
		}
		batchDataPool.Put(batch.Data[:batchSize])
	}

	return nil
}

// Create a set of AES block ciphers.
// A cipher is created using an idx encrypted with challenge:
// cipher[i] = AES(ch).Encrypt(i).
func createAesCiphers(ch Challenge, count uint8) (ciphers []cipher.Block, err error) {
	// a temporary cipher used only to create keys.
	keyCipher, err := aes.NewCipher(ch)
	if err != nil {
		return nil, err
	}

	keyBuffer := make([]byte, aes.BlockSize)
	key := make([]byte, aes.BlockSize)

	for i := byte(0); i < count; i++ {
		keyBuffer[0] = i
		keyCipher.Encrypt(key, keyBuffer)
		c, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		ciphers = append(ciphers, c)
	}
	return ciphers, nil
}

// Get an uint64 that consists of 34 bits from the data slice starting from bit i.
func le34(data []byte, i uint) uint64 {
	b := data[i/8 : (i/8)+5]
	x := binary.LittleEndian.Uint32(b)
	// Combine the two values into an uint64
	z := uint64(x) | uint64(b[4])<<32
	// Shift the result to the right by the remaining bits
	z = z >> (i % 8)
	// Return the 34 bits from the data slice
	return z & 0x3FFFFFFFFFFFF
}

// Get an uint64 that consists of 34 bits from the data slice starting from bit i.
// SAFETY: Assumes len(data) >= (i/8)+8.
func le34Faster(data []byte, i uint) uint64 {
	b := data[i/8 : (i/8)+8]
	z := binary.LittleEndian.Uint64(b)
	// Shift the result to the right by the remaining bits
	z = z >> (i % 8)
	// Return the 34 bits from the data slice
	return z & 0x3FFFFFFFFFFFF
}
