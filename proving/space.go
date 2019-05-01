package proving

import "fmt"

// In bytes. 1 peta-byte of storage.
// This would protect against number of labels uint64 overflow as well,
// since the number of labels per byte can be 8 at most (3 extra bit shifts).
const MaxSpace = 1 << 40 // 1099511627777

type Space uint64

// Validate validates whether the given space amount is valid.
func (s Space) Validate(labelGroupSize uint64) error {
	if s > MaxSpace {
		return fmt.Errorf("space (%d) is greater than the supported max (%d)", s, MaxSpace)
	}
	if uint64(s)%labelGroupSize != 0 {
		return fmt.Errorf("space (%d) must be a multiple of %d", s, labelGroupSize)
	}

	return nil
}

// LabelGroups returns the number of label groups of a given space amount.
func (s Space) LabelGroups(labelGroupSize uint64) uint64 {
	return uint64(s) / labelGroupSize
}
