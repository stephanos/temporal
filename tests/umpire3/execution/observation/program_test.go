package observation

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
)

func TestGeneratedNexusObservationProgramsEvaluateFourValues(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	cancellationWon, ok := catalog.Program(protocolcatalog.ObservationIDCancellationWon)
	require.True(t, ok)
	staleSuccessAbsent, ok := catalog.Program(protocolcatalog.ObservationIDStaleSuccessAbsent)
	require.True(t, ok)

	cancelled := historyFact("history/cancelled", "nexus-cancellation-committed", 2)
	closed := windowFact("window/closed", true, 4)
	stale := historyFact("history/stale-success", "nexus-success-recorded", 3)
	stale.History.OwnerEpoch = int64Pointer(1)
	stale.History.CurrentOwnerEpoch = int64Pointer(2)
	stale.History.CancellationCommitted = boolPointer(true)

	require.Equal(t, Evaluation{Value: True, Support: []string{"history/cancelled"}},
		cancellationWon.Evaluate([]Fact{cancelled}))
	require.Equal(t, Evaluation{Value: Unknown}, staleSuccessAbsent.Evaluate([]Fact{cancelled}))
	require.Equal(t, Evaluation{Value: True, Support: []string{"window/closed"}},
		staleSuccessAbsent.Evaluate([]Fact{cancelled, closed}))
	require.Equal(t, Evaluation{Value: False, Support: []string{"history/stale-success"}},
		staleSuccessAbsent.Evaluate([]Fact{cancelled, stale, closed}))

	conflicting := windowFact("window/closed", false, 4)
	require.Equal(t, Conflict,
		staleSuccessAbsent.Evaluate([]Fact{closed, conflicting}).Value)
}

func TestGeneratedObservationFixturesDetectWrongMapping(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	require.NotEmpty(t, catalog.Fixtures)

	program, ok := catalog.Program(protocolcatalog.ObservationIDStaleSuccessAbsent)
	require.True(t, ok)
	program.Violations = []Selector{{
		FactType: FactTypeHistoryEvent,
		Kind:     NexusCancellationCommitted,
	}}
	catalog.Programs[programIndex(t, catalog, program.Observation)] = program

	require.ErrorContains(t, catalog.Validate(), "fixture")
}

func TestObservationCatalogRequiresProgramForEverySemanticObservation(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	missing := protocolcatalog.ObservationIDUpdateAccepted
	catalog.Programs = slices.DeleteFunc(catalog.Programs, func(program Program) bool {
		return program.Observation == missing
	})
	catalog.Fixtures = slices.DeleteFunc(catalog.Fixtures, func(fixture Fixture) bool {
		return fixture.Observation == missing
	})

	require.ErrorContains(t, catalog.Validate(), `semantic observation "update-accepted" has no program`)
}

func TestAllExistAbsentWhenClosedRequiresCompletePositiveEvidence(t *testing.T) {
	program := Program{
		Observation: "complete-cancellation",
		Operation:   OperationAllExistAbsentWhenClosed,
		Matches: []Selector{
			{FactType: FactTypeHistoryEvent, Kind: NexusCancellationAccepted},
			{FactType: FactTypeHistoryEvent, Kind: NexusCancellationCommitted},
		},
		Violations: []Selector{{FactType: FactTypeHistoryEvent, Kind: NexusSuccessRecorded}},
		Closures: []Selector{{
			FactType: FactTypeEvidenceWindow, Kind: NexusCancellationWindow, Closed: boolPointer(true),
		}},
	}
	accepted := historyFact("history/accepted", NexusCancellationAccepted, 1)
	committed := historyFact("history/committed", NexusCancellationCommitted, 2)
	closed := windowFact("window/closed", true, 4)
	violation := historyFact("history/violation", NexusSuccessRecorded, 3)
	violation.History.OwnerEpoch = int64Pointer(1)
	violation.History.CurrentOwnerEpoch = int64Pointer(1)
	violation.History.CancellationCommitted = boolPointer(false)

	require.Equal(t, Evaluation{Value: Unknown}, program.Evaluate([]Fact{accepted, closed}))
	require.Equal(t, Evaluation{Value: Unknown}, program.Evaluate([]Fact{accepted, committed}))
	require.Equal(t, Evaluation{
		Value: True, Support: []string{"history/accepted", "history/committed", "window/closed"},
	}, program.Evaluate([]Fact{accepted, committed, closed}))
	require.Equal(t, Evaluation{Value: False, Support: []string{"history/violation"}},
		program.Evaluate([]Fact{accepted, committed, violation, closed}))
}

func TestFactValidationRejectsUntypedOrUnboundedEvidence(t *testing.T) {
	fact := historyFact("history/stale-success", "nexus-success-recorded", 3)
	fact.History.OwnerEpoch = int64Pointer(1)
	require.ErrorContains(t, fact.Validate(), "current owner epoch")

	fact = historyFact("history/cancelled", "nexus-cancellation-committed", 2)
	fact.Window = &EvidenceWindow{Purpose: "nexus-cancellation", Closed: true, ThroughSequence: 2}
	require.ErrorContains(t, fact.Validate(), "exactly one")
}

func historyFact(identifier, eventType string, sequence int64) Fact {
	return Fact{
		Identifier: identifier,
		Source: Source{
			Identity: "history/source", ClockDomain: "history/sequence", Sequence: sequence,
			Reference: "operation/1/" + identifier, EntityIdentity: "operation/1",
			Lineage: []string{"namespace/1", "operation/1"},
		},
		History: &HistoryEvent{EventType: eventType, EventID: sequence, OperationID: "operation/1"},
	}
}

func windowFact(identifier string, closed bool, sequence int64) Fact {
	return Fact{
		Identifier: identifier,
		Source: Source{
			Identity: "history/source", ClockDomain: "history/sequence", Sequence: sequence,
			Reference: "operation/1/" + identifier, EntityIdentity: "operation/1",
			Lineage: []string{"namespace/1", "operation/1"},
		},
		Window: &EvidenceWindow{Purpose: "nexus-cancellation", Closed: closed, ThroughSequence: sequence},
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func programIndex(t *testing.T, catalog Catalog, observation protocolcatalog.ObservationID) int {
	t.Helper()
	for index, program := range catalog.Programs {
		if program.Observation == observation {
			return index
		}
	}
	require.FailNow(t, "program is missing", observation)
	return -1
}
