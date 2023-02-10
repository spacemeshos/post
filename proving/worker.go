package proving

import (
	"context"
	"crypto/aes"
)

const (
	blockSize = aes.BlockSize
	m         = blockSize * 8
	d         = 34

	numNonces = 20
)

// ioWorker is a worker that reads labels from disk and writes them to a batch channel to be processed by the
// labelWorkers.
//
// TODO(mafa): use this as base to replace GranSpecificReader / GranSpecificWriter and the persistence package.
func ioWorker(ctx context.Context) error {
	return nil
}

// labelWorker is a worker that receives batches from ioWorker and looks for indices to be included in the proof.
func labelWorker(ctx context.Context) error {
	return nil
}
