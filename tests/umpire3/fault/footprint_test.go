package fault

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecorderNormalizesOccurrencesWithoutRetainingRequestData(t *testing.T) {
	t.Parallel()

	recorder := NewRecorder()
	require.NoError(t, recorder.Record(Call{
		Protocol: "HTTP", Service: "nexus", Route: "/service/operation?token=secret",
		Direction: DirectionOutbound, Role: CallRoleInternal, Namespace: "namespace",
		Participant: "worker", Attempt: 2, Risk: 8,
		CausalReferences: []string{"workflow/run/2", "workflow/run/1", "workflow/run/1"},
	}))
	require.NoError(t, recorder.Record(Call{
		Protocol: "http", Service: "nexus", Route: "/service/operation",
		Direction: DirectionOutbound, Role: CallRoleInternal, Namespace: "namespace",
		Participant: "worker", Attempt: 3, Risk: 8,
	}))

	calls := recorder.Snapshot()
	require.Len(t, calls, 2)
	require.Equal(t, "http", calls[0].Protocol)
	require.Equal(t, "/service/operation", calls[0].Route)
	require.Equal(t, 1, calls[0].Occurrence)
	require.Equal(t, Interval{Start: 1, Stop: 2}, calls[0].Interval)
	require.Equal(t, []string{"workflow/run/1", "workflow/run/2"}, calls[0].CausalReferences)
	require.Equal(t, 2, calls[1].Occurrence)
	require.NotContains(t, recorder.Digest(), "secret")
}

func TestRecorderSelectsOnlyInternalFaultTargetsAndReconcilesDrift(t *testing.T) {
	t.Parallel()

	recorder := NewRecorder()
	require.NoError(t, recorder.Record(Call{
		Protocol: "grpc", Service: "frontend", Route: "StartNexusOperationExecution",
		Direction: DirectionOutbound, Role: CallRoleClientEntry, Namespace: "namespace", Participant: "driver",
	}))
	require.NoError(t, recorder.Record(Call{
		Protocol: "grpc", Service: "matching", Route: "DispatchNexusTask",
		Direction: DirectionOutbound, Role: CallRoleInternal, Namespace: "namespace", Participant: "frontend", Risk: 4,
	}))
	require.NoError(t, recorder.Record(Call{
		Protocol: "http", Service: "nexus", Route: "/service/operation",
		Direction: DirectionOutbound, Role: CallRoleInternal, Namespace: "namespace", Participant: "history", Risk: 8,
	}))
	require.NoError(t, recorder.Record(Call{
		Protocol: "http", Service: "nexus", Route: "/service/operation",
		Direction: DirectionOutbound, Role: CallRoleInternal, Namespace: "namespace", Participant: "history", Risk: 8,
	}))

	targets := FaultTargets(recorder.Snapshot(), 99, 3)
	require.Equal(t, []Footprint{
		{Protocol: "http", Service: "nexus", Route: "/service/operation", Occurrence: 1, Risk: 8, RealizationEvidence: true},
		{Protocol: "http", Service: "nexus", Route: "/service/operation", Occurrence: 2, Risk: 8, RealizationEvidence: true},
		{Protocol: "grpc", Service: "matching", Route: "DispatchNexusTask", Occurrence: 1, Risk: 4, RealizationEvidence: true},
	}, targets)
	require.NotContains(t, targets, Footprint{
		Protocol: "grpc", Service: "frontend", Route: "StartNexusOperationExecution",
	})
	require.Empty(t, ReconcileFootprints(targets, recorder.Snapshot(), nil))
}

func TestReconcileFootprintsReportsMissingAndUnexpectedButAllowsNoise(t *testing.T) {
	t.Parallel()

	declared := []Footprint{
		{Protocol: "http", Service: "nexus", Route: "/service/operation"},
		{Protocol: "grpc", Service: "matching", Route: "DispatchNexusTask"},
	}
	observed := []Call{
		{Protocol: "http", Service: "nexus", Route: "/service/operation"},
		{Protocol: "grpc", Service: "history", Route: "RecordActivityTaskStarted"},
		{Protocol: "grpc", Service: "health", Route: "Check"},
	}
	allowed := []Footprint{{Protocol: "grpc", Service: "health", Route: "Check"}}

	drift := ReconcileFootprints(declared, observed, allowed)
	require.Equal(t, []Footprint{{Protocol: "grpc", Service: "matching", Route: "DispatchNexusTask"}}, drift.Missing)
	require.Equal(t, []Footprint{{Protocol: "grpc", Service: "history", Route: "RecordActivityTaskStarted"}}, drift.Unexpected)
}

func TestBuildFootprintReportIsStableAcrossRuntimeIdentities(t *testing.T) {
	declared := []Footprint{{Protocol: "http", Service: "nexus", Route: "/service/operation"}}
	first, err := BuildFootprintReport(declared, []Call{{
		Protocol: "HTTP", Service: "nexus", Route: "/service/operation?token=secret",
		Direction: DirectionInbound, Role: CallRoleInternal, Namespace: "namespace-a",
		Participant: "handler", Attempt: 1, Occurrence: 1, Interval: Interval{Start: 1, Stop: 2},
		CausalReferences: []string{"request-a"}, Risk: 8,
	}}, nil)
	require.NoError(t, err)
	second, err := BuildFootprintReport(declared, []Call{{
		Protocol: "http", Service: "nexus", Route: "/service/operation",
		Direction: DirectionInbound, Role: CallRoleInternal, Namespace: "namespace-b",
		Participant: "handler", Attempt: 1, Occurrence: 1, Interval: Interval{Start: 1, Stop: 2},
		CausalReferences: []string{"request-b"}, Risk: 8,
	}}, nil)
	require.NoError(t, err)
	require.True(t, first.Complete)
	require.Equal(t, first.FootprintDigest, second.FootprintDigest)
	require.Equal(t, first.ReconciliationDigest, second.ReconciliationDigest)
	require.NoError(t, first.Validate())
}

func TestBuildFootprintReportFailsClosedOnDrift(t *testing.T) {
	report, err := BuildFootprintReport(
		[]Footprint{{Protocol: "grpc", Service: "matching", Route: "DispatchNexusTask"}},
		[]Call{{
			Protocol: "grpc", Service: "history", Route: "RecordNexusTaskStarted",
			Direction: DirectionOutbound, Role: CallRoleInternal, Namespace: "namespace",
			Participant: "history", Attempt: 1, Occurrence: 1, Interval: Interval{Start: 1, Stop: 2},
		}}, nil)
	require.NoError(t, err)
	require.False(t, report.Complete)
	require.Len(t, report.Drift.Missing, 1)
	require.Len(t, report.Drift.Unexpected, 1)
	require.ErrorContains(t, report.RequireComplete(), "footprint reconciliation drift")
}
