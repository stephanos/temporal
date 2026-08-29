package nexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"google.golang.org/protobuf/proto"
)

func TestProjectTerminalHistoryDrainsTheIteratorAndClosesTheCausalChain(t *testing.T) {
	request := checkedCallerClosureRequest(t, "project-history")
	command, ok := request.Command(umpireruntime.CommandObserve)
	require.True(t, ok)
	iterator := &historyIteratorStub{events: closedCallerHistory()}

	facts, err := projectTerminalHistory(command, iterator, requestCorrelations(request), 1)
	require.NoError(t, err)
	require.Len(t, facts, len(iterator.events))
	require.Equal(t, len(iterator.events)+1, iterator.hasNextCalls)
	require.Equal(t, len(iterator.events), iterator.nextCalls)
	for index, fact := range facts {
		require.Equal(t, umpireruntime.EvidenceSourceHistory, fact.SourceDefinitionID())
		require.Equal(t, []string{
			umpireruntime.EvidenceFieldEventID,
			umpireruntime.EvidenceFieldEventType,
			umpireruntime.EvidenceFieldOperationCorrelationID,
			umpireruntime.EvidenceFieldRunCorrelationID,
			umpireruntime.EvidenceFieldWorkflowCorrelationID,
		}, factFieldDefinitionIDs(fact))
		if index == 0 {
			require.Empty(t, fact.CausalDefinitionIDs())
			continue
		}
		require.Equal(t, []string{facts[index-1].DefinitionID()}, fact.CausalDefinitionIDs())
	}
	require.Equal(t,
		"temporal.history.WorkflowExecutionCanceled",
		factFieldValue(t, facts[len(facts)-1], umpireruntime.EvidenceFieldEventType),
	)
}

func TestProjectTerminalHistoryRejectsEveryIncompleteOrCorruptClosure(t *testing.T) {
	request := checkedCallerClosureRequest(t, "reject-history")
	command, ok := request.Command(umpireruntime.CommandObserve)
	require.True(t, ok)
	correlations := requestCorrelations(request)

	for _, test := range []struct {
		name          string
		events        []*historypb.HistoryEvent
		iteratorError error
		cancellations uint64
		retained      int
	}{
		{name: "iterator error after partial history", events: closedCallerHistory()[:2], iteratorError: errors.New("page failed"), cancellations: 1, retained: 2},
		{name: "missing terminal event", events: closedCallerHistory()[:5], cancellations: 1, retained: 5},
		{name: "duplicate event ID", events: mutateHistory(closedCallerHistory(), func(events []*historypb.HistoryEvent) {
			events[2].EventId = events[1].EventId
		}), cancellations: 1, retained: 2},
		{name: "missing control receipt event", events: mutateHistory(closedCallerHistory(), func(events []*historypb.HistoryEvent) {
			events[4].EventType = enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED
		}), cancellations: 1, retained: 6},
		{name: "duplicate required event", events: append(closedCallerHistory(), &historypb.HistoryEvent{
			EventId: 7, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED,
		}), cancellations: 1, retained: 7},
		{name: "missing cancellation callback", events: closedCallerHistory(), cancellations: 0, retained: 6},
		{name: "duplicate cancellation callback", events: closedCallerHistory(), cancellations: 2, retained: 6},
		{name: "nil history event", events: mutateHistory(closedCallerHistory(), func(events []*historypb.HistoryEvent) {
			events[2] = nil
		}), cancellations: 1, retained: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			iterator := &historyIteratorStub{events: test.events, terminalError: test.iteratorError}
			facts, err := projectTerminalHistory(command, iterator, correlations, test.cancellations)
			require.Error(t, err)
			require.Len(t, facts, test.retained)
		})
	}

	t.Run("history capacity N plus one", func(t *testing.T) {
		events := make([]*historypb.HistoryEvent, command.Limit().MaxRecords())
		for index := range events {
			events[index] = &historypb.HistoryEvent{
				EventId: int64(index + 1), EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
			}
		}
		iterator := &historyIteratorStub{events: events}
		facts, err := projectTerminalHistory(command, iterator, correlations, 1)
		require.ErrorIs(t, err, errHistoryCapacity)
		require.Len(t, facts, int(command.Limit().MaxRecords()-1))
		require.LessOrEqual(t, iterator.nextCalls, int(command.Limit().MaxRecords()))
	})
}

