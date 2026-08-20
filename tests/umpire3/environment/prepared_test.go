package environment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestPreparedEnvironmentIsSingleUse(t *testing.T) {
	session := &preparedTestSession{}
	factory, err := PrepareOnce([]string{"workflow-task-control"}, session)
	require.NoError(t, err)
	prepared, err := factory.Prepare(context.Background(), protocol.Experiment{})
	require.NoError(t, err)
	require.Equal(t, session, prepared)
	_, err = factory.Prepare(context.Background(), protocol.Experiment{})
	require.EqualError(t, err, "prepared environment is single use")
}

type preparedTestSession struct{}

func (*preparedTestSession) Realize(context.Context, protocol.Action, Bindings) (ActionEvidence, error) {
	return ActionEvidence{}, nil
}

func (*preparedTestSession) Observe(context.Context, protocol.Checkpoint, Bindings) (Observation, error) {
	return Observation{}, nil
}

func (*preparedTestSession) Cleanup(context.Context) CleanupResult {
	return CleanupResult{Complete: true}
}

func (*preparedTestSession) RecoveryMetadata() map[string]string {
	return nil
}
