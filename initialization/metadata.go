package initialization

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spacemeshos/post/shared"
)

const MetadataFileName = "postdata_metadata.json"

func getEffectiveDir(dir string) string {
	envDir := os.Getenv("POST_METADATA_DIR")
	fmt.Println("Got postmeta dir of env:", envDir)
	if envDir != "" {
		return envDir
	}
	return dir
}

func SaveMetadata(dir string, v *shared.PostMetadata) error {
	dir = getEffectiveDir(dir)

	err := os.MkdirAll(dir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %w", err)
	}

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("serialization failure: %w", err)
	}

	filePath := filepath.Join(dir, MetadataFileName)
	fmt.Println("Writing metadata to:", filePath)

	err = os.WriteFile(filepath.Join(dir, MetadataFileName), data, shared.OwnerReadWrite)

	if err != nil {
		return fmt.Errorf("write to disk failure: %w", err)
	}

	return nil
}

func LoadMetadata(dir string) (*shared.PostMetadata, error) {
	dir = getEffectiveDir(dir)

	filename := filepath.Join(dir, MetadataFileName)
	fmt.Println("Reading metadata from:", filename)
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
