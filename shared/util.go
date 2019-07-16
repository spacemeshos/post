package shared

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/nullstyle/go-xdr/xdr3"
	"github.com/spacemeshos/post/config"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetInitDir(datadir string, id []byte) string {
	return filepath.Join(datadir, hex.EncodeToString(id))
}

func GetProofsDir(datadir string, id []byte) string {
	return filepath.Join(GetInitDir(datadir, id), "proofs")
}

func GetProofFilename(datadir string, id []byte, challenge []byte) string {
	// Use a special name for the zero challenge, which otherwise
	// will result in empty filename.
	c := hex.EncodeToString(challenge)
	if c == "" {
		c = "0"
	}

	return filepath.Join(GetProofsDir(datadir, id), c)
}

// TODO: use logic from initializer.State method, after resolving packages dependencies issue.
func VerifyInitCompleted(cfg *config.Config, id []byte) error {
	initialized, err := isInitialized(cfg, id)
	if err != nil {
		return err
	}

	if !initialized {
		return ErrInitNotCompleted
	}

	return nil
}

func isInitialized(cfg *config.Config, id []byte) (bool, error) {
	if id == nil {
		return false, errors.New("id is missing")
	}

	dir := GetInitDir(cfg.DataDir, id)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var numFiles int
	for _, file := range files {
		if !file.IsDir() && uint64(file.Size()) == cfg.FileSize {
			numFiles++
		}
	}

	expectedNumFiles, err := NumFiles(cfg.SpacePerUnit, cfg.FileSize)
	if err != nil {
		return false, err
	}

	return numFiles == expectedNumFiles, nil
}

func PersistProof(datadir string, proof *Proof) error {
	var w bytes.Buffer
	_, err := xdr.Marshal(&w, &proof)
	if err != nil {
		return fmt.Errorf("serialization failure: %v", err)
	}

	dir := GetProofsDir(datadir, proof.Identity)
	err = os.Mkdir(dir, OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %v", err)
	}

	filename := GetProofFilename(datadir, proof.Identity, proof.Challenge)
	err = ioutil.WriteFile(filename, w.Bytes(), OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil
}

func FetchProof(datadir string, id []byte, challenge []byte) (*Proof, error) {
	filename := GetProofFilename(datadir, id, challenge)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrProofNotExist
		}

		return nil, fmt.Errorf("read file failure: %v", err)
	}

	proof := &Proof{}
	_, err = xdr.Unmarshal(bytes.NewReader(data), proof)
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
