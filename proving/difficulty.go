package proving

type Difficulty uint8

// LabelBits returns the number of bits per label.
func (d Difficulty) LabelBits() uint64 {
	return 1 << (8 - d)
}

// LabelMask returns a bit mask according to the label size.
func (d Difficulty) LabelMask() uint8 {
	return (uint8(1) << d.LabelBits()) - 1
}

// ByteIndex clears the part of the index within a byte. This can be applied to an absolute index or to an index within
// a leaf.
func (d Difficulty) ByteIndex(labelIndex uint64) uint64 {
	return labelIndex >> (d - 5)
}

// LeafIndex clears the part of the index within a leaf, leaving the absolute index of the leaf.
func (d Difficulty) LeafIndex(labelIndex uint64) uint64 {
	return labelIndex >> d
}

// IndexInLeaf returns the relative label index within a leaf.
func (d Difficulty) IndexInLeaf(labelIndex uint64) uint64 {
	return labelIndex &^ (^uint64(0) << d)
}

// IndexInByte returns the relative label index within the byte that contains it.
func (d Difficulty) IndexInByte(labelIndex uint64) uint64 {
	return labelIndex &^ (^uint64(0) << (d - 5))
}

func (d Difficulty) LabelsPerByte() uint64 {
	return 1 << (d - 5)
}
