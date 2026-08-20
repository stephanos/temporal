package explore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestExploreIsDeterministicAndExportsOrdinaryExperiments(t *testing.T) {
	t.Parallel()

	template := callbackTemplate()
	bounds := testBounds()
	first, err := Run(context.Background(), template, bounds)
	require.NoError(t, err)
	second, err := Run(context.Background(), template, bounds)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, StatusExhaustive, first.Status)
	require.Len(t, first.Candidates, 2)
	for _, candidate := range first.Candidates {
		require.NoError(t, candidate.Experiment.Validate())
	}
}

func TestExploreReportsResourceLimitedWithoutClaimingExhaustive(t *testing.T) {
	t.Parallel()

	bounds := testBounds()
	bounds.MaxAssignments = 1
	report, err := Run(context.Background(), callbackTemplate(), bounds)
	require.NoError(t, err)
	require.Equal(t, StatusResourceLimited, report.Status)
	require.False(t, report.Complete)
	require.NotEmpty(t, report.Omissions)
}

func TestExploreRejectsInvalidConstraintsBeforeSearch(t *testing.T) {
	t.Parallel()

	template := callbackTemplate()
	template.Required = []Fragment{{Hole: "missing", Value: "value"}}

	_, err := Run(context.Background(), template, testBounds())
	require.ErrorContains(t, err, "unknown hole")
}

func TestExploreCoversGeneratedNexusLifecycleDenominator(t *testing.T) {
	t.Parallel()

	values, err := NexusLifecycleValues()
	require.NoError(t, err)
	require.Len(t, values, 17)
	template := Template{
		Identifier: "nexus-lifecycle",
		Goal: Goal{
			Kind: GoalTransitionCoverage, Target: protocol.TargetIDFeatureNexus,
			Property: protocol.PropertyIDNexusOperationClosure,
		},
		Holes: []Hole{{Identifier: "edge", Kind: HoleAction, Values: values}},
		Build: func(assignment Assignment) (compiler.Scenario, error) {
			value := assignment["edge"]
			return compiler.Scenario{
				Identifier: "nexus-" + strings.ReplaceAll(value.Key, "/", "-"),
				Target:     protocol.TargetIDFeatureNexus,
				Resources: []compiler.Resource{{
					Identifier: "operation", Kind: protocol.EntityKindNexusOperation,
				}},
				Root: compiler.OnePath(
					compiler.Action("edge-action", nexusCoverageAction(value.Text)),
					compiler.Require(protocol.PropertyIDNexusOperationClosure),
				),
			}, nil
		},
		Observe: func(_ context.Context, candidate Candidate) ([]string, error) {
			return candidate.Coverage, nil
		},
	}
	bounds := testBounds()
	bounds.MaxAssignments = 17
	bounds.Compiler.MaxActions = 32

	report, err := Run(context.Background(), template, bounds)
	require.NoError(t, err)
	require.True(t, report.Complete)
	require.True(t, report.Coverage.Complete, "%+v", report)
	require.Equal(t, 17, report.Coverage.Total)
	require.Len(t, report.Coverage.Covered, 17)
	require.Empty(t, report.Coverage.Uncovered)
}

func TestTransitionCoverageRequiresPositiveRuntimeObservation(t *testing.T) {
	t.Parallel()

	values, err := NexusLifecycleValues()
	require.NoError(t, err)
	template := Template{
		Identifier: "unobserved-nexus-lifecycle",
		Goal: Goal{Kind: GoalTransitionCoverage, Target: protocol.TargetIDFeatureNexus,
			Property: protocol.PropertyIDNexusOperationClosure},
		Holes: []Hole{{Identifier: "edge", Kind: HoleAction, Values: values[:1]}},
		Build: func(Assignment) (compiler.Scenario, error) { return compiler.Scenario{}, nil },
	}

	_, err = Run(context.Background(), template, testBounds())
	require.ErrorContains(t, err, "positive runtime observation")
}

func TestSymmetryReductionRequiresAndReportsPreservation(t *testing.T) {
	t.Parallel()

	template := callbackTemplate()
	template.Holes = append(template.Holes, Hole{
		Identifier: "second", Kind: HoleEntityCount,
		Values: []Value{{Key: "one", Integer: 1}, {Key: "two", Integer: 2}},
	})
	template.SymmetryGroups = [][]string{{"variant", "second"}}
	_, err := Run(context.Background(), template, testBounds())
	require.ErrorContains(t, err, "preservation")

	template.SymmetryPreservationChecked = true
	report, err := Run(context.Background(), template, testBounds())
	require.NoError(t, err)
	require.Positive(t, report.Pruned.Symmetry)
}

func callbackTemplate() Template {
	return Template{
		Identifier: "callback-template",
		Goal:       Goal{Kind: GoalChallengeSafety, Property: protocol.PropertyIDCallbackResponseConsistency},
		Holes: []Hole{{
			Identifier: "variant", Kind: HoleAction,
			Values: []Value{{Key: "first", Text: "first"}, {Key: "second", Text: "second"}},
		}},
		Build: func(assignment Assignment) (compiler.Scenario, error) {
			return compiler.Scenario{
				Identifier: "callback-" + assignment["variant"].Text,
				Target:     protocol.TargetIDProtocolAtomic,
				Resources:  []compiler.Resource{{Identifier: "callback", Kind: protocol.EntityKindCallback}},
				Root: compiler.OnePath(
					compiler.Action("respond-"+assignment["variant"].Text, protocol.ActionKindRecordCallbackResponse),
					compiler.Require(protocol.PropertyIDCallbackResponseConsistency),
				),
			}, nil
		},
	}
}

func testBounds() Bounds {
	return Bounds{
		MaxAssignments: 8,
		Compiler: compiler.Limits{
			MaxPaths: 4, MaxActions: 8, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
		},
	}
}

func nexusCoverageAction(action string) protocol.ActionKind {
	switch action {
	case "schedule", "reject":
		return protocol.ActionKindScheduleOperation
	case "attempt-failed":
		return protocol.ActionKindRetryTask
	case "start":
		return protocol.ActionKindDispatchTask
	case "succeed":
		return protocol.ActionKindPersistSuccess
	case "fail", "terminate":
		return protocol.ActionKindCloseNexusOperation
	case "cancel":
		return protocol.ActionKindCommitCancellation
	case "timeout":
		return protocol.ActionKindTimeoutNexusOperation
	default:
		return ""
	}
}
