package shared_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/post/shared"
)

func TestMarshalNonceValue(t *testing.T) {
	n := shared.NonceValue{0x01, 0x02, 0x03}
	data, err := n.MarshalJSON()
	require.NoError(t, err)
	require.EqualValues(t, `"010203"`, data)
}

func TestUnmarshalNonceValue(t *testing.T) {
	data := `"010203"`
	n := shared.NonceValue{}
	err := n.UnmarshalJSON([]byte(data))
	require.NoError(t, err)
	require.Equal(t, shared.NonceValue{0x01, 0x02, 0x03}, n)
}
