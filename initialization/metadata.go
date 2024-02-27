package initialization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/natefinch/atomic"

	"github.com/spacemeshos/post/shared"
)

const MetadataFileName = "postdata_metadata.json"

func SaveMetadata(dir string, v *shared.PostMetadata) error {
	err := os.MkdirAll(dir, shared.OwnerReadWriteExec)
	switch {
	case errors.Is(err, fs.ErrExist):
	case err != nil:
		return fmt.Errorf("dir creation failure: %w", err)
	}

	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	if err := atomic.WriteFile(filepath.Join(dir, MetadataFileName), bytes.NewBuffer(data)); err != nil {
		return fmt.Errorf("write to disk failure: %w", err)
	}

	return nil
}

func LoadMetadata(dir string) (*shared.PostMetadata, error) {
	filename := filepath.Join(dir, MetadataFileName)
	data, err := os.ReadFile(filename)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, ErrStateMetadataFileMissing
	case err != nil:
		return nil, fmt.Errorf("read file failure: %w", err)
	}

	metadata := shared.PostMetadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}