func TestValidateExecutionClosureAdmitsOnlyTheClosedMechanicalFourMemberSet(t *testing.T) {
	executable, admitted, run, rawEvidence := closedExecutionFixture(t)
	require.NoError(t, validateExecutionClosure(executable, admitted, run, rawEvidence))

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.ExperimentRun, *artifactv2.RawEvidence)
	}{
		{
			name: "non-allowlisted field",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				setRawFieldID(
					rawEvidence.Facts[len(rawEvidence.Facts)-1].Fields,
					umpireruntime.EvidenceFieldEndpointIdentity,
					"umpire.evidence.field.headers",
				)
			},
		},
		{
			name: "wrong digest disposition",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				setRawFieldDisposition(
					rawEvidence.Facts[len(rawEvidence.Facts)-1].Fields,
					umpireruntime.EvidenceFieldEndpointIdentity,
					"plain",
				)
			},
		},
		{
			name: "noncanonical fact order",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				rawEvidence.Facts[2], rawEvidence.Facts[3] = rawEvidence.Facts[3], rawEvidence.Facts[2]
			},
		},
		{
			name: "source byte count drift",
			mutate: func(run *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				count, err := strconv.ParseUint(rawEvidence.Sources[2].ByteCount.String(), 10, 64)
				require.NoError(t, err)
				rawEvidence.Sources[2].ByteCount = artifactv2.NaturalFromUint64(count + 1)
				run.SourceClosures[2].ByteCount = rawEvidence.Sources[2].ByteCount
			},
		},
		{
			name: "closed successful output with a known gap",
			mutate: func(run *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				subject := umpireruntime.EvidenceSourceHistory
				gap := artifactv2.KnownGap{Kind: "input", Code: "umpire.evidence.gap.capacity", Subject: &subject}
				run.KnownGaps = []artifactv2.KnownGap{gap}
				rawEvidence.KnownGaps = []artifactv2.KnownGap{gap}
			},
		},
		{
			name: "broken history causal chain",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				rawEvidence.Facts[7].CausalFactDefinitionIDs = []string{}
			},
		},
		{
			name: "stale run binding",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				rawEvidence.Run.ArtifactChecksum = testDigest('f')
			},
		},
		{
			name: "nonterminal complete history",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				setRawField(rawEvidence.Facts[7].Fields, umpireruntime.EvidenceFieldEventType,
					"temporal.history.WorkflowExecutionFailed")
			},
		},
		{
			name: "control receipt mismatch",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				setRawField(rawEvidence.Facts[1].Fields, artifactv2.ControlReceiptStatusFieldDefinitionID,
					"rejected")
			},
		},
		{
			name: "closed cleanup with an open handle",
			mutate: func(_ *artifactv2.ExperimentRun, rawEvidence *artifactv2.RawEvidence) {
				setRawField(rawEvidence.Facts[0].Fields, umpireruntime.EvidenceFieldOpenHandleCount,
					json.Number("1"))
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			executable, admitted, run, rawEvidence := closedExecutionFixture(t)
			test.mutate(&run, &rawEvidence)
			run = sealRun(t, run)
			if test.name != "stale run binding" {
				rawEvidence.Run = artifactv2.ExperimentRunArtifactBinding(run)
			}
			rawEvidence = sealRawEvidence(t, rawEvidence)
			if candidate, err := executable.AdmitExecution(run, rawEvidence); err == nil {
				admitted = candidate
			}
			require.Error(t, validateExecutionClosure(executable, admitted, run, rawEvidence))
		})
	}
}

