package proving_test

import (
	"fmt"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	maxSpace       = proving.MaxSpace
	labelGroupSize = initialization.LabelGroupSize
)

func TestSpace_Validate(t *testing.T) {
	r := require.New(t)
	r.EqualError(proving.Space(labelGroupSize-1).Validate(labelGroupSize), fmt.Sprintf("space (%d) must be a multiple of %d", labelGroupSize-1, labelGroupSize))
	r.NoError(proving.Space(labelGroupSize).Validate(labelGroupSize))
	r.EqualError(proving.Space(labelGroupSize+1).Validate(labelGroupSize), fmt.Sprintf("space (%d) must be a multiple of %d", labelGroupSize+1, labelGroupSize))

	r.EqualError(proving.Space(maxSpace-1).Validate(labelGroupSize), fmt.Sprintf("space (%d) must be a multiple of 32", maxSpace-1))
	r.NoError(proving.Space(maxSpace).Validate(labelGroupSize))
	r.EqualError(proving.Space(maxSpace+1).Validate(labelGroupSize), fmt.Sprintf("space (%d) is greater than the supported max (%d)", maxSpace+1, maxSpace))
}
