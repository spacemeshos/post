package initialization

import (
	"encoding/json"
	"fmt"
	"github.com/spacemeshos/post/shared"
	"io/ioutil"
	"os"
	"path/filepath"
)

const metadataFileName = "postdata_metadata.json"

// metadata is the data associated with the PoST init procedure, persisted in the datadir next to the init files.
type Metadata struct {
	ID            []byte
	BitsPerLabel  uint
	LabelsPerUnit uint
	NumUnits      uint
	NumFiles      uint
}

func SaveMetadata(dir string, v *Metadata) error {
	err := os.MkdirAll(dir, shared.OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %v", err)
	}

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("serialization failure: %v", err)
	}

	err = ioutil.WriteFile(filepath.Join(dir, metadataFileName), data, shared.OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil
}

func LoadMetadata(dir string) (*Metadata, error) {
	filename := filepath.Join(dir, metadataFileName)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrStateMetadataFileMissing
		}
		return nil, fmt.Errorf("read file failure: %v", err)
	}

	metadata := Metadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}