func closedCallerHistory() []*historypb.HistoryEvent {
	return []*historypb.HistoryEvent{
		{EventId: 1, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
		{EventId: 2, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED},
		{EventId: 3, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED},
		{EventId: 4, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUESTED},
		{EventId: 5, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED},
		{EventId: 6, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED},
	}
}

func mutateHistory(
	events []*historypb.HistoryEvent,
	mutate func([]*historypb.HistoryEvent),
) []*historypb.HistoryEvent {
	cloned := make([]*historypb.HistoryEvent, len(events))
	for index, event := range events {
		cloned[index] = proto.Clone(event).(*historypb.HistoryEvent)
	}
	mutate(cloned)
	return cloned
}

func requestCorrelations(request umpireruntime.CheckedRunRequest) adapterCorrelations {
	result := adapterCorrelations{}
	for _, correlation := range request.Correlations() {
		switch correlation.Kind() {
		case umpireruntime.CorrelationWorkflow:
			result.workflow = correlation.Identity()
		case umpireruntime.CorrelationOperation:
			result.operation = correlation.Identity()
		}
	}
	return result
}

func factFieldValue(t *testing.T, fact umpireruntime.Fact, definitionID string) string {
	t.Helper()
	for _, field := range fact.Fields() {
		if field.DefinitionID() == definitionID {
			return field.Value()
		}
	}
	require.FailNow(t, "fact field is missing", definitionID)
	return ""
}

func factFieldDefinitionIDs(fact umpireruntime.Fact) []string {
	fields := fact.Fields()
	result := make([]string, len(fields))
	for index, field := range fields {
		result[index] = field.DefinitionID()
	}
	return result
}

type historyIteratorStub struct {
	events        []*historypb.HistoryEvent
	terminalError error
	index         int
	errorReturned bool
	hasNextCalls  int
	nextCalls     int
}

func (i *historyIteratorStub) HasNext() bool {
	i.hasNextCalls++
	return i.index < len(i.events) || i.terminalError != nil && !i.errorReturned
}

func (i *historyIteratorStub) Next() (*historypb.HistoryEvent, error) {
	i.nextCalls++
	if i.index < len(i.events) {
		event := i.events[i.index]
		i.index++
		return event, nil
	}
	i.errorReturned = true
	return nil, i.terminalError
}

func closedExecutionFixture(t *testing.T) (
	artifact.ExecutableSet,
	artifact.AdmittedSet,
	artifactv2.ExperimentRun,
	artifactv2.RawEvidence,
) {
	t.Helper()
	input := admitCallerClosureSet(t)
	executable, ok := input.Executable()
	require.True(t, ok)
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	runIdentity := "umpire.local.caller-closure.closed-fixture"
	receiptID := "umpire.runtime.fact.control-receipt.fixture"

	sources := []artifactv2.RawEvidenceSource{
		closedSource(umpireruntime.EvidenceSourceCleanup, 1),
		closedSource(umpireruntime.EvidenceSourceControlReceipt, 1),
		closedSource(umpireruntime.EvidenceSourceHistory, 6),
		closedSource(umpireruntime.EvidenceSourceParticipantOutput, 1),
	}
	run := artifactv2.ExperimentRun{
		FormatVersion:        artifactv2.ExperimentRunFormat,
		RunIdentity:          runIdentity,
		BehaviorFingerprint:  testDigest('a'),
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Attempt:              artifactv2.NaturalFromUint64(1),
		OperationalStatus:    "succeeded",
		PhaseOutcomes: []artifactv2.PhaseOutcome{
			succeededPhase("preparation", 1),
			succeededPhase("realization", 2),
			succeededPhase("observation", 3),
			succeededPhase("isolation", 4),
			succeededPhase("cleanup", 5),
		},
		ControlAttempts: []artifactv2.ControlAttempt{{
			OccurrenceDefinitionID:  forceCloseOccurrenceDefinitionID,
			ActionDefinitionID:      forceCloseActionDefinitionID,
			Attempt:                 artifactv2.NaturalFromUint64(1),
			ReceiptFactDefinitionID: &receiptID,
			Status:                  "accepted",
		}},
		SourceClosures: []artifactv2.SourceClosure{
			closedClosure(umpireruntime.EvidenceSourceCleanup, 1),
			closedClosure(umpireruntime.EvidenceSourceControlReceipt, 1),
			closedClosure(umpireruntime.EvidenceSourceHistory, 6),
			closedClosure(umpireruntime.EvidenceSourceParticipantOutput, 1),
		},
		Cleanup:    artifactv2.CleanupOutcome{Status: "complete", OpenHandleCount: artifactv2.NaturalFromUint64(0)},
		Limits:     configuration.PhaseLimits,
		KnownGaps:  []artifactv2.KnownGap{},
		Provenance: fixtureProvenance(),
	}
	run = sealRun(t, run)

	correlations := requestCorrelations(checkedCallerClosureRequest(t, "closed-fixture"))
	facts := []artifactv2.RawEvidenceFact{
		{
			FactDefinitionID:        "umpire.runtime.fact.cleanup.fixture",
			SourceDefinitionID:      umpireruntime.EvidenceSourceCleanup,
			Ordinal:                 artifactv2.NaturalFromUint64(0),
			KindDefinitionID:        "umpire.evidence.kind.cleanup",
			CausalFactDefinitionIDs: []string{},
			Fields: []artifactv2.RawEvidenceField{
				plainNumberField(umpireruntime.EvidenceFieldOpenHandleCount, "0"),
				plainField(umpireruntime.EvidenceFieldStatus, "complete"),
			},
		},
		{
			FactDefinitionID:        receiptID,
			SourceDefinitionID:      umpireruntime.EvidenceSourceControlReceipt,
			Ordinal:                 artifactv2.NaturalFromUint64(0),
			KindDefinitionID:        artifactv2.ControlReceiptKindDefinitionID,
			CausalFactDefinitionIDs: []string{},
			Fields: []artifactv2.RawEvidenceField{
				plainField(artifactv2.ControlReceiptActionFieldDefinitionID, forceCloseActionDefinitionID),
				plainNumberField(artifactv2.ControlReceiptAttemptFieldDefinitionID, "1"),
				plainField(artifactv2.ControlReceiptOccurrenceFieldDefinitionID, forceCloseOccurrenceDefinitionID),
				plainField(artifactv2.ControlReceiptStatusFieldDefinitionID, "accepted"),
			},
		},
	}
	previous := ""
	for index, event := range closedCallerHistory() {
		factID := fmt.Sprintf("umpire.runtime.fact.history.%020d.fixture", event.GetEventId())
		causes := []string{}
		if previous != "" {
			causes = []string{previous}
		}
		facts = append(facts, artifactv2.RawEvidenceFact{
			FactDefinitionID:        factID,
			SourceDefinitionID:      umpireruntime.EvidenceSourceHistory,
			Ordinal:                 artifactv2.NaturalFromUint64(uint64(index)),
			KindDefinitionID:        "umpire.evidence.kind.workflow-history-event",
			CausalFactDefinitionIDs: causes,
			Fields: []artifactv2.RawEvidenceField{
				plainNumberField(umpireruntime.EvidenceFieldEventID, fmt.Sprintf("%d", event.GetEventId())),
				plainField(umpireruntime.EvidenceFieldEventType, "temporal.history."+event.GetEventType().String()),
				plainField(umpireruntime.EvidenceFieldOperationCorrelationID, correlations.operation),
				plainField(umpireruntime.EvidenceFieldRunCorrelationID, runIdentity),
				plainField(umpireruntime.EvidenceFieldWorkflowCorrelationID, correlations.workflow),
			},
		})
		previous = factID
	}
	facts = append(facts, artifactv2.RawEvidenceFact{
		FactDefinitionID:        "umpire.runtime.fact.participant.fixture",
		SourceDefinitionID:      umpireruntime.EvidenceSourceParticipantOutput,
		Ordinal:                 artifactv2.NaturalFromUint64(0),
		KindDefinitionID:        "umpire.evidence.kind.participant-command",
		CausalFactDefinitionIDs: []string{},
		Fields: []artifactv2.RawEvidenceField{
			plainNumberField(umpireruntime.EvidenceFieldCancellationCallbackCount, "1"),
			{
				FieldDefinitionID: umpireruntime.EvidenceFieldEndpointIdentity,
				Disposition:       "sha256",
				Value:             testDigest('e'),
			},
		},
	})
	rawEvidence := artifactv2.RawEvidence{
		FormatVersion:        artifactv2.RawEvidenceFormat,
		RunIdentity:          runIdentity,
		BehaviorFingerprint:  testDigest('b'),
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Run:                  artifactv2.ExperimentRunArtifactBinding(run),
		CaptureStatus:        "closed",
		Sources:              sources,
		Facts:                facts,
		KnownGaps:            []artifactv2.KnownGap{},
		Provenance:           fixtureProvenance(),
	}
	recomputeFixtureByteCounts(t, &run, &rawEvidence)
	run = sealRun(t, run)
	rawEvidence.Run = artifactv2.ExperimentRunArtifactBinding(run)
	rawEvidence = sealRawEvidence(t, rawEvidence)
	admitted, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)
	return executable, admitted, run, rawEvidence
}

