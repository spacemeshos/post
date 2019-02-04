package persistence

import (
	"bufio"
	"encoding/hex"
	"os"
	"path/filepath"
	"post-private/datatypes"
)

func PersistPostLabels(id []byte, labels []datatypes.Label) {
	dataPath := "data" // TODO @noam: put in config
	labelsPath := filepath.Join(dataPath, hex.EncodeToString(id))
	err := os.MkdirAll(labelsPath, os.ModePerm)
	if err != nil {
		panic(err) // TODO @noam: handle
	}
	filename := "all.labels"
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
