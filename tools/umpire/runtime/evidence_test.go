package runtime

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestEvidenceAccumulatorRetainsExactlyNAndReportsNPlusOneBeforeAppend(t *testing.T) {
	atN := newEvidenceAccumulator(CanonicalPhaseLimits())
	appendEvidenceCapacity(t, atN, 0)
	for _, source := range evidenceSourceOrder {
		require.NoError(t, atN.closeSource(source, "closed"))
	}
	sources, facts, gaps := atN.materialize()
	require.Len(t, facts, 4096)
	require.Empty(t, gaps)
	require.Equal(t, "closed", sourceStatusFromEvidence(sources, EvidenceSourceHistory))

	atNPlusOne := newEvidenceAccumulator(CanonicalPhaseLimits())
	appendEvidenceCapacity(t, atNPlusOne, 1)
	for _, source := range evidenceSourceOrder {
		require.NoError(t, atNPlusOne.closeSource(source, "closed"))
	}
	sources, facts, gaps = atNPlusOne.materialize()
	require.Len(t, facts, 4096)
	require.Equal(t, "partial", sourceStatusFromEvidence(sources, EvidenceSourceHistory))
	require.Len(t, gaps, 1)
	require.Equal(t, "umpire.evidence.gap.capacity", gaps[0].Code)
	require.Equal(t, EvidenceSourceHistory, *gaps[0].Subject)
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
			accumulator := newEvidenceAccumulator(CanonicalPhaseLimits())
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
	accumulator := newEvidenceAccumulator(CanonicalPhaseLimits())
	fieldIDs := []string{
		EvidenceFieldCommandKind,
		EvidenceFieldErrorCode,
		EvidenceFieldEventID,
		EvidenceFieldEventType,
		EvidenceFieldOperationCorrelationID,
		EvidenceFieldRunCorrelationID,
		EvidenceFieldStatus,
		EvidenceFieldWorkflowCorrelationID,
	}
	slices.Sort(fieldIDs)
	fields := make([]FactField, len(fieldIDs))
	for index, fieldID := range fieldIDs {
		field, err := NewFactField(fieldID, strings.Repeat("x", MaximumFactValueBytes))
		require.NoError(t, err)
		fields[index] = field
	}
	for index := 0; ; index++ {
		fact := evidenceFact(t, fmt.Sprintf("runtime.byte-fact.%04d", index), EvidenceSourceHistory, nil, fields)
		before := accumulator.retainedCount()
		outcome, err := accumulator.append(PhasePreparation, fact)
		require.NoError(t, err)
		if outcome == appendCapacity {
			require.Equal(t, before, accumulator.retainedCount())
			require.Less(t, before, uint64(128))
			break
		}
		require.Equal(t, appendRetained, outcome)
	}
}

func appendEvidenceCapacity(t *testing.T, accumulator *evidenceAccumulator, extra int) {
	t.Helper()
	index := 0
	for _, limit := range CanonicalPhaseLimits() {
		count := int(limit.MaxRecords())
		if limit.Phase() == PhaseCleanup {
			count += extra
		}
		for range count {
			fact := evidenceFact(t, fmt.Sprintf("runtime.fact.%04d", index), EvidenceSourceHistory, nil, nil)
			outcome, err := accumulator.append(limit.Phase(), fact)
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