func recomputeFixtureByteCounts(
	t *testing.T,
	run *artifactv2.ExperimentRun,
	rawEvidence *artifactv2.RawEvidence,
) {
	t.Helper()
	byteCounts := make(map[string]uint64, len(rawEvidence.Sources))
	for _, fact := range rawEvidence.Facts {
		encoded, err := artifact.CanonicalPretty(fact)
		require.NoError(t, err)
		byteCounts[fact.SourceDefinitionID] += uint64(len(encoded))
	}
	for index := range rawEvidence.Sources {
		count := artifactv2.NaturalFromUint64(byteCounts[rawEvidence.Sources[index].SourceDefinitionID])
		rawEvidence.Sources[index].ByteCount = count
		run.SourceClosures[index].ByteCount = count
	}
}

func closedSource(definitionID string, facts uint64) artifactv2.RawEvidenceSource {
	return artifactv2.RawEvidenceSource{
		SourceDefinitionID: definitionID,
		Status:             "closed",
		FactCount:          artifactv2.NaturalFromUint64(facts),
		ByteCount:          artifactv2.NaturalFromUint64(facts),
	}
}

func closedClosure(definitionID string, facts uint64) artifactv2.SourceClosure {
	return artifactv2.SourceClosure{
		SourceDefinitionID: definitionID,
		Status:             "closed",
		RecordCount:        artifactv2.NaturalFromUint64(facts),
		ByteCount:          artifactv2.NaturalFromUint64(facts),
	}
}

