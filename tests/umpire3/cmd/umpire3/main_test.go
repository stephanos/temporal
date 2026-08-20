package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplainUsesStableDiagnosticData(t *testing.T) {
	result, err := execute(context.Background(), []string{
		"explain", "-experiment", "../../testdata/update-lifecycle.json",
	})
	require.NoError(t, err)
	value, ok := result.(explanation)
	require.True(t, ok)
	require.Equal(t, "workflow-update-lifecycle-v1", value.ExperimentID)
	require.Equal(t, "workflow-update.accepted-completes-through-history", value.Property)
	require.Contains(t, value.RequiredCapabilities, "update")
	require.NotEmpty(t, value.ExperimentDigest)
}

func TestCommandsFailBeforeAllocationWhenRequiredInputsAreMissing(t *testing.T) {
	for _, command := range []string{"run", "replay", "campaign", "qualify"} {
		_, err := execute(context.Background(), []string{command})
		require.Error(t, err, command)
	}
}

func TestUnknownCommandListsStableSurface(t *testing.T) {
	_, err := execute(context.Background(), []string{"unknown"})
	require.EqualError(t, err, `unknown command "unknown": expected explain, run, replay, campaign, or qualify`)
}
