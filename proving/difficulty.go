package proving

import "fmt"

const MinDifficulty = 5 // 1 byte per label
const MaxDifficulty = 8 // 1 bit per label

type Difficulty uint8

// LabelsPerGroup returns the number of labels in a label group. A value between 32 and 256.
func (d Difficulty) LabelsPerGroup() uint64 {
	return 1 << d
}

// LabelsPerByte returns the number of labels in a single byte. A value between 1 and 8.
func (d Difficulty) LabelsPerByte() uint64 {
	return 1 << (d - MinDifficulty)
}

// LabelBits returns the number of bits per label.
func (d Difficulty) LabelBits() uint64 {
	return 1 << (MaxDifficulty - d)
}

// LabelMask returns a bit mask according to the label size.
func (d Difficulty) LabelMask() uint8 {
	return (uint8(1) << d.LabelBits()) - 1
}

// ByteIndex clears the part of the index within a byte. This can be applied to an absolute index or to an index within
// a leaf.
func (d Difficulty) ByteIndex(labelIndex uint64) uint64 {
	return labelIndex >> (d - MinDifficulty)
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
	return labelIndex &^ (^uint64(0) << (d - MinDifficulty))
}

func (d Difficulty) Validate() error {
	if d < MinDifficulty || d > MaxDifficulty {
		return fmt.Errorf("difficulty must be between %d and %d (received %d)", MinDifficulty, MaxDifficulty, d)
	}
	return nil
}
