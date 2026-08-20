package regress

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestGeneratedFacadeCapturesAuthorSource(t *testing.T) {
	t.Parallel()

	scenario := NewScenario("source", protocol.TargetIDNexusCancellation,
		[]Resource{NexusOperation("operation")},
		OnePath(
			ScheduleOperation("duplicate"),
			ScheduleOperation("duplicate"),
			RequireNexusCancellationWonExcludesSuccess(),
		),
	)
	_, err := compiler.Compile(context.Background(), scenario, compiler.Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	var compileErr *compiler.Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, compiler.ErrorAmbiguousProducer, compileErr.Category)
	require.Equal(t, "regress_test.go", filepath.Base(compileErr.Source.File))
	require.Positive(t, compileErr.Source.Line)
}

func TestResponseOptionsReadAsBehaviorAndCompileWithoutProtocolPlumbing(t *testing.T) {
	t.Parallel()

	scenario := NewScenario("response-options", protocol.TargetIDFoundationDeliverySafety,
		[]Resource{Workflow("workflow")}, OnePath(
			ProgressEntity("sync", Synchronously()),
			ProgressEntity("async", Asynchronously()),
			ProgressEntity("deferred", Deferred()),
			ProgressEntity("blocking", BlockingFor(time.Second)),
			ProgressEntity("failure", FailingResponse()),
			RequireEntityProgress(),
		))
	suite, err := compiler.Compile(context.Background(), scenario, compiler.Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, []protocol.ResponseMode{
		protocol.ResponseSynchronous, protocol.ResponseAsynchronous, protocol.ResponseDeferred,
		protocol.ResponseBlocking, protocol.ResponseFailure,
	}, responseModes(suite.Experiments[0].Actions))
}

func responseModes(actions []protocol.Action) []protocol.ResponseMode {
	result := make([]protocol.ResponseMode, len(actions))
	for index, action := range actions {
		result[index] = action.EffectiveResponseMode()
	}
	return result
}
