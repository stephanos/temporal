package scenario

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestCompileAllPathsCompletesDependenciesDeterministically(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "nexus-cancel",
		Target:     protocolcatalog.TargetIDNexusCancellation,
		Resources:  []Resource{{Identifier: "operation", Kind: protocolcatalog.EntityKindNexusOperation}},
		Root: AllPaths(
			Action("cancel", protocolcatalog.ActionKindRequestCancellation),
			AnyOrder(
				Action("commit", protocolcatalog.ActionKindCommitCancellation),
				Action("ownership", protocolcatalog.ActionKindAcquireOwnership),
			),
			Require(protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess),
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
		Target:     protocolcatalog.TargetIDProtocolAtomic,
		Resources:  []Resource{{Identifier: "callback", Kind: protocolcatalog.EntityKindCallback}},
		Root: OnePath(
			Action("respond", protocolcatalog.ActionKindRecordCallbackResponse,
				WithResponse(protocolexperiment.ResponseDeferred)),
			Require(protocolcatalog.PropertyIDCallbackResponseConsistency),
		),
	}, Limits{MaxPaths: 1, MaxActions: 1, MaxStates: 4, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.NoError(t, err)
	require.Equal(t, protocolexperiment.ResponseDeferred, suite.Experiments[0].Actions[0].ResponseMode)
	require.Equal(t, ModelReplayChecked, suite.Explain.ModelReplay.Status)
	require.Equal(t, "Umpire3.Temporal.System.CallbackResponse.behavior",
		suite.Explain.ModelReplay.CanonicalModel)
}

func TestCompileAllPathsEnumeratesOnlyValidLinearizations(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "unordered-assurance",
		Target:     protocolcatalog.TargetIDProtocolAtomic,
		Resources:  []Resource{{Identifier: "callback", Kind: protocolcatalog.EntityKindCallback}},
		Root: AllPaths(
			AnyOrder(
				Action("left", protocolcatalog.ActionKindRecordCallbackResponse),
				Action("right", protocolcatalog.ActionKindRecordCallbackResponse),
			),
			Require(protocolcatalog.PropertyIDCallbackResponseConsistency),
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
		Target:     protocolcatalog.TargetIDProtocolAtomic,
		Resources:  []Resource{{Identifier: "callback", Kind: protocolcatalog.EntityKindCallback}},
		Root: AllPaths(
			AnyOrder(
				Action("left", protocolcatalog.ActionKindRecordCallbackResponse),
				Action("right", protocolcatalog.ActionKindRecordCallbackResponse),
			),
			Require(protocolcatalog.PropertyIDCallbackResponseConsistency),
		),
	}

	_, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 2, MaxStates: 8, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorIncompleteEnumeration, compileErr.Category)
}

