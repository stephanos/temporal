package umpire3_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNexusFeatureExcludesSystemMechanics(t *testing.T) {
	source, err := os.ReadFile("model/Temporal/Families/NexusCancellation/Feature.lean")
	require.NoError(t, err)
	for _, forbidden := range []string{"Task", "RPC", "Shard", "Persist", "Owner"} {
		require.NotContains(t, string(source), forbidden)
	}
}
