package initialization

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/natefinch/atomic"

	"github.com/spacemeshos/post/shared"
)

const MetadataFileName = "postdata_metadata.json"

func SaveMetadata(dir string, v *shared.PostMetadata) error {
	err := os.MkdirAll(dir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %w", err)
	}

	filename := filepath.Join(dir, MetadataFileName)

	tmp, err := os.Create(fmt.Sprintf("%s.tmp", filename))
	if err != nil {
		return fmt.Errorf("create temporary file %s: %w", tmp.Name(), err)
	}
	defer tmp.Close()

	if err := json.NewEncoder(tmp).Encode(v); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close tmp file %s: %w", tmp.Name(), err)
	}

	if err := atomic.ReplaceFile(tmp.Name(), filename); err != nil {
		return fmt.Errorf("save file from %s, %s: %w", tmp.Name(), filename, err)
	}

	return nil
}

func LoadMetadata(dir string) (*shared.PostMetadata, error) {
	filename := filepath.Join(dir, MetadataFileName)
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrStateMetadataFileMissing
		}
		return nil, fmt.Errorf("read file failure: %w", err)
	}

	metadata := shared.PostMetadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}
