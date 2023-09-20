package initialization

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func Fuzz_MigrateMetadata(f *testing.F) {
	nodeId := make([]byte, 32)
	rand.Read(nodeId)

	commitmentAtxId := make([]byte, 32)
	rand.Read(commitmentAtxId)

	f.Add(make([]byte, 32), make([]byte, 32), []byte{1}, uint64(67), uint64(1024), uint64(1024*1024), uint64(1024), uint32(4))
	f.Add(nodeId, commitmentAtxId, []byte{1, 23}, uint64(128), uint64(1024*1024), uint64(1024*1024*1024), uint64(2389712), uint32(16))

	f.Fuzz(func(t *testing.T, nodeId, commitmentAtxId, nonceValue []byte, nonce, labelsPerUnit, maxFileSize, lastPosition uint64, numUnits uint32) {
		if len(nodeId) != 32 || len(commitmentAtxId) != 32 {
			return
		}

		path := t.TempDir()

		f, err := os.Create(filepath.Join(path, MetadataFileName))
		require.NoError(t, err)
		defer f.Close()

		old := postMetadataV0{
			NodeId:          nodeId,
			CommitmentAtxId: commitmentAtxId,
			LabelsPerUnit:   labelsPerUnit,
			NumUnits:        numUnits,
			MaxFileSize:     maxFileSize,
			Nonce:           &nonce,
			NonceValue:      nonceValue,
			LastPosition:    &lastPosition,
		}

		if len(nonceValue) == 0 {
			old.NonceValue = nil
			old.Nonce = nil
		}

		require.NoError(t, json.NewEncoder(f).Encode(old))
		require.NoError(t, f.Close())

		log := zaptest.NewLogger(t)
		require.NoError(t, MigratePoST(path, log))

		metadata, err := LoadMetadata(path)
		require.NoError(t, err)

		require.Equal(t, 1, metadata.Version)

		require.Equal(t, nodeId, metadata.NodeId)
		require.Equal(t, commitmentAtxId, metadata.CommitmentAtxId)
		require.Equal(t, labelsPerUnit, metadata.LabelsPerUnit)
		require.Equal(t, numUnits, metadata.NumUnits)
		require.Equal(t, maxFileSize, metadata.MaxFileSize)
		if old.NonceValue == nil {
			require.Nil(t, metadata.NonceValue)
			require.Nil(t, metadata.Nonce)
		} else {
			require.Equal(t, old.NonceValue, metadata.NonceValue)
			require.Equal(t, *old.Nonce, *metadata.Nonce)
		}
		require.Equal(t, lastPosition, *metadata.LastPosition)

		require.NotNil(t, metadata.Scrypt)
	})
}

func Test_Migrate_MissingMetadataFile(t *testing.T) {
	path := t.TempDir()
	log := zaptest.NewLogger(t)
	require.ErrorIs(t, MigratePoST(path, log), ErrStateMetadataFileMissing)
}

func Test_Migrate_Adds_NonceValue(t *testing.T) {
	nonce := uint64(10)
	old := postMetadataV0{
		NodeId:          make([]byte, 32),
		CommitmentAtxId: make([]byte, 32),
		LabelsPerUnit:   1024,
		NumUnits:        4,
		MaxFileSize:     1024 * 1014,
		Nonce:           &nonce,
		NonceValue:      nil,
		LastPosition:    nil,
	}

	path := t.TempDir()
	f, err := os.Create(filepath.Join(path, MetadataFileName))
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, json.NewEncoder(f).Encode(old))

	log := zaptest.NewLogger(t)
	require.NoError(t, MigratePoST(path, log))

	metadata, err := LoadMetadata(path)
	require.NoError(t, err)

	require.Equal(t, 1, metadata.Version)

	require.NotNil(t, metadata.NonceValue)
	require.NotNil(t, metadata.Nonce)
}
