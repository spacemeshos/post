package shared

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

func InitFileName(id []byte, index int) string {
	return fmt.Sprintf("%x-%d", id, index)
}

func IsInitFile(id []byte, file os.FileInfo) bool {
	return !file.IsDir() && strings.HasPrefix(file.Name(), fmt.Sprintf("%x", id))
}

func PersistProof(datadir string, id []byte, ch Challenge, proof []byte) error {
	dir := GetProofsDir(datadir, id)
	err := os.Mkdir(dir, OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("dir creation failure: %v", err)
	}

	filename := GetProofFilename(datadir, id, ch)
	err = ioutil.WriteFile(filename, proof, OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil
}

func FetchProof(datadir string, id []byte, challenge []byte) ([]byte, error) {
	filename := GetProofFilename(datadir, id, challenge)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrProofNotExist
		}

		return nil, fmt.Errorf("read file failure: %v", err)
	}

	return data, nil
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

func IsPowerOfTwo(x uint64) bool {
	return x != 0 &&
		x&(x-1) == 0
}
