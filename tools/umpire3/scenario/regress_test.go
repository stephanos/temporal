package scenario

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestGeneratedFacadeCapturesAuthorSource(t *testing.T) {
	t.Parallel()

	authored := NexusCancellationScenario("source",
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

	authored := FoundationDeliverySafetyScenario("response-options",
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
	require.Equal(t, []protocolexperiment.ResponseMode{
		protocolexperiment.ResponseSynchronous, protocolexperiment.ResponseAsynchronous, protocolexperiment.ResponseDeferred,
		protocolexperiment.ResponseBlocking, protocolexperiment.ResponseFailure,
	}, responseModes(suite.Experiments[0].Actions))
}

func TestGeneratedFacadeCompilationIsDeterministic(t *testing.T) {
	t.Parallel()

	authored := FoundationDeliverySafetyScenario("deterministic",
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
	require.Equal(t, protocolcatalog.LeanVersion, first.Experiments[0].Model.LeanVersion)
}

func responseModes(actions []protocolexperiment.Action) []protocolexperiment.ResponseMode {
	result := make([]protocolexperiment.ResponseMode, len(actions))
	for index, action := range actions {
		result[index] = action.EffectiveResponseMode()
	}
	return result
}
