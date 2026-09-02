package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestLiveCallerClosureReturnsAndPublishesOneExactOperationalSet(t *testing.T) {
	input := admitCallerClosureSet(t)
	ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
	defer cancel()

	output, err := runner.Run(
		ctx,
		input,
		callerClosureInputBinding(),
		"umpire.local.caller-closure.integration-1",
		Binding{},
	)
	require.NoError(t, err)
	run := output.ExperimentRun()
	rawEvidence := output.RawEvidence()
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, "closed", rawEvidence.CaptureStatus)
	require.Equal(t, []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}, sourceDefinitionIDs(rawEvidence.Sources))
	for _, source := range rawEvidence.Sources {
		require.Equal(t, "closed", source.Status)
	}
	require.Len(t, run.ControlAttempts, 1)
	require.Equal(t, "accepted", run.ControlAttempts[0].Status)
	require.EqualValues(t, "0", run.Cleanup.OpenHandleCount)
	require.Empty(t, run.KnownGaps)
	require.Empty(t, rawEvidence.KnownGaps)
	receipt := rawFactsByKind(rawEvidence.Facts, artifactv2.ControlReceiptKindDefinitionID)
	require.Len(t, receipt, 1)
	require.Equal(t, []string{
		artifactv2.ControlReceiptActionFieldDefinitionID,
		artifactv2.ControlReceiptAttemptFieldDefinitionID,
		artifactv2.ControlReceiptOccurrenceFieldDefinitionID,
		artifactv2.ControlReceiptStatusFieldDefinitionID,
	}, rawEvidenceFieldDefinitionIDs(receipt[0]))

	runBytes, err := artifact.EncodeExperimentRunV2(run)
	require.NoError(t, err)
	rawEvidenceBytes, err := artifact.EncodeRawEvidenceV2(rawEvidence)
	require.NoError(t, err)
	for _, encoded := range [][]byte{runBytes, rawEvidenceBytes} {
		require.True(t, bytes.HasSuffix(encoded, []byte("\n")))
		require.False(t, bytes.HasSuffix(encoded, []byte("\n\n")))
		require.Contains(t, string(encoded), "\n  \"")
	}

	executable, ok := input.Executable()
	require.True(t, ok)
	expected, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)
	require.Equal(t, expected.Identity(), output.AdmittedSet().Identity())
	require.Equal(t, expected.ManifestBytes(), output.AdmittedSet().ManifestBytes())

	destination, err := artifact.PublishSet(t.TempDir(), output.AdmittedSet())
	require.NoError(t, err)
	reopened, err := artifact.LoadSet(destination)
	require.NoError(t, err)
	require.Equal(t, output.AdmittedSet().Identity(), reopened.Identity())
	require.Equal(t, output.AdmittedSet().Checksum(), reopened.Checksum())
	require.Equal(t, output.AdmittedSet().ManifestSHA256(), reopened.ManifestSHA256())
	require.Equal(t, output.AdmittedSet().ManifestBytes(), reopened.ManifestBytes())
}

