package oracle_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/oracle"
	"github.com/stretchr/testify/require"
)

func TestComputeLabel(t *testing.T) {
	dir := os.Getenv("METADIR")
	require.NotEmpty(t, dir)
	meta, err := initialization.LoadMetadata(dir)
	require.NoError(t, err)
	rst, err := oracle.WorkOracle(
		oracle.WithComputeProviderID(initialization.CPUProviderID()),
		oracle.WithPosition(985063564),
		oracle.WithCommitment(oracle.CommitmentBytes(meta.NodeId, meta.CommitmentAtxId)),
		oracle.WithScryptParams(config.DefaultLabelParams()),
		oracle.WithComputeLeaves(false),
	)
	require.NoError(t, err)
	fmt.Println(rst)
}
