package initialization

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/natefinch/atomic"
	"go.uber.org/zap"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/shared"
)

var migrateData map[int]func(dir string, logger *zap.Logger) (err error)

func init() {
	migrateData = make(map[int]func(dir string, logger *zap.Logger) (err error))
	migrateData[0] = migrateV0
}

type MetadataVersion struct {
	Version int `json:",omitempty"`
}

// MigratePoST migrates the PoST metadata file to the latest version.
func MigratePoST(dir string, logger *zap.Logger) (err error) {
	logger.Info("checking PoST for migrations")

	filename := filepath.Join(dir, MetadataFileName)
	file, err := os.Open(filename)
	switch {
	case os.IsNotExist(err):
		return ErrStateMetadataFileMissing
	case err != nil:
		return fmt.Errorf("could not open metadata file: %w", err)
	}
	defer file.Close()

	version := MetadataVersion{}
	if err := json.NewDecoder(file).Decode(&version); err != nil {
		return fmt.Errorf("failed to determine metadata version: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close metadata file: %w", err)
	}

	if version.Version == len(migrateData) {
		logger.Info("PoST is up to date, no migration needed")
		return nil
	}

	if version.Version > len(migrateData) {
		return fmt.Errorf("PoST metadata version %d is newer than the latest supported version %d", version.Version, len(migrateData))
	}

	logger.Info("determined PoST version", zap.Int("version", version.Version))

	for v := version.Version; v < len(migrateData); v++ {
		if err := migrateData[v](dir, logger); err != nil {
			return fmt.Errorf("failed to migrate metadata from version %d to version %d: %w", v, v+1, err)
		}

		logger.Info("migrated PoST successfully to version", zap.Int("version", v+1))
	}

	logger.Info("PoST migration process finished successfully")
	return nil
}

type postMetadataV0 struct {
	NodeId          []byte
	CommitmentAtxId []byte

	LabelsPerUnit uint64
	NumUnits      uint32
	MaxFileSize   uint64
	Nonce         *uint64           `json:",omitempty"`
	NonceValue    shared.NonceValue `json:",omitempty"`
	LastPosition  *uint64           `json:",omitempty"`
}

// migrateV0 upgrades PoST from version 0 to version 1.
//
// - add version field to postdata_metadata.json (missing in version 0)
// - add NonceValue field to postdata_metadata.json if missing (was introduced before migrations, not every PoST version 0 metadata file has it)
// - re-encode NodeId and CommitmentAtxId as hex strings.
// - add Scrypt field to postdata_metadata.json (missing in version 0), assume default mainnet values.
func migrateV0(dir string, logger *zap.Logger) (err error) {
	filename := filepath.Join(dir, MetadataFileName)
	file, err := os.Open(filename)
	switch {
	case os.IsNotExist(err):
		return ErrStateMetadataFileMissing
	case err != nil:
		return fmt.Errorf("could not read metadata file: %w", err)
	}
	defer file.Close()

	old := postMetadataV0{}
	if err := json.NewDecoder(file).Decode(&old); err != nil {
		return fmt.Errorf("failed to determine metadata version: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close metadata file: %w", err)
	}

	if len(old.NodeId) != 32 {
		return fmt.Errorf("invalid node ID length: %d", len(old.NodeId))
	}

	if len(old.CommitmentAtxId) != 32 {
		return fmt.Errorf("invalid commitment ATX ID length: %d", len(old.CommitmentAtxId))
	}

	new := shared.PostMetadata{
		Version: 1,

		NodeId:          old.NodeId,
		CommitmentAtxId: old.CommitmentAtxId,

		LabelsPerUnit: old.LabelsPerUnit,
		NumUnits:      old.NumUnits,
		MaxFileSize:   old.MaxFileSize,
		Scrypt:        config.DefaultLabelParams(), // we don't know the scrypt params, but on mainnet they are the default ones

		Nonce:        old.Nonce,
		NonceValue:   old.NonceValue,
		LastPosition: old.LastPosition,
	}

	if new.Nonce != nil && new.NonceValue == nil {
		// there is a nonce in the metadata but no nonce value
		commitment := oracle.CommitmentBytes(new.NodeId, new.CommitmentAtxId)
		cpuProviderID := CPUProviderID()

		wo, err := oracle.New(
			oracle.WithProviderID(&cpuProviderID),
			oracle.WithCommitment(commitment),
			oracle.WithVRFDifficulty(make([]byte, 32)), // we are not looking for it, so set difficulty to 0
			oracle.WithScryptParams(new.Scrypt),
			oracle.WithLogger(logger),
		)
		if err != nil {
			return fmt.Errorf("failed to create oracle: %w", err)
		}

		result, err := wo.Position(*new.Nonce)
		if err != nil {
			return fmt.Errorf("failed to compute nonce value: %w", err)
		}
		new.NonceValue = result.Output
	}

	tmp, err := os.Create(fmt.Sprintf("%s.tmp", filename))
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer tmp.Close()

	if err := json.NewEncoder(tmp).Encode(new); err != nil {
		return fmt.Errorf("failed to encode metadata during migration: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close tmp file %s: %w", tmp.Name(), err)
	}

	if err := atomic.ReplaceFile(tmp.Name(), filename); err != nil {
		return fmt.Errorf("atomic replace: %w", err)
	}

	return nil
}