func succeededPhase(phase string, at uint64) artifactv2.PhaseOutcome {
	started := artifactv2.NaturalFromUint64(at)
	finished := artifactv2.NaturalFromUint64(at + 1)
	return artifactv2.PhaseOutcome{
		Phase:                phase,
		Status:               "succeeded",
		StartedAtUnixMillis:  &started,
		FinishedAtUnixMillis: &finished,
	}
}

func plainField(definitionID string, value any) artifactv2.RawEvidenceField {
	return artifactv2.RawEvidenceField{
		FieldDefinitionID: definitionID,
		Disposition:       "plain",
		Value:             value,
	}
}

func plainNumberField(definitionID string, value string) artifactv2.RawEvidenceField {
	return plainField(definitionID, json.Number(value))
}

func fixtureProvenance() artifactv2.Provenance {
	return artifactv2.Provenance{
		SourceDefinitionIDs: []string{"umpire.runtime.engine"},
		SourceLocations: []artifactv2.SourceLocation{{
			Path:       "tools/umpire/runtime/engine.go",
			Line:       artifactv2.NaturalFromUint64(1),
			Column:     artifactv2.NaturalFromUint64(1),
			Provenance: "runtime-engine",
		}},
	}
}

func sealRun(t *testing.T, run artifactv2.ExperimentRun) artifactv2.ExperimentRun {
	t.Helper()
	sealed, err := artifactv2.SealExperimentRun(run)
	require.NoError(t, err)
	return sealed
}

func sealRawEvidence(t *testing.T, rawEvidence artifactv2.RawEvidence) artifactv2.RawEvidence {
	t.Helper()
	sealed, err := artifactv2.SealRawEvidence(rawEvidence)
	require.NoError(t, err)
	return sealed
}

func setRawField(fields []artifactv2.RawEvidenceField, definitionID string, value any) {
	for index := range fields {
		if fields[index].FieldDefinitionID == definitionID {
			fields[index].Value = value
			return
		}
	}
}

func setRawFieldID(fields []artifactv2.RawEvidenceField, definitionID string, replacement string) {
	for index := range fields {
		if fields[index].FieldDefinitionID == definitionID {
			fields[index].FieldDefinitionID = replacement
			return
		}
	}
}

func setRawFieldDisposition(fields []artifactv2.RawEvidenceField, definitionID string, disposition string) {
	for index := range fields {
		if fields[index].FieldDefinitionID == definitionID {
			fields[index].Disposition = disposition
			return
		}
	}
}

func testDigest(character byte) string {
	value := make([]byte, 64)
	for index := range value {
		value[index] = character
	}
	return "sha256:" + string(value)
}
