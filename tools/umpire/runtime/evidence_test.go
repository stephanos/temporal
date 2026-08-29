package runtime

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestEvidenceAccumulatorRetainsExactlyNAndReportsNPlusOneBeforeAppend(t *testing.T) {
	for _, capacitySource := range evidenceSourceOrder {
		t.Run(capacitySource, func(t *testing.T) {
			atN := newTestEvidenceAccumulator(t)
			appendEvidenceCapacity(t, atN, capacitySource, 0)
			for _, source := range evidenceSourceOrder {
				require.NoError(t, atN.closeSource(source, "closed"))
			}
			sources, facts, gaps := atN.materialize()
			require.Len(t, facts, 4096)
			require.Empty(t, gaps)
			require.Equal(t, "closed", sourceStatusFromEvidence(sources, capacitySource))

			atNPlusOne := newTestEvidenceAccumulator(t)
			appendEvidenceCapacity(t, atNPlusOne, capacitySource, 1)
			for _, source := range evidenceSourceOrder {
				require.NoError(t, atNPlusOne.closeSource(source, "closed"))
			}
			sources, facts, gaps = atNPlusOne.materialize()
			require.Len(t, facts, 4096)
			require.Equal(t, "partial", sourceStatusFromEvidence(sources, capacitySource))
			require.Len(t, gaps, 1)
			require.Equal(t, "umpire.evidence.gap.capacity", gaps[0].Code)
			require.Equal(t, capacitySource, *gaps[0].Subject)
		})
	}
}

func TestEvidenceAccumulatorRejectsMutationsBeforeAppend(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, *evidenceAccumulator)
		fact    func(*testing.T) Fact
	}{
		{
			name: "duplicate fact",
			prepare: func(t *testing.T, accumulator *evidenceAccumulator) {
				outcome, err := accumulator.append(PhaseObservation, evidenceFact(t, "runtime.fact.duplicate", EvidenceSourceHistory, nil, nil))
				require.NoError(t, err)
				require.Equal(t, appendRetained, outcome)
			},
			fact: func(t *testing.T) Fact {
				return evidenceFact(t, "runtime.fact.duplicate", EvidenceSourceHistory, nil, nil)
			},
		},
		{
			name:    "causal orphan",
			prepare: func(*testing.T, *evidenceAccumulator) {},
			fact: func(t *testing.T) Fact {
				return evidenceFact(t, "runtime.fact.orphan", EvidenceSourceHistory, []string{"runtime.fact.missing"}, nil)
			},
		},
		{
			name: "canonical-order forward reference",
			prepare: func(t *testing.T, accumulator *evidenceAccumulator) {
				outcome, err := accumulator.append(PhasePreparation, evidenceFact(t, "runtime.fact.later-source", EvidenceSourceParticipantOutput, nil, nil))
				require.NoError(t, err)
				require.Equal(t, appendRetained, outcome)
			},
			fact: func(t *testing.T) Fact {
				return evidenceFact(t, "runtime.fact.earlier-source", EvidenceSourceHistory, []string{"runtime.fact.later-source"}, nil)
			},
		},
		{
			name: "source close race",
			prepare: func(t *testing.T, accumulator *evidenceAccumulator) {
				require.NoError(t, accumulator.closeSource(EvidenceSourceHistory, "closed"))
			},
			fact: func(t *testing.T) Fact {
				return evidenceFact(t, "runtime.fact.after-close", EvidenceSourceHistory, nil, nil)
			},
		},
		{
			name:    "field outside mechanical allowlist",
			prepare: func(*testing.T, *evidenceAccumulator) {},
			fact: func(t *testing.T) Fact {
				field, err := NewFactField("runtime.field.arbitrary", "secret")
				require.NoError(t, err)
				return evidenceFact(t, "runtime.fact.unknown-field", EvidenceSourceHistory, nil, []FactField{field})
			},
		},
		{
			name:    "adapter-supplied control receipt",
			prepare: func(*testing.T, *evidenceAccumulator) {},
			fact: func(t *testing.T) Fact {
				return evidenceFact(t, "runtime.fact.forged-control", EvidenceSourceControlReceipt, nil, nil)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			accumulator := newTestEvidenceAccumulator(t)
			test.prepare(t, accumulator)
			before := accumulator.retainedCount()
			outcome, err := accumulator.append(PhaseObservation, test.fact(t))
			require.Error(t, err)
			require.Equal(t, appendRejected, outcome)
			require.Equal(t, before, accumulator.retainedCount())
		})
	}
}

