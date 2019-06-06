package shared

import (
	"encoding/hex"
	"errors"
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

func VerifyInitialized(cfg *Config, id []byte) error {
	initialized, err := isInitialized(cfg, id)
	if err != nil {
		return err
	}

	if !initialized {
		return ErrNotInitialized
	}

	return nil
}

func VerifyNotInitialized(cfg *Config, id []byte) error {
	initialized, err := isInitialized(cfg, id)
	if err != nil {
		return err
	}

	if initialized {
		return ErrAlreadyInitialized
	}

	return nil
}

func isInitialized(cfg *Config, id []byte) (bool, error) {
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

	var numOfFiles int
	for _, file := range files {
		if !file.IsDir() && uint64(file.Size()) == cfg.FileSize {
			numOfFiles++
		}
	}

	expectedNumOfFiles, err := NumOfFiles(cfg.SpacePerUnit, cfg.FileSize)
	if err != nil {
		return false, err
	}

	return numOfFiles == expectedNumOfFiles, nil
}
