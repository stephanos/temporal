package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultMonitorCatalogEvaluatesPropertiesGenerically(t *testing.T) {
	catalog, err := DefaultMonitorCatalog()
	require.NoError(t, err)
	require.Len(t, catalog.Programs, 15)

	program, ok := catalog.Program(PropertyIDNexusCancellationWonExcludesSuccess)
	require.True(t, ok)
	evaluation, err := program.Evaluate([]ObservedFact{
		{Observation: ObservationIDCancellationWon, Value: true},
		{Observation: ObservationIDStaleSuccessAbsent, Value: true},
	})
	require.NoError(t, err)
	require.True(t, evaluation.Complete)
	require.True(t, evaluation.Satisfied)

	evaluation, err = program.Evaluate([]ObservedFact{
		{Observation: ObservationIDCancellationWon, Value: true},
		{Observation: ObservationIDStaleSuccessAbsent, Value: false},
	})
	require.NoError(t, err)
	require.True(t, evaluation.Complete)
	require.False(t, evaluation.Satisfied)
}

func TestMonitorEvaluationReportsMissingAndConflictingFacts(t *testing.T) {
	catalog, err := DefaultMonitorCatalog()
	require.NoError(t, err)
	program, ok := catalog.Program(PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory)
	require.True(t, ok)

	evaluation, err := program.Evaluate([]ObservedFact{{Observation: ObservationIDUpdateAccepted, Value: true}})
	require.NoError(t, err)
	require.False(t, evaluation.Complete)
	require.Equal(t, []ObservationID{ObservationIDUpdateCompleted}, evaluation.Missing)

	_, err = program.Evaluate([]ObservedFact{
		{Observation: ObservationIDUpdateAccepted, Value: true},
		{Observation: ObservationIDUpdateAccepted, Value: false},
	})
	require.ErrorContains(t, err, "conflicting")
}

func TestMonitorCatalogRejectsProductSpecificUnknownObservation(t *testing.T) {
	catalog, err := DefaultMonitorCatalog()
	require.NoError(t, err)
	catalog.Programs[0].Expression = MonitorExpression{
		Operation:   MonitorObservation,
		Observation: "unknown-observation",
		Expected:    boolPointer(true),
	}
	require.ErrorContains(t, catalog.Validate(), "unknown observation")
}

func boolPointer(value bool) *bool {
	return &value
}
