package persistence

import (
	"bufio"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"post-private/datatypes"
)

const (
	dataPath = "data" // TODO @noam: put in config
	filename = "all.labels"
)

func PersistPostLabels(id []byte, labels []datatypes.Label) {
	labelsPath := filepath.Join(dataPath, hex.EncodeToString(id))
	err := os.MkdirAll(labelsPath, os.ModePerm)
	if err != nil {
		panic(err) // TODO @noam: handle
	}
	fullFilename := filepath.Join(labelsPath, filename)
	f, err := os.OpenFile(fullFilename, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err) // TODO @noam: handle
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	for _, l := range labels {
		nn, err := w.Write(l)
		if err != nil {
			panic(err) // TODO @noam: handle
		}
		if nn != datatypes.LabelSize {
			panic(err) // TODO @noam: handle
		}
	}
	err = w.Flush()
	if err != nil {
		panic(err) // TODO @noam: handle
	}
}

func ReadPostLabels(id []byte) ([]datatypes.Label, error) {
	fullFilename := filepath.Join(dataPath, hex.EncodeToString(id), filename)
	f, err := os.OpenFile(fullFilename, os.O_RDONLY, os.ModePerm)
	if os.IsNotExist(err) {
		return nil, err
	}
	if err != nil {
		panic(err)
	}
	r := bufio.NewReader(f)

	labels := make([]datatypes.Label, 0)
	for {
		var l datatypes.Label = make([]byte, datatypes.LabelSize)
		n, err := r.Read(l)
		if err == io.EOF {
			if n != 0 {
				return nil, io.ErrUnexpectedEOF
			}
			break
		}
		labels = append(labels, l)
	}
	return labels, nil
}
