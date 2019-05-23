package shared

type Params struct {
	SpacePerUnit                            uint64     `long:"space" description:"Space per unit, in bytes"`
	Difficulty                              Difficulty `long:"difficulty" description:"Computational cost of the initialization"`
	NumOfProvenLabels                       uint8      `long:"t" description:"Number of labels to prove in non-interactive proof (security parameter)"`
	LowestLayerToCacheDuringProofGeneration uint       `long:"cachelayer" description:"Lowest layer to cache in-memory during proof generation (optimization parameter)"`
}

func DefaultParams() *Params {
	return &Params{
		SpacePerUnit:                            SpacePerUnit,
		Difficulty:                              MinDifficulty,
		NumOfProvenLabels:                       NumOfProvenLabels,
		LowestLayerToCacheDuringProofGeneration: LowestLayerToCacheDuringProofGeneration,
	}
}
