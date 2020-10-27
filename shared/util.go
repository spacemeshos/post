package shared

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/nullstyle/go-xdr/xdr3"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

func GetProofsDir(datadir string) string {
	return filepath.Join(datadir, "proofs")
}

func GetProofFilename(datadir string, challenge []byte) string {
	// Use a special name for the zero challenge, which otherwise
	// will result in empty filename.
	c := hex.EncodeToString(challenge)
	if c == "" {
		c = "0"
	}

	return filepath.Join(GetProofsDir(datadir), c)
}

func InitFileName(index int) string {
	return fmt.Sprintf("postdata_%d.bin", index)
}

func IsInitFile(file os.FileInfo) bool {
	if file.IsDir() {
		return false
	}

	re := regexp.MustCompile("postdata_(.*).bin")
	matches := re.FindStringSubmatch(file.Name())
	if len(matches) != 2 {
		return false
	}
	if _, err := strconv.Atoi(matches[1]); err != nil {
		return false
	}

	return true
}

type ProofWithMetadata struct {
	Proof         *Proof
	ProofMetadata *ProofMetadata
}

func PersistProof(datadir string, proof *Proof, proofMetadata *ProofMetadata) error {
	var w bytes.Buffer
	_, err := xdr.Marshal(&w, &ProofWithMetadata{proof, proofMetadata})
	if err != nil {
		return fmt.Errorf("encoding failure: %v", err)
	}

	dir := GetProofsDir(datadir)
	err = os.Mkdir(dir, OwnerReadWriteExec)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdir failure: %v", err)
	}
	filename := GetProofFilename(datadir, proofMetadata.Challenge)
	err = ioutil.WriteFile(filename, w.Bytes(), OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil
}

func FetchProof(datadir string, challenge []byte) (*Proof, *ProofMetadata, error) {
	filename := GetProofFilename(datadir, challenge)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrProofNotExist
		}

		return nil, nil, fmt.Errorf("read file failure: %v", err)
	}

	proofWithMetadata := &ProofWithMetadata{}
	_, err = xdr.Unmarshal(bytes.NewReader(data), proofWithMetadata)
	if err != nil {
		return nil, nil, err
	}

	return proofWithMetadata.Proof, proofWithMetadata.ProofMetadata, nil
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
