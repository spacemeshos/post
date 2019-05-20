package proving

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSpace_Validate(t *testing.T) {
	r := require.New(t)
	r.EqualError(ValidateSpace(LabelGroupSize-1), fmt.Sprintf("space (%d) must be a multiple of %d", LabelGroupSize-1, LabelGroupSize))
	r.NoError(ValidateSpace(LabelGroupSize))
	r.EqualError(ValidateSpace(LabelGroupSize+1), fmt.Sprintf("space (%d) must be a multiple of %d", LabelGroupSize+1, LabelGroupSize))

	r.EqualError(ValidateSpace(MaxSpace-1), fmt.Sprintf("space (%d) must be a multiple of 32", MaxSpace-1))
	r.NoError(ValidateSpace(MaxSpace))
	r.EqualError(ValidateSpace(MaxSpace+1), fmt.Sprintf("space (%d) is greater than the supported max (%d)", MaxSpace+1, MaxSpace))
}
