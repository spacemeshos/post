package shared

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPowerOfTwo(t *testing.T) {
	r := require.New(t)

	r.False(IsPowerOfTwo(0))
	r.False(IsPowerOfTwo(3))
	r.False(IsPowerOfTwo(5))
	r.False(IsPowerOfTwo(6))
	r.False(IsPowerOfTwo(7))
	r.False(IsPowerOfTwo(9))

	r.True(IsPowerOfTwo(1))
	r.True(IsPowerOfTwo(2))
	r.True(IsPowerOfTwo(4))
	r.True(IsPowerOfTwo(8))
	r.True(IsPowerOfTwo(16))
	r.True(IsPowerOfTwo(32))
	r.True(IsPowerOfTwo(64))
}
