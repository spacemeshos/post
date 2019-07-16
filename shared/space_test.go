package shared

import (
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSpace_Validate(t *testing.T) {
	r := require.New(t)
	r.EqualError(ValidateSpace(uint64(LabelGroupSize-1)), fmt.Sprintf("space (%d) must be a multiple of %d", LabelGroupSize-1, LabelGroupSize))
	r.NoError(ValidateSpace(uint64(LabelGroupSize)))
	r.EqualError(ValidateSpace(uint64(LabelGroupSize+1)), fmt.Sprintf("space (%d) must be a multiple of %d", LabelGroupSize+1, LabelGroupSize))

	r.EqualError(ValidateSpace(uint64(MaxSpace-1)), fmt.Sprintf("space (%d) must be a multiple of 32", MaxSpace-1))
	r.NoError(ValidateSpace(uint64(MaxSpace)))
	r.EqualError(ValidateSpace(uint64(MaxSpace+1)), fmt.Sprintf("space (%d) is greater than the supported max (%d)", MaxSpace+1, MaxSpace))
}

func TestAvailableSpace(t *testing.T) {
	r := require.New(t)

	// Sanity test.
	space := AvailableSpace(config.DefaultConfig().DataDir)
	r.True(space > 0)
}
