package post

import (
	"context"
)

// Data is a PoST data directory. It is used to generate PoST proofs.
// The directory is locked while open. The caller is responsible for closing the Data object.
// Data abstracts the underlying storage and exposes it as single io.Reader.
type Data struct {
}

func (*Data) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (*Data) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, nil
}

func (*Data) Close() error {
	// release file locks
	return nil
}

// Open a directory for use as a PoST data directory.
//
// Basic checks will be performed to ensure the directory is valid:
// - The directory exists.
// - The directory contains a valid PoST metadata file.
// - The directory contains the correct number of PoST data files with the correct size.
//
// If the directory is valid, a Data object will be returned that can be used for proof generation.
// The caller is responsible for closing the Data object, it locks the directory while open.
func Open(path string) (*Data, error) {
	return nil, nil
}

// Init a directory for use as a PoST data directory.
// The directory will be created if it does not exist. While the directory is being initialized, it will be locked.
func Init(ctx context.Context, path string) (*Data, error) {
	return nil, nil
}
