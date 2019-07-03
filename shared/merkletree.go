package shared

import "github.com/spacemeshos/merkle-tree/cache"

type (
	CacheReader = cache.CacheReader
)

type MTreeOutput struct {
	Reader cache.CacheReader
	Root   []byte
}

type MTreeOutputEntry struct {
	Index int
	*MTreeOutput
}

func Merge(outputs []*MTreeOutput) (*MTreeOutput, error) {
	switch len(outputs) {
	case 0:
		return nil, nil
	case 1:
		return outputs[0], nil
	default:
		readers := make([]CacheReader, len(outputs))
		for i, output := range outputs {
			readers[i] = output.Reader
		}

		reader, err := cache.Merge(readers)
		if err != nil {
			return nil, err
		}

		reader, root, err := cache.BuildTop(reader)
		if err != nil {
			return nil, err
		}

		return &MTreeOutput{reader, root}, nil
	}
}