func TestEvidenceAccumulatorEnforcesPhaseByteLimitBeforeAppend(t *testing.T) {
	accumulator := newTestEvidenceAccumulator(t)
	limit := accumulator.limits[PhasePreparation]
	limit.maxBytes = 1
	accumulator.limits[PhasePreparation] = limit
	field, err := NewFactField(EvidenceFieldEventType, "runtime.event.large")
	require.NoError(t, err)
	fields := []FactField{field}
	fact := evidenceFact(t, "runtime.byte-fact", EvidenceSourceHistory, nil, fields)
	outcome, err := accumulator.append(PhasePreparation, fact)
	require.NoError(t, err)
	require.Equal(t, appendCapacity, outcome)
	require.Zero(t, accumulator.retainedCount())
}

func TestEvidenceAccumulatorRejectsIllTypedOrUnboundAllowlistedFields(t *testing.T) {
	request := newEngineRequest(t)
	correlations := request.Correlations()
	wrongCorrelation := correlations[0].Identity() + ".forged"
	tests := []struct {
		name         string
		definitionID string
		value        string
	}{
		{name: "command outside closed set", definitionID: EvidenceFieldCommandKind, value: "arbitrary"},
		{name: "status outside closed set", definitionID: EvidenceFieldStatus, value: "arbitrary"},
		{name: "error code outside closed set", definitionID: EvidenceFieldErrorCode, value: "secret.token"},
		{name: "event ID is not numeric", definitionID: EvidenceFieldEventID, value: "event.one"},
		{name: "run correlation is not request bound", definitionID: EvidenceFieldRunCorrelationID, value: "runtime.run.forged"},
		{name: "workflow correlation is not request bound", definitionID: EvidenceFieldWorkflowCorrelationID, value: wrongCorrelation},
		{name: "operation correlation is not request bound", definitionID: EvidenceFieldOperationCorrelationID, value: wrongCorrelation},
		{name: "namespace identity is not hashed", definitionID: EvidenceFieldNamespaceIdentity, value: "namespace.secret"},
		{name: "endpoint identity is not hashed", definitionID: EvidenceFieldEndpointIdentity, value: "endpoint.secret"},
		{name: "task queue identity is not hashed", definitionID: EvidenceFieldTaskQueueIdentity, value: "taskqueue.secret"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field, err := NewFactField(test.definitionID, test.value)
			require.NoError(t, err)
			accumulator := newEvidenceAccumulator(request)
			outcome, err := accumulator.append(
				PhaseObservation,
				evidenceFact(t, "runtime.fact.invalid-field", EvidenceSourceHistory, nil, []FactField{field}),
			)
			require.Error(t, err)
			require.Equal(t, appendRejected, outcome)
			require.Zero(t, accumulator.retainedCount())
		})
	}
}

