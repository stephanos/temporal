package compiler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestCompileAllPathsCompletesDependenciesDeterministically(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "nexus-cancel",
		Target:     protocol.TargetIDNexusCancellation,
		Resources:  []Resource{{Identifier: "operation", Kind: protocol.EntityKindNexusOperation}},
		Root: AllPaths(
			Action("cancel", protocol.ActionKindRequestCancellation),
			AnyOrder(
				Action("commit", protocol.ActionKindCommitCancellation),
				Action("ownership", protocol.ActionKindAcquireOwnership),
			),
			Require(protocol.PropertyIDNexusCancellationWonExcludesSuccess),
		),
	}
	limits := Limits{MaxPaths: 8, MaxActions: 8, MaxStates: 100, MaxMemoryBytes: 1 << 20, MaxTime: time.Second}

	first, err := Compile(context.Background(), scenario, limits)
	require.NoError(t, err)
	second, err := Compile(context.Background(), scenario, limits)
	require.NoError(t, err)

	firstJSON, err := first.CanonicalJSON()
	require.NoError(t, err)
	secondJSON, err := second.CanonicalJSON()
	require.NoError(t, err)
	require.JSONEq(t, string(firstJSON), string(secondJSON))
	require.Equal(t, ExplainFormatVersion, first.Explain.FormatVersion)
	encoded, err := json.Marshal(first.Explain)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"complete"`)
	require.Len(t, first.Experiments, 1)
	require.Equal(t, []string{
		"generated-schedule-operation", "generated-dispatch-task", "cancel", "commit", "ownership",
	}, actionIdentifiers(first.Experiments[0]))
	require.Equal(t, []string{"schedule-operation", "dispatch-task"}, first.Explain.AddedActionKinds)
}

func TestCompileCarriesParticipantResponseSemanticsIntoExperiment(t *testing.T) {
	t.Parallel()

	suite, err := Compile(context.Background(), Scenario{
		Identifier: "deferred-callback",
		Target:     protocol.TargetIDProtocolAtomic,
		Resources:  []Resource{{Identifier: "callback", Kind: protocol.EntityKindCallback}},
		Root: OnePath(
			Action("respond", protocol.ActionKindRecordCallbackResponse,
				WithResponse(protocol.ResponseDeferred)),
			Require(protocol.PropertyIDCallbackResponseConsistency),
		),
	}, Limits{MaxPaths: 1, MaxActions: 1, MaxStates: 4, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.NoError(t, err)
	require.Equal(t, protocol.ResponseDeferred, suite.Experiments[0].Actions[0].ResponseMode)
}

func TestCompileAllPathsEnumeratesOnlyValidLinearizations(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "unordered-assurance",
		Target:     protocol.TargetIDProtocolAtomic,
		Resources:  []Resource{{Identifier: "callback", Kind: protocol.EntityKindCallback}},
		Root: AllPaths(
			AnyOrder(
				Action("left", protocol.ActionKindRecordCallbackResponse),
				Action("right", protocol.ActionKindRecordCallbackResponse),
			),
			Require(protocol.PropertyIDCallbackResponseConsistency),
		),
	}

	suite, err := Compile(context.Background(), scenario, Limits{MaxPaths: 2, MaxActions: 2, MaxStates: 8, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 2)
	require.Equal(t, []string{"left", "right"}, actionIdentifiers(suite.Experiments[0]))
	require.Equal(t, []string{"right", "left"}, actionIdentifiers(suite.Experiments[1]))
}

func TestCompileFailsInsteadOfTruncatingAllPaths(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "bounded-unordered",
		Target:     protocol.TargetIDProtocolAtomic,
		Resources:  []Resource{{Identifier: "callback", Kind: protocol.EntityKindCallback}},
		Root: AllPaths(
			AnyOrder(
				Action("left", protocol.ActionKindRecordCallbackResponse),
				Action("right", protocol.ActionKindRecordCallbackResponse),
			),
			Require(protocol.PropertyIDCallbackResponseConsistency),
		),
	}

	_, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 2, MaxStates: 8, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorIncompleteEnumeration, compileErr.Category)
}

func TestCompileGroundsTypedSymbolAndAddsCausalOrder(t *testing.T) {
	t.Parallel()

	runID := Symbol{Name: "run-id", Type: protocol.SemanticTypeIDIdentity}
	scenario := Scenario{
		Identifier: "update-identity",
		Target:     protocol.TargetIDWorkflowUpdateLifecycle,
		Resources: []Resource{
			{Identifier: "workflow", Kind: protocol.EntityKindWorkflow},
			{Identifier: "update", Kind: protocol.EntityKindWorkflowUpdate},
		},
		Root: OnePath(
			Action("start", protocol.ActionKindStartUpdate),
			Bind(runID, Project("start", "update-id", protocol.SemanticTypeIDIdentity)),
			Action("dispatch", protocol.ActionKindDispatchWorkflowTask, WithArgument("update", runID.Value())),
			Action("accept", protocol.ActionKindAcceptUpdate),
			Action("history", protocol.ActionKindRecordUpdateHistory),
			Action("complete-task", protocol.ActionKindCompleteWorkflowTask),
			Action("complete", protocol.ActionKindCompleteUpdate),
			Require(protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
		),
	}

	suite, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.NoError(t, err)
	require.Len(t, suite.Explain.Identities, 1)
	require.Equal(t, "start", suite.Explain.Identities[0].ProducerAction)
	require.Equal(t, []string{"dispatch"}, suite.Explain.Identities[0].ConsumerActions)
	require.Contains(t, suite.Experiments[0].Actions[0].Bindings, protocol.Binding{
		Symbol: "run-id", Type: "identity", Projection: "update-id",
	})
	require.Contains(t, suite.Experiments[0].Order, protocol.OrderConstraint{
		Before: "start", After: "dispatch", Relation: protocol.OrderRuntimeCausal,
	})
}

func TestCompileReportsSourceBearingBindingErrors(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "bad-binding",
		Target:     protocol.TargetIDWorkflowUpdateLifecycle,
		Resources:  []Resource{{Identifier: "update", Kind: protocol.EntityKindWorkflowUpdate}},
		Root: OnePath(
			ActionAt(Source{File: "scenario_test.go", Line: 42}, "start", protocol.ActionKindStartUpdate),
			BindAt(Source{File: "scenario_test.go", Line: 43},
				Symbol{Name: "run-id", Type: protocol.SemanticTypeIDString},
				Project("start", "missing", protocol.SemanticTypeIDIdentity),
			),
			Require(protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
		),
	}

	_, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorMissingProjection, compileErr.Category)
	require.Equal(t, "scenario_test.go", compileErr.Source.File)
	require.Equal(t, 43, compileErr.Source.Line)
}

func TestCompileSupportsDuringRepeatAndBefore(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "fault-shape",
		Target:     protocol.TargetIDNexusCancellation,
		Resources:  []Resource{{Identifier: "operation", Kind: protocol.EntityKindNexusOperation}},
		Root: OnePath(
			Action("schedule", protocol.ActionKindScheduleOperation),
			Action("dispatch", protocol.ActionKindDispatchTask),
			Before(
				Action("cancel", protocol.ActionKindRequestCancellation),
				During(
					Fault("stale", protocol.FaultKindStaleWorkerCompletion),
					Repeat(2, Action("retry", protocol.ActionKindRetryTask)),
				),
			),
			Require(protocol.PropertyIDNexusCancellationWonExcludesSuccess),
		),
	}

	suite, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 12, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.NoError(t, err)
	require.Len(t, suite.Experiments[0].Policies, 1)
	require.Len(t, suite.Experiments[0].Faults, 1)
	require.Equal(t, []string{"retry#1", "retry#2"}, suite.Experiments[0].Policies[0].Scope)
}

func TestCompileRejectsCycleWithActionSource(t *testing.T) {
	t.Parallel()

	updateID := Symbol{Name: "update-id", Type: protocol.SemanticTypeIDIdentity}
	scenario := Scenario{
		Identifier: "cycle",
		Target:     protocol.TargetIDWorkflowUpdateLifecycle,
		Resources:  []Resource{{Identifier: "update", Kind: protocol.EntityKindWorkflowUpdate}},
		Root: OnePath(
			ActionAt(Source{File: "cycle_test.go", Line: 17}, "dispatch", protocol.ActionKindDispatchWorkflowTask,
				WithArgument("update", updateID.Value())),
			Action("start", protocol.ActionKindStartUpdate),
			Bind(updateID, Project("start", "update-id", protocol.SemanticTypeIDIdentity)),
			Require(protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
		),
	}

	_, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorCycle, compileErr.Category)
	require.Equal(t, "cycle_test.go", compileErr.Source.File)
}

func TestCompileRejectsRebindAndProjectionTypeMismatch(t *testing.T) {
	t.Parallel()

	identity := Symbol{Name: "identity", Type: protocol.SemanticTypeIDIdentity}
	base := Scenario{
		Identifier: "binding-error",
		Target:     protocol.TargetIDWorkflowUpdateLifecycle,
		Resources:  []Resource{{Identifier: "update", Kind: protocol.EntityKindWorkflowUpdate}},
	}

	rebind := base
	rebind.Root = OnePath(
		Action("start", protocol.ActionKindStartUpdate),
		Bind(identity, Project("start", "update-id", protocol.SemanticTypeIDIdentity)),
		BindAt(Source{File: "binding_test.go", Line: 9}, identity,
			Project("start", "update-id", protocol.SemanticTypeIDIdentity)),
		Require(protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
	)
	_, err := Compile(context.Background(), rebind, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorRebind, compileErr.Category)
	require.Equal(t, 9, compileErr.Source.Line)

	mistyped := base
	mistyped.Root = OnePath(
		Action("start", protocol.ActionKindStartUpdate),
		BindAt(Source{File: "binding_test.go", Line: 19},
			Symbol{Name: "identity", Type: protocol.SemanticTypeIDString},
			Project("start", "update-id", protocol.SemanticTypeIDString)),
		Require(protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
	)
	_, err = Compile(context.Background(), mistyped, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorTypeMismatch, compileErr.Category)
	require.Equal(t, 19, compileErr.Source.Line)
}

func actionIdentifiers(experiment protocol.Experiment) []string {
	identifiers := make([]string, len(experiment.Actions))
	for index, action := range experiment.Actions {
		identifiers[index] = action.Identifier
	}
	return identifiers
}
