package gomadv3sim

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInspectClusterRecordProducesStableJSONAndTextProjection(t *testing.T) {
	spec := validSpec()
	scenarioChoices, err := NewScenarioChoicePlan([]ScenarioChoiceOverride{{
		Ordinal: 0, ID: "route", Occurrence: 1, Alternatives: []string{"alpha", "beta"}, Selected: 0,
	}})
	require.NoError(t, err)
	spec.ScenarioChoices = &scenarioChoices
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	oracle, err := StateInvariant("state.valid", true, []OracleEvidence{{Label: "state", Value: []byte("ok")}}, 1024)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{
		Outcome: OutcomeCompleted,
		History: []HistoryOperation{{ID: "op-1", Actor: "client", Kind: "request", Invocation: 1, Completion: 2}},
		Oracles: []OracleResult{oracle},
	})
	require.NoError(t, err)

	inspection, err := InspectClusterRecord(record)
	require.NoError(t, err)
	require.Equal(t, ClusterInspectionSchema, inspection.Schema)
	require.Equal(t, record.Identity, inspection.RecordIdentity)
	require.Equal(t, record.Static, inspection.Static)
	require.Equal(t, record.Models, inspection.Models)
	require.EqualValues(t, len(spec.Nodes), inspection.Counts.Nodes)
	require.EqualValues(t, 1, inspection.Counts.HistoryOperations)
	require.EqualValues(t, 1, inspection.Counts.OracleResults)
	require.EqualValues(t, 1, inspection.Counts.ScenarioChoiceOverrides)
	require.Equal(t, scenarioChoices.Identity, inspection.Tapes.ScenarioChoicePlanSHA256)
	require.Equal(t, record.Network.Snapshot.Identity, inspection.Terminal.NetworkSHA256)
	require.Equal(t, record.Volumes.Snapshot.Identity, inspection.Terminal.VolumeSHA256)

	encoded, err := EncodeClusterInspection(inspection)
	require.NoError(t, err)
	second, err := EncodeClusterInspection(inspection)
	require.NoError(t, err)
	require.Equal(t, encoded, second)
	require.NotContains(t, string(encoded), "\n")

	text, err := FormatClusterInspection(inspection)
	require.NoError(t, err)
	require.Contains(t, text, "outcome: completed")
	require.Contains(t, text, "history operations: 1")
	require.True(t, strings.HasSuffix(text, "\n"))
}

func TestInspectClusterRecordRejectsInvalidEvidence(t *testing.T) {
	_, err := InspectClusterRecord(ClusterRecord{})
	require.Error(t, err)
	_, err = EncodeClusterInspection(ClusterInspection{})
	require.Error(t, err)
}