func TestCompileGroundsTypedSymbolAndAddsCausalOrder(t *testing.T) {
	t.Parallel()

	runID := Symbol{Name: "run-id", Type: protocolcatalog.SemanticTypeIDIdentity}
	scenario := Scenario{
		Identifier: "update-identity",
		Target:     protocolcatalog.TargetIDWorkflowUpdateLifecycle,
		Resources: []Resource{
			{Identifier: "workflow", Kind: protocolcatalog.EntityKindWorkflow},
			{Identifier: "update", Kind: protocolcatalog.EntityKindWorkflowUpdate},
		},
		Root: OnePath(
			Action("start", protocolcatalog.ActionKindStartUpdate),
			Bind(runID, Project("start", "update-id", protocolcatalog.SemanticTypeIDIdentity)),
			Action("dispatch", protocolcatalog.ActionKindDispatchWorkflowTask, WithArgument("update", runID.Value())),
			Action("accept", protocolcatalog.ActionKindAcceptUpdate),
			Action("history", protocolcatalog.ActionKindRecordUpdateHistory),
			Action("complete-task", protocolcatalog.ActionKindCompleteWorkflowTask),
			Action("complete", protocolcatalog.ActionKindCompleteUpdate),
			Require(protocolcatalog.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
		),
	}

	suite, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.NoError(t, err)
	require.Len(t, suite.Explain.Identities, 1)
	require.Equal(t, "start", suite.Explain.Identities[0].ProducerAction)
	require.Equal(t, []string{"dispatch"}, suite.Explain.Identities[0].ConsumerActions)
	require.Contains(t, suite.Experiments[0].Actions[0].Bindings, protocolexperiment.Binding{
		Symbol: "run-id", Type: "identity", Projection: "update-id",
	})
	require.Contains(t, suite.Experiments[0].Order, protocolexperiment.OrderConstraint{
		Before: "start", After: "dispatch", Relation: protocolexperiment.OrderRuntimeCausal,
	})
}

func TestCompileReportsSourceBearingBindingErrors(t *testing.T) {
	t.Parallel()

	scenario := Scenario{
		Identifier: "bad-binding",
		Target:     protocolcatalog.TargetIDWorkflowUpdateLifecycle,
		Resources:  []Resource{{Identifier: "update", Kind: protocolcatalog.EntityKindWorkflowUpdate}},
		Root: OnePath(
			ActionAt(Source{File: "scenario_test.go", Line: 42}, "start", protocolcatalog.ActionKindStartUpdate),
			BindAt(Source{File: "scenario_test.go", Line: 43},
				Symbol{Name: "run-id", Type: protocolcatalog.SemanticTypeIDString},
				Project("start", "missing", protocolcatalog.SemanticTypeIDIdentity),
			),
			Require(protocolcatalog.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
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
		Target:     protocolcatalog.TargetIDNexusCancellation,
		Resources:  []Resource{{Identifier: "operation", Kind: protocolcatalog.EntityKindNexusOperation}},
		Root: OnePath(
			Action("schedule", protocolcatalog.ActionKindScheduleOperation),
			Action("dispatch", protocolcatalog.ActionKindDispatchTask),
			Before(
				Action("cancel", protocolcatalog.ActionKindRequestCancellation),
				During(
					Fault("stale", protocolcatalog.FaultKindStaleWorkerCompletion),
					Repeat(2, Action("retry", protocolcatalog.ActionKindRetryTask)),
				),
			),
			Require(protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess),
		),
	}

	suite, err := Compile(context.Background(), scenario, Limits{MaxPaths: 1, MaxActions: 12, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.NoError(t, err)
	require.Len(t, suite.Experiments[0].Policies, 1)
	require.Len(t, suite.Experiments[0].Faults, 1)
	require.Equal(t, []string{"retry#1", "retry#2"}, suite.Experiments[0].Policies[0].Scope)
}

func TestCompileRejectsPathThatCanonicalModelCannotExecute(t *testing.T) {
	t.Parallel()

	authored := Scenario{
		Identifier: "impossible-stale-success",
		Target:     protocolcatalog.TargetIDNexusCancellation,
		Resources: []Resource{
			{Identifier: "operation", Kind: protocolcatalog.EntityKindNexusOperation},
			{Identifier: "worker", Kind: protocolcatalog.EntityKindNexusWorker},
		},
		Root: OnePath(
			Action("schedule", protocolcatalog.ActionKindScheduleOperation),
			Action("dispatch", protocolcatalog.ActionKindDispatchTask),
			Action("cancel", protocolcatalog.ActionKindRequestCancellation),
			Action("commit", protocolcatalog.ActionKindCommitCancellation),
			Action("ownership", protocolcatalog.ActionKindAcquireOwnership),
			Action("returned", protocolcatalog.ActionKindWorkerReturnsSuccess),
			Action("persist", protocolcatalog.ActionKindPersistSuccess,
				WithOutcomes(protocolexperiment.ActionOutcomeApplied)),
			Require(protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess),
		),
	}

	_, err := Compile(context.Background(), authored, Limits{
		MaxPaths: 1, MaxActions: 16, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorSemanticallyImpossible, compileErr.Category)
}

func TestCompilePreservesSuppressedAttemptWithoutApplyingAbstractTransition(t *testing.T) {
	t.Parallel()

	authored := Scenario{
		Identifier: "suppressed-stale-success",
		Target:     protocolcatalog.TargetIDNexusCancellation,
		Resources: []Resource{
			{Identifier: "operation", Kind: protocolcatalog.EntityKindNexusOperation},
			{Identifier: "worker", Kind: protocolcatalog.EntityKindNexusWorker},
		},
		Root: OnePath(
			Action("schedule", protocolcatalog.ActionKindScheduleOperation),
			Action("dispatch", protocolcatalog.ActionKindDispatchTask),
			Action("cancel", protocolcatalog.ActionKindRequestCancellation),
			Action("commit", protocolcatalog.ActionKindCommitCancellation),
			Action("ownership", protocolcatalog.ActionKindAcquireOwnership),
			Action("returned", protocolcatalog.ActionKindWorkerReturnsSuccess),
			Action("persist", protocolcatalog.ActionKindPersistSuccess),
			Require(protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess),
		),
	}

	suite, err := Compile(context.Background(), authored, Limits{
		MaxPaths: 1, MaxActions: 16, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, ModelReplayChecked, suite.Explain.ModelReplay.Status)
	require.Equal(t, "sound", suite.Explain.ModelReplay.Variant)
	persist := suite.Experiments[0].Actions[len(suite.Experiments[0].Actions)-1]
	require.Equal(t, []protocolexperiment.ActionOutcome{
		protocolexperiment.ActionOutcomeApplied,
		protocolexperiment.ActionOutcomeSuppressed,
		protocolexperiment.ActionOutcomeRejected,
		protocolexperiment.ActionOutcomeRetried,
		protocolexperiment.ActionOutcomeFaultIntercepted,
	}, persist.AllowedOutcomes)
}

func TestCompileReplaysFaultChallengeThroughMutatedExecutableView(t *testing.T) {
	t.Parallel()

	authored := Scenario{
		Identifier: "checked-stale-success-challenge",
		Target:     protocolcatalog.TargetIDNexusCancellation,
		Resources: []Resource{
			{Identifier: "operation", Kind: protocolcatalog.EntityKindNexusOperation},
			{Identifier: "worker", Kind: protocolcatalog.EntityKindNexusWorker},
		},
		Root: OnePath(
			Action("schedule", protocolcatalog.ActionKindScheduleOperation),
			Action("dispatch", protocolcatalog.ActionKindDispatchTask),
			Action("cancel", protocolcatalog.ActionKindRequestCancellation),
			Action("commit", protocolcatalog.ActionKindCommitCancellation),
			During(
				Fault("stale", protocolcatalog.FaultKindStaleWorkerCompletion),
				OnePath(
					Action("ownership", protocolcatalog.ActionKindAcquireOwnership),
					Action("returned", protocolcatalog.ActionKindWorkerReturnsSuccess),
					Action("persist", protocolcatalog.ActionKindPersistSuccess),
				),
			),
			Require(protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess),
		),
	}

	suite, err := Compile(context.Background(), authored, Limits{
		MaxPaths: 1, MaxActions: 16, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, ModelReplayChecked, suite.Explain.ModelReplay.Status)
	require.Equal(t, "stale-completion-guard-removed", suite.Explain.ModelReplay.Variant)
	require.Equal(t, []protocolcatalog.ActionKind{protocolcatalog.ActionKindScheduleOperation},
		suite.Explain.ModelReplay.LiveOnlyActions)
}

func TestModelTraceReceiptMustMatchTheGeneratedExecutableView(t *testing.T) {
	t.Parallel()

	view, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation,
		"stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	input := protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        view.Target,
		Property:      view.Property,
		World:         view.World,
		Variant:       view.Variant,
		SemanticHash:  "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Actions: []protocolcatalog.ActionKind{
			protocolcatalog.ActionKindDispatchTask,
			protocolcatalog.ActionKindAcquireOwnership,
			protocolcatalog.ActionKindWorkerReturnsSuccess,
			protocolcatalog.ActionKindPersistSuccess,
		},
	}
	digest, err := input.Digest()
	require.NoError(t, err)
	_, err = FromTraceReplayReceipt("mismatched-view", protocolchecker.TraceReplayReceipt{
		FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
		TraceDigest:   digest,
		Target:        input.Target,
		Property:      input.Property,
		World:         input.World,
		Variant:       input.Variant,
		SemanticHash:  input.SemanticHash,
		Actions:       input.Actions,
		Status:        protocolchecker.TraceReplayAccepted,
		TrustBadge:    protocolcatalog.TrustBadgeCheckedCertificate,
		Axioms:        []string{},
	})
	require.ErrorContains(t, err, "does not match generated executable view")
}

func TestCompileRejectsCycleWithActionSource(t *testing.T) {
	t.Parallel()

	updateID := Symbol{Name: "update-id", Type: protocolcatalog.SemanticTypeIDIdentity}
	scenario := Scenario{
		Identifier: "cycle",
		Target:     protocolcatalog.TargetIDWorkflowUpdateLifecycle,
		Resources:  []Resource{{Identifier: "update", Kind: protocolcatalog.EntityKindWorkflowUpdate}},
		Root: OnePath(
			ActionAt(Source{File: "cycle_test.go", Line: 17}, "dispatch", protocolcatalog.ActionKindDispatchWorkflowTask,
				WithArgument("update", updateID.Value())),
			Action("start", protocolcatalog.ActionKindStartUpdate),
			Bind(updateID, Project("start", "update-id", protocolcatalog.SemanticTypeIDIdentity)),
			Require(protocolcatalog.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
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

	identity := Symbol{Name: "identity", Type: protocolcatalog.SemanticTypeIDIdentity}
	base := Scenario{
		Identifier: "binding-error",
		Target:     protocolcatalog.TargetIDWorkflowUpdateLifecycle,
		Resources:  []Resource{{Identifier: "update", Kind: protocolcatalog.EntityKindWorkflowUpdate}},
	}

	rebind := base
	rebind.Root = OnePath(
		Action("start", protocolcatalog.ActionKindStartUpdate),
		Bind(identity, Project("start", "update-id", protocolcatalog.SemanticTypeIDIdentity)),
		BindAt(Source{File: "binding_test.go", Line: 9}, identity,
			Project("start", "update-id", protocolcatalog.SemanticTypeIDIdentity)),
		Require(protocolcatalog.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
	)
	_, err := Compile(context.Background(), rebind, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	var compileErr *Error
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorRebind, compileErr.Category)
	require.Equal(t, 9, compileErr.Source.Line)

	mistyped := base
	mistyped.Root = OnePath(
		Action("start", protocolcatalog.ActionKindStartUpdate),
		BindAt(Source{File: "binding_test.go", Line: 19},
			Symbol{Name: "identity", Type: protocolcatalog.SemanticTypeIDString},
			Project("start", "update-id", protocolcatalog.SemanticTypeIDString)),
		Require(protocolcatalog.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
	)
	_, err = Compile(context.Background(), mistyped, Limits{MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second})
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, ErrorTypeMismatch, compileErr.Category)
	require.Equal(t, 19, compileErr.Source.Line)
}

func actionIdentifiers(experiment protocolexperiment.Experiment) []string {
	identifiers := make([]string, len(experiment.Actions))
	for index, action := range experiment.Actions {
		identifiers[index] = action.Identifier
	}
	return identifiers
}