func TestEvidenceAccumulatorRetainsRequestBindingsAndHashedSensitiveValues(t *testing.T) {
	request := newEngineRequest(t)
	correlations := request.Correlations()
	values := map[string]string{
		EvidenceFieldCommandKind:            string(CommandObserve),
		EvidenceFieldEndpointIdentity:       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EvidenceFieldErrorCode:              "umpire.runtime.code.failed",
		EvidenceFieldEventID:                "42",
		EvidenceFieldNamespaceIdentity:      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		EvidenceFieldOperationCorrelationID: correlationIdentity(correlations, CorrelationOperation),
		EvidenceFieldRunCorrelationID:       request.RunIdentity(),
		EvidenceFieldStatus:                 "succeeded",
		EvidenceFieldTaskQueueIdentity:      "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		EvidenceFieldWorkflowCorrelationID:  correlationIdentity(correlations, CorrelationWorkflow),
	}
	fieldIDs := make([]string, 0, len(values))
	for definitionID := range values {
		fieldIDs = append(fieldIDs, definitionID)
	}
	slices.Sort(fieldIDs)
	fields := make([]FactField, 0, len(fieldIDs))
	for _, definitionID := range fieldIDs {
		field, err := NewFactField(definitionID, values[definitionID])
		require.NoError(t, err)
		fields = append(fields, field)
	}
	accumulator := newEvidenceAccumulator(request)
	outcome, err := accumulator.append(
		PhaseObservation,
		evidenceFact(t, "runtime.fact.bound-fields", EvidenceSourceHistory, nil, fields),
	)
	require.NoError(t, err)
	require.Equal(t, appendRetained, outcome)
	_, facts, _ := accumulator.materialize()
	require.Len(t, facts, 1)
	require.Equal(t, "plain", rawFieldDisposition(facts[0].Fields, EvidenceFieldRunCorrelationID))
	for _, definitionID := range []string{
		EvidenceFieldEndpointIdentity,
		EvidenceFieldNamespaceIdentity,
		EvidenceFieldTaskQueueIdentity,
	} {
		require.Equal(t, "sha256", rawFieldDisposition(facts[0].Fields, definitionID))
	}
}

func appendEvidenceCapacity(
	t *testing.T,
	accumulator *evidenceAccumulator,
	sourceDefinitionID string,
	extra int,
) {
	t.Helper()
	index := 0
	for _, limit := range CanonicalPhaseLimits() {
		count := int(limit.MaxRecords())
		if limit.Phase() == PhaseCleanup {
			count += extra
		}
		for range count {
			fact := evidenceFact(t, fmt.Sprintf("runtime.fact.%04d", index), sourceDefinitionID, nil, nil)
			var outcome appendOutcome
			var err error
			if sourceDefinitionID == EvidenceSourceControlReceipt {
				outcome, err = accumulator.appendControlReceipt(limit.Phase(), fact)
			} else {
				outcome, err = accumulator.append(limit.Phase(), fact)
			}
			require.NoError(t, err)
			if index < 4096 {
				require.Equal(t, appendRetained, outcome)
			} else {
				require.Equal(t, appendCapacity, outcome)
			}
			index++
		}
	}
}

func evidenceFact(
	t *testing.T,
	definitionID string,
	sourceDefinitionID string,
	causes []string,
	fields []FactField,
) Fact {
	t.Helper()
	if causes == nil {
		causes = []string{}
	}
	if fields == nil {
		fields = []FactField{}
	}
	fact, err := NewFact(
		definitionID,
		sourceDefinitionID,
		"umpire.evidence.kind.mechanical",
		causes,
		fields,
	)
	require.NoError(t, err)
	return fact
}

func sourceStatusFromEvidence(sources []artifactv2.RawEvidenceSource, source string) string {
	for _, candidate := range sources {
		if candidate.SourceDefinitionID == source {
			return candidate.Status
		}
	}
	return ""
}

func newTestEvidenceAccumulator(t *testing.T) *evidenceAccumulator {
	t.Helper()
	return newEvidenceAccumulator(newEngineRequest(t))
}

func correlationIdentity(correlations []Correlation, kind CorrelationKind) string {
	for _, correlation := range correlations {
		if correlation.Kind() == kind {
			return correlation.Identity()
		}
	}
	return ""
}

func rawFieldDisposition(fields []artifactv2.RawEvidenceField, definitionID string) string {
	for _, field := range fields {
		if field.FieldDefinitionID == definitionID {
			return field.Disposition
		}
	}
	return ""
}