func TestLiveFaultedCallerClosureReturnsClosedFaultRealizationEvidence(t *testing.T) {
	input := admitCallerClosureDuplicateDeliverySet(t)
	ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
	defer cancel()

	output, err := runner.Run(
		ctx,
		input,
		callerClosureDuplicateDeliveryInputBinding(),
		"umpire.local.caller-closure.duplicate-delivery.integration-1",
		Binding{},
	)
	require.NoError(t, err)
	run := output.ExperimentRun()
	rawEvidence := output.RawEvidence()
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, "closed", rawEvidence.CaptureStatus)
	require.Equal(t, []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}, sourceDefinitionIDs(rawEvidence.Sources))
	for index, source := range rawEvidence.Sources {
		require.Equal(t, "closed", source.Status)
		require.Equal(t, source.SourceDefinitionID, run.SourceClosures[index].SourceDefinitionID)
		require.Equal(t, source.FactCount, run.SourceClosures[index].RecordCount)
		require.Equal(t, source.ByteCount, run.SourceClosures[index].ByteCount)
	}
	require.Len(t, run.ControlAttempts, 1)
	require.Equal(t, "accepted", run.ControlAttempts[0].Status)
	require.NotNil(t, run.ControlAttempts[0].ReceiptFactDefinitionID)
	require.Equal(t, forceCloseOccurrenceDefinitionID, run.ControlAttempts[0].OccurrenceDefinitionID)
	require.Equal(t, forceCloseActionDefinitionID, run.ControlAttempts[0].ActionDefinitionID)
	require.EqualValues(t, "1", run.ControlAttempts[0].Attempt)
	require.EqualValues(t, "0", run.Cleanup.OpenHandleCount)
	require.Empty(t, run.KnownGaps)
	require.Empty(t, rawEvidence.KnownGaps)
	executable, ok := input.Executable()
	require.True(t, ok)
	require.Len(t, executable.Experiment().Plan.RequestedFaults, 1)
	require.Equal(t, duplicateDeliveryFaultDefinitionID,
		executable.Experiment().Plan.RequestedFaults[0].DefinitionID)
	require.Equal(t, forceCloseOccurrenceDefinitionID,
		executable.Experiment().Plan.RequestedFaults[0].Value)

	synthetic := rawEvidenceFactsWithField(
		rawEvidence,
		umpireruntime.EvidenceFieldSyntheticContributionMarker,
	)
	require.Len(t, synthetic, 1)
	require.Equal(t, umpireruntime.EvidenceKindParticipantCommandSyntheticDuplicate,
		synthetic[0].KindDefinitionID)
	require.Equal(t, duplicateDeliverySyntheticFieldOrder[:],
		rawEvidenceFieldDefinitionIDs(synthetic[0]))
	require.Equal(t, "1", rawFactFieldValue(t, synthetic[0],
		"umpire.evidence.field.synthetic-contribution-count"))
	require.Equal(t, duplicateObservationFactKind, rawFactFieldValue(t, synthetic[0],
		"umpire.evidence.field.synthetic-contribution-marker"))
	require.Equal(t, duplicateDeliveryFaultDefinitionID, rawFactFieldValue(t, synthetic[0],
		umpireruntime.EvidenceFieldFaultDefinitionID))
	require.Equal(t, duplicateDeliveryFaultReceiptDefinitionID, rawFactFieldValue(t, synthetic[0],
		umpireruntime.EvidenceFieldFaultReceiptDefinitionID))
	require.Equal(t, "nexus.capability.cancellation", rawFactFieldValue(t, synthetic[0],
		umpireruntime.EvidenceFieldCapabilityDefinitionID))
	require.Equal(t, "1", rawFactFieldValue(t, synthetic[0],
		umpireruntime.EvidenceFieldCancellationRequestedCount))
	require.Equal(t, "1", rawFactFieldValue(t, synthetic[0],
		umpireruntime.EvidenceFieldCancellationCompletedCount))
	receipt := rawFactsByKind(rawEvidence.Facts, artifactv2.ControlReceiptKindDefinitionID)
	require.Len(t, receipt, 1)
	require.Equal(t, []string{
		artifactv2.ControlReceiptActionFieldDefinitionID,
		artifactv2.ControlReceiptAttemptFieldDefinitionID,
		umpireruntime.EvidenceFieldCapabilityDefinitionID,
		umpireruntime.EvidenceFieldFaultDefinitionID,
		umpireruntime.EvidenceFieldFaultReceiptDefinitionID,
		artifactv2.ControlReceiptOccurrenceFieldDefinitionID,
		umpireruntime.EvidenceFieldOperationCorrelationID,
		artifactv2.ControlReceiptStatusFieldDefinitionID,
	}, rawEvidenceFieldDefinitionIDs(receipt[0]))
	for _, definitionID := range []string{
		umpireruntime.EvidenceFieldCapabilityDefinitionID,
		umpireruntime.EvidenceFieldFaultDefinitionID,
		umpireruntime.EvidenceFieldFaultReceiptDefinitionID,
		umpireruntime.EvidenceFieldOperationCorrelationID,
	} {
		require.Equal(t, rawFactFieldValue(t, synthetic[0], definitionID),
			rawFactFieldValue(t, receipt[0], definitionID))
	}

	callback := rawEvidenceFactsWithField(
		rawEvidence,
		umpireruntime.EvidenceFieldCancellationCallbackCount,
	)
	require.Len(t, callback, 2)
	require.Equal(t, "1", rawFactFieldValue(t, synthetic[0],
		umpireruntime.EvidenceFieldCancellationCallbackCount))
	require.Equal(t, []string{callback[0].FactDefinitionID},
		synthetic[0].CausalFactDefinitionIDs)
	requested, err := exactHistoryEventFact(
		rawEvidence,
		"temporal.history.NexusOperationCancelRequested",
	)
	require.NoError(t, err)
	completed, err := exactHistoryEventFact(
		rawEvidence,
		"temporal.history.NexusOperationCancelRequestCompleted",
	)
	require.NoError(t, err)
	require.Equal(t, []string{requested.FactDefinitionID}, completed.CausalFactDefinitionIDs)
	for _, definitionID := range []string{
		umpireruntime.EvidenceFieldOperationCorrelationID,
		umpireruntime.EvidenceFieldRunCorrelationID,
		umpireruntime.EvidenceFieldWorkflowCorrelationID,
	} {
		expected := rawFactFieldValue(t, synthetic[0], definitionID)
		require.NotEmpty(t, expected)
		for _, fact := range []artifactv2.RawEvidenceFact{callback[0], requested, completed} {
			require.Equal(t, expected, rawFactFieldValue(t, fact, definitionID))
		}
	}
}

func sourceDefinitionIDs(sources []artifactv2.RawEvidenceSource) []string {
	result := make([]string, len(sources))
	for index, source := range sources {
		result[index] = source.SourceDefinitionID
	}
	return result
}

func rawFactFieldValue(t *testing.T, fact artifactv2.RawEvidenceFact, definitionID string) string {
	t.Helper()
	for _, field := range fact.Fields {
		if field.FieldDefinitionID == definitionID {
			return stringValue(t, field.Value)
		}
	}
	require.FailNow(t, "raw evidence field is missing", definitionID)
	return ""
}

func stringValue(t *testing.T, value any) string {
	t.Helper()
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return string(typed)
	default:
		require.FailNow(t, "raw evidence field has an unexpected value type")
		return ""
	}
}
