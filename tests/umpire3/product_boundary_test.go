package umpire3_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNexusProductExcludesSystemMechanics(t *testing.T) {
	source, err := os.ReadFile("model/Temporal/Product/Nexus.lean")
	require.NoError(t, err)
	for _, forbidden := range []string{"Task", "RPC", "Shard", "Persist", "Owner"} {
		require.NotContains(t, string(source), forbidden)
	}
}
