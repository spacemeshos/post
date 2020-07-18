package shared

import (
	"github.com/spacemeshos/smutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAvailableSpace(t *testing.T) {
	r := require.New(t)

	// Sanity test.
	space := AvailableSpace(smutil.GetUserHomeDirectory())
	r.True(space > 0)
}
