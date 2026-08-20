package scenario

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestGeneratedFacadeCapturesAuthorSource(t *testing.T) {
	t.Parallel()

	authored := NewScenario("source", protocol.TargetIDNexusCancellation,
		[]Resource{NexusOperation("operation")},
		OnePath(
			ScheduleOperation("duplicate"),
			ScheduleOperation("duplicate"),
			RequireNexusCancellationWonExcludesSuccess(),
		),
	)
	_, err := Compile(context.Background(), authored, Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorAmbiguousProducer, compileErr.Category)
	require.Equal(t, "regress_test.go", filepath.Base(compileErr.Source.File))
	require.Positive(t, compileErr.Source.Line)
}

func TestResponseOptionsReadAsBehaviorAndCompileWithoutProtocolPlumbing(t *testing.T) {
	t.Parallel()

	authored := NewScenario("response-options", protocol.TargetIDFoundationDeliverySafety,
		[]Resource{Workflow("workflow")}, OnePath(
			ProgressEntity("sync", Synchronously()),
			ProgressEntity("async", Asynchronously()),
			ProgressEntity("deferred", Deferred()),
			ProgressEntity("blocking", BlockingFor(time.Second)),
			ProgressEntity("failure", FailingResponse()),
			RequireEntityProgress(),
		))
	suite, err := Compile(context.Background(), authored, Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, []protocol.ResponseMode{
		protocol.ResponseSynchronous, protocol.ResponseAsynchronous, protocol.ResponseDeferred,
		protocol.ResponseBlocking, protocol.ResponseFailure,
	}, responseModes(suite.Experiments[0].Actions))
}

func TestGeneratedFacadeCompilationIsDeterministic(t *testing.T) {
	t.Parallel()

	authored := NewScenario("deterministic", protocol.TargetIDFoundationDeliverySafety,
		[]Resource{Workflow("workflow")}, OnePath(
			ProgressEntity("progress", Synchronously()),
			RequireEntityProgress(),
		))
	limits := Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	}
	first, err := Compile(context.Background(), authored, limits)
	require.NoError(t, err)
	second, err := Compile(context.Background(), authored, limits)
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func responseModes(actions []protocol.Action) []protocol.ResponseMode {
	result := make([]protocol.ResponseMode, len(actions))
	for index, action := range actions {
		result[index] = action.EffectiveResponseMode()
	}
	return result
}
