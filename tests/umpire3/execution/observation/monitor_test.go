package observation

import (
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolmonitor "go.temporal.io/server/tests/umpire3/protocol/monitor"
)

func TestGeneratedMonitorEvaluation(t *testing.T) {
	catalog, err := protocolmonitor.DefaultMonitorCatalog()
	require.NoError(t, err)
	require.Len(t, catalog.Programs, 16)

	program, found := catalog.Program(protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess)
	require.True(t, found)
	evaluation, err := EvaluateMonitor(program, []protocolmonitor.ObservedFact{
		{Observation: protocolcatalog.ObservationIDCancellationWon, Value: true},
		{Observation: protocolcatalog.ObservationIDStaleSuccessAbsent, Value: true},
	})
	require.NoError(t, err)
	require.Equal(t, protocolmonitor.MonitorEvaluation{
		Complete:  true,
		Satisfied: true,
		Missing:   []protocolcatalog.ObservationID{},
	}, evaluation)

	evaluation, err = EvaluateMonitor(program, []protocolmonitor.ObservedFact{
		{Observation: protocolcatalog.ObservationIDCancellationWon, Value: true},
		{Observation: protocolcatalog.ObservationIDStaleSuccessAbsent, Value: false},
	})
	require.NoError(t, err)
	require.True(t, evaluation.Complete)
	require.False(t, evaluation.Satisfied)
	require.Equal(t, []protocolcatalog.ObservationID{protocolcatalog.ObservationIDStaleSuccessAbsent},
		evaluation.Contradictions)
}

func TestMonitorEvaluationReportsMissingAndConflictingFacts(t *testing.T) {
	catalog, err := protocolmonitor.DefaultMonitorCatalog()
	require.NoError(t, err)
	program, found := catalog.Program(protocolcatalog.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory)
	require.True(t, found)

	evaluation, err := EvaluateMonitor(program, []protocolmonitor.ObservedFact{{
		Observation: protocolcatalog.ObservationIDUpdateAccepted, Value: true,
	}})
	require.NoError(t, err)
	require.False(t, evaluation.Complete)
	require.Equal(t, []protocolcatalog.ObservationID{protocolcatalog.ObservationIDUpdateCompleted}, evaluation.Missing)

	_, err = EvaluateMonitor(program, []protocolmonitor.ObservedFact{
		{Observation: protocolcatalog.ObservationIDUpdateAccepted, Value: true},
		{Observation: protocolcatalog.ObservationIDUpdateAccepted, Value: false},
	})
	require.ErrorContains(t, err, "conflicting")
}

func TestGeneratedMonitorEvaluatesNexusProgress(t *testing.T) {
	catalog, err := protocolmonitor.DefaultMonitorCatalog()
	require.NoError(t, err)
	program, found := catalog.Program(protocolcatalog.PropertyID("nexus-operation.progress"))
	require.True(t, found)

	evaluation, err := EvaluateMonitor(program, []protocolmonitor.ObservedFact{{
		Observation: protocolcatalog.ObservationID("nexus-operation-progressed"), Value: false,
	}})
	require.NoError(t, err)
	require.True(t, evaluation.Complete)
	require.False(t, evaluation.Satisfied)
}
