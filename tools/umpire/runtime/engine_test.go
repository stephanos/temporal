package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

type oracleTerminal string

const (
	oracleSucceeded oracleTerminal = "succeeded"
	oracleFailed    oracleTerminal = "failed"
	oracleTimedOut  oracleTerminal = "timed-out"
	oracleCanceled  oracleTerminal = "canceled"
)

type oracleRow struct {
	name              string
	target            Phase
	terminal          oracleTerminal
	cleanupFailure    bool
	phases            [5]string
	control           string
	operational       string
	historySource     string
	controlSource     string
	cleanupSource     string
	capture           string
	cleanupStatus     string
	cleanupOpenHandle uint64
}

var exhaustiveOracle = []oracleRow{
	{name: "success", terminal: oracleSucceeded, phases: [5]string{"succeeded", "succeeded", "succeeded", "succeeded", "succeeded"}, control: "accepted", operational: "succeeded", historySource: "closed", controlSource: "closed", cleanupSource: "closed", capture: "closed", cleanupStatus: "complete"},
	{name: "preparation failed", target: PhasePreparation, terminal: oracleFailed, phases: [5]string{"failed", "not-started", "not-started", "succeeded", "succeeded"}, control: "not-attempted", operational: "failed", historySource: "partial", controlSource: "partial", cleanupSource: "closed", capture: "partial", cleanupStatus: "complete"},
	{name: "preparation timed out", target: PhasePreparation, terminal: oracleTimedOut, phases: [5]string{"timed-out", "not-started", "not-started", "succeeded", "succeeded"}, control: "not-attempted", operational: "incomplete", historySource: "partial", controlSource: "partial", cleanupSource: "closed", capture: "partial", cleanupStatus: "complete"},
	{name: "preparation canceled", target: PhasePreparation, terminal: oracleCanceled, phases: [5]string{"canceled", "not-started", "not-started", "succeeded", "succeeded"}, control: "not-attempted", operational: "incomplete", historySource: "partial", controlSource: "partial", cleanupSource: "closed", capture: "partial", cleanupStatus: "complete"},
	{name: "realization failed", target: PhaseRealization, terminal: oracleFailed, phases: [5]string{"succeeded", "failed", "succeeded", "succeeded", "succeeded"}, control: "failed", operational: "failed", historySource: "closed", controlSource: "closed", cleanupSource: "closed", capture: "closed", cleanupStatus: "complete"},
	{name: "realization timed out", target: PhaseRealization, terminal: oracleTimedOut, phases: [5]string{"succeeded", "timed-out", "succeeded", "succeeded", "succeeded"}, control: "canceled", operational: "incomplete", historySource: "closed", controlSource: "closed", cleanupSource: "closed", capture: "closed", cleanupStatus: "complete"},
	{name: "realization canceled", target: PhaseRealization, terminal: oracleCanceled, phases: [5]string{"succeeded", "canceled", "succeeded", "succeeded", "succeeded"}, control: "canceled", operational: "incomplete", historySource: "closed", controlSource: "closed", cleanupSource: "closed", capture: "closed", cleanupStatus: "complete"},
	{name: "observation failed", target: PhaseObservation, terminal: oracleFailed, phases: [5]string{"succeeded", "succeeded", "failed", "succeeded", "succeeded"}, control: "accepted", operational: "failed", historySource: "failed", controlSource: "closed", cleanupSource: "closed", capture: "failed", cleanupStatus: "complete"},
	{name: "observation timed out", target: PhaseObservation, terminal: oracleTimedOut, phases: [5]string{"succeeded", "succeeded", "timed-out", "succeeded", "succeeded"}, control: "accepted", operational: "incomplete", historySource: "partial", controlSource: "closed", cleanupSource: "closed", capture: "partial", cleanupStatus: "complete"},
	{name: "observation canceled", target: PhaseObservation, terminal: oracleCanceled, phases: [5]string{"succeeded", "succeeded", "canceled", "succeeded", "succeeded"}, control: "accepted", operational: "incomplete", historySource: "partial", controlSource: "closed", cleanupSource: "closed", capture: "partial", cleanupStatus: "complete"},
	{name: "isolation failed", target: PhaseIsolation, terminal: oracleFailed, phases: [5]string{"succeeded", "succeeded", "succeeded", "failed", "succeeded"}, control: "accepted", operational: "failed", historySource: "closed", controlSource: "closed", cleanupSource: "closed", capture: "closed", cleanupStatus: "complete"},
	{name: "isolation timed out", target: PhaseIsolation, terminal: oracleTimedOut, phases: [5]string{"succeeded", "succeeded", "succeeded", "timed-out", "succeeded"}, control: "accepted", operational: "incomplete", historySource: "closed", controlSource: "closed", cleanupSource: "closed", capture: "closed", cleanupStatus: "complete"},
	{name: "isolation canceled", target: PhaseIsolation, terminal: oracleCanceled, phases: [5]string{"succeeded", "succeeded", "succeeded", "canceled", "succeeded"}, control: "accepted", operational: "incomplete", historySource: "closed", controlSource: "closed", cleanupSource: "closed", capture: "closed", cleanupStatus: "complete"},
	{name: "cleanup failed", target: PhaseCleanup, terminal: oracleFailed, phases: [5]string{"succeeded", "succeeded", "succeeded", "succeeded", "failed"}, control: "accepted", operational: "failed", historySource: "closed", controlSource: "closed", cleanupSource: "failed", capture: "failed", cleanupStatus: "failed", cleanupOpenHandle: 1},
	{name: "cleanup timed out", target: PhaseCleanup, terminal: oracleTimedOut, phases: [5]string{"succeeded", "succeeded", "succeeded", "succeeded", "timed-out"}, control: "accepted", operational: "incomplete", historySource: "closed", controlSource: "closed", cleanupSource: "partial", capture: "partial", cleanupStatus: "incomplete", cleanupOpenHandle: 1},
	{name: "cleanup canceled", target: PhaseCleanup, terminal: oracleCanceled, phases: [5]string{"succeeded", "succeeded", "succeeded", "succeeded", "canceled"}, control: "accepted", operational: "incomplete", historySource: "closed", controlSource: "closed", cleanupSource: "partial", capture: "partial", cleanupStatus: "incomplete", cleanupOpenHandle: 1},
}

func TestRunMatchesIndependentExhaustivePhaseOracle(t *testing.T) {
	request := newEngineRequest(t)
	for _, row := range exhaustiveOracle {
		t.Run(row.name, func(t *testing.T) {
			state := newOracleState(t, row.target, row.terminal, row.cleanupFailure)
			output, err := runWithPhaseContexts(
				context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
			)
			require.NoError(t, err)
			run := output.ExperimentRun()
			rawEvidence := output.RawEvidence()
			require.Equal(t, row.operational, run.OperationalStatus)
			require.Equal(t, row.phases[:], phaseStatuses(run.PhaseOutcomes))
			require.Equal(t, row.control, run.ControlAttempts[0].Status)
			require.Equal(t, row.historySource, sourceStatus(run.SourceClosures, EvidenceSourceHistory))
			require.Equal(t, row.controlSource, sourceStatus(run.SourceClosures, EvidenceSourceControlReceipt))
			require.Equal(t, row.cleanupSource, sourceStatus(run.SourceClosures, EvidenceSourceCleanup))
			require.Equal(t, "closed", sourceStatus(run.SourceClosures, EvidenceSourceParticipantOutput))
			require.Equal(t, row.cleanupStatus, run.Cleanup.Status)
			require.Equal(t, artifactv2.NaturalFromUint64(row.cleanupOpenHandle), run.Cleanup.OpenHandleCount)
			require.Equal(t, row.capture, rawEvidence.CaptureStatus)
			require.Equal(t, run.RunIdentity, rawEvidence.RunIdentity)
			require.Equal(t, run.ArtifactChecksum, rawEvidence.Run.ArtifactChecksum)
			require.Equal(t, run.KnownGaps, rawEvidence.KnownGaps)
			require.Equal(t, sourceStatus(run.SourceClosures, EvidenceSourceCleanup), sourceStatusFromEvidence(rawEvidence.Sources, EvidenceSourceCleanup))
			require.Equal(t, sourceStatus(run.SourceClosures, EvidenceSourceControlReceipt), sourceStatusFromEvidence(rawEvidence.Sources, EvidenceSourceControlReceipt))
			require.Equal(t, sourceStatus(run.SourceClosures, EvidenceSourceHistory), sourceStatusFromEvidence(rawEvidence.Sources, EvidenceSourceHistory))
			require.Equal(t, sourceStatus(run.SourceClosures, EvidenceSourceParticipantOutput), sourceStatusFromEvidence(rawEvidence.Sources, EvidenceSourceParticipantOutput))
			expectedFactCount := 1
			if row.control == "not-attempted" {
				expectedFactCount = 0
			}
			require.Len(t, rawEvidence.Facts, expectedFactCount)
			require.Equal(t, 1, state.participant.cleanupCalls)
			require.Equal(t, 1, state.environment.cleanupCalls)
			require.Equal(t, 1, state.environment.isolationCalls)
			requireExactExecutionSet(t, output.AdmittedSet())
		})
	}
}

func TestRunCleanupFailureDominatesEveryEarlierOutcome(t *testing.T) {
	request := newEngineRequest(t)
	for _, row := range exhaustiveOracle[1:13] {
		t.Run(row.name, func(t *testing.T) {
			state := newOracleState(t, row.target, row.terminal, true)
			output, err := runWithPhaseContexts(
				context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
			)
			require.NoError(t, err)
			run := output.ExperimentRun()
			require.Equal(t, "failed", run.OperationalStatus)
			require.Equal(t, "failed", run.PhaseOutcomes[4].Status)
			require.Equal(t, "failed", run.Cleanup.Status)
			require.Equal(t, "failed", sourceStatus(run.SourceClosures, EvidenceSourceCleanup))
			require.Equal(t, 1, state.participant.cleanupCalls)
			require.Equal(t, 1, state.environment.cleanupCalls)
		})
	}
}

func TestRunConcreteCleanupFailureDominatesItsExpiredDeadline(t *testing.T) {
	request := newEngineRequest(t)
	state := newOracleState(t, "", oracleSucceeded, false)
	state.deadlineCleanupFailure = true
	output, err := runWithPhaseContexts(
		context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
	)
	require.NoError(t, err)
	run := output.ExperimentRun()
	require.Equal(t, "failed", run.PhaseOutcomes[4].Status)
	require.Equal(t, "failed", run.OperationalStatus)
	require.Equal(t, "failed", run.Cleanup.Status)
	require.Equal(t, "failed", sourceStatus(run.SourceClosures, EvidenceSourceCleanup))
}

func TestRunStopsPreparationWhenFactoryContextIsTerminal(t *testing.T) {
	request := newEngineRequest(t)
	for _, test := range []struct {
		name          string
		context       oracleTerminal
		receiptStatus ReceiptStatus
		expected      string
	}{
		{name: "accepted after deadline", context: oracleTimedOut, receiptStatus: ReceiptAccepted, expected: "timed-out"},
		{name: "accepted after cancellation", context: oracleCanceled, receiptStatus: ReceiptAccepted, expected: "canceled"},
		{name: "failure dominates deadline", context: oracleTimedOut, receiptStatus: ReceiptFailed, expected: "failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newOracleState(t, "", oracleSucceeded, false)
			state.factoryContextTerminal = test.context
			state.factoryReceiptStatus = test.receiptStatus
			output, err := runWithPhaseContexts(
				context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
			)
			require.NoError(t, err)
			run := output.ExperimentRun()
			require.Equal(t, test.expected, run.PhaseOutcomes[0].Status)
			require.Equal(t, 0, state.participant.prepareCalls)
			require.Equal(t, 0, state.participant.cleanupCalls)
			require.Equal(t, 1, state.environment.isolationCalls)
			require.Equal(t, 1, state.environment.cleanupCalls)
			requireExactExecutionSet(t, output.AdmittedSet())
		})
	}
}

func TestRunRecordsFactoryPreparationReceiptOutcomes(t *testing.T) {
	request := newEngineRequest(t)
	for _, test := range []struct {
		status   ReceiptStatus
		expected string
	}{
		{status: ReceiptFailed, expected: "failed"},
		{status: ReceiptRejected, expected: "failed"},
		{status: ReceiptUnsupported, expected: "failed"},
		{status: ReceiptCanceled, expected: "canceled"},
	} {
		t.Run(string(test.status), func(t *testing.T) {
			state := newOracleState(t, "", oracleSucceeded, false)
			state.factoryReceiptStatus = test.status
			output, err := runWithPhaseContexts(
				context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
			)
			require.NoError(t, err)
			run := output.ExperimentRun()
			require.Equal(t, test.expected, run.PhaseOutcomes[0].Status)
			require.Equal(t, "not-attempted", run.ControlAttempts[0].Status)
			require.Equal(t, 0, state.participant.prepareCalls)
			require.Equal(t, 1, state.environment.cleanupCalls)
			requireExactExecutionSet(t, output.AdmittedSet())
		})
	}
}

func TestRunRecordsParticipantPreparationReceiptOutcomes(t *testing.T) {
	request := newEngineRequest(t)
	for _, status := range []ReceiptStatus{ReceiptRejected, ReceiptUnsupported} {
		t.Run(string(status), func(t *testing.T) {
			state := newOracleState(t, "", oracleSucceeded, false)
			state.forcedReceiptStatuses = map[Phase]ReceiptStatus{PhasePreparation: status}
			output, err := runWithPhaseContexts(
				context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
			)
			require.NoError(t, err)
			run := output.ExperimentRun()
			require.Equal(t, "failed", run.PhaseOutcomes[0].Status)
			require.Equal(t, "not-attempted", run.ControlAttempts[0].Status)
			require.Equal(t, 1, state.participant.prepareCalls)
			require.Equal(t, 1, state.participant.cleanupCalls)
			require.Equal(t, 1, state.environment.cleanupCalls)
			requireExactExecutionSet(t, output.AdmittedSet())
		})
	}
}

func TestRunRecordsRejectedAndUnsupportedControlReceipts(t *testing.T) {
	request := newEngineRequest(t)
	for _, status := range []ReceiptStatus{ReceiptRejected, ReceiptUnsupported} {
		t.Run(string(status), func(t *testing.T) {
			state := newOracleState(t, "", oracleSucceeded, false)
			state.forcedReceiptStatuses = map[Phase]ReceiptStatus{PhaseRealization: status}
			output, err := runWithPhaseContexts(
				context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
			)
			require.NoError(t, err)
			run := output.ExperimentRun()
			require.Equal(t, "failed", run.PhaseOutcomes[1].Status)
			require.Equal(t, string(status), run.ControlAttempts[0].Status)
			require.Equal(t, "failed", run.OperationalStatus)
			require.Equal(t, 1, state.participant.cleanupCalls)
			require.Equal(t, 1, state.environment.cleanupCalls)
			requireExactExecutionSet(t, output.AdmittedSet())
		})
	}
}

func TestRunAppliesCompoundOutcomePrecedence(t *testing.T) {
	request := newEngineRequest(t)
	state := newOracleState(t, "", oracleSucceeded, false)
	state.phaseTerminals = map[Phase]oracleTerminal{
		PhaseObservation: oracleFailed,
		PhaseIsolation:   oracleTimedOut,
		PhaseCleanup:     oracleCanceled,
	}
	output, err := runWithPhaseContexts(
		context.Background(), request, state.factory, state.participant, oraclePhaseContextFactory{},
	)
	require.NoError(t, err)
	run := output.ExperimentRun()
	require.Equal(t, []string{"succeeded", "succeeded", "failed", "timed-out", "canceled"}, phaseStatuses(run.PhaseOutcomes))
	require.Equal(t, "failed", run.OperationalStatus)
	require.Equal(t, "failed", sourceStatus(run.SourceClosures, EvidenceSourceHistory))
	require.Equal(t, "partial", sourceStatus(run.SourceClosures, EvidenceSourceCleanup))
	require.Equal(t, 1, state.participant.cleanupCalls)
	require.Equal(t, 1, state.environment.cleanupCalls)
	requireExactExecutionSet(t, output.AdmittedSet())
}

func TestRunDetachesIsolationAndCleanupFromCanceledParent(t *testing.T) {
	request := newEngineRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	state := newOracleState(t, PhaseRealization, oracleCanceled, false)
	state.cancelParent = cancel
	output, err := Run(ctx, request, state.factory, state.participant)
	require.NoError(t, err)
	require.Equal(t, "canceled", output.ExperimentRun().PhaseOutcomes[1].Status)
	require.ErrorIs(t, state.participant.observationEntryError, context.Canceled)
	require.NoError(t, state.environment.isolationEntryError)
	require.NoError(t, state.participant.cleanupEntryError)
	require.NoError(t, state.environment.cleanupEntryError)
	require.Equal(t, 1, state.participant.cleanupCalls)
	require.Equal(t, 1, state.environment.cleanupCalls)
}

func TestRunRejectsMissingOrInvalidControlReceiptBeforeAdmission(t *testing.T) {
	request := newEngineRequest(t)
	state := newOracleState(t, "", oracleSucceeded, false)
	state.participant.invalidReceipt = true
	output, err := Run(context.Background(), request, state.factory, state.participant)
	var invariant *InvariantError
	require.ErrorAs(t, err, &invariant)
	require.True(t, invariant.ExecutionOccurred())
	require.Empty(t, output.AdmittedSet().Identity())
	require.Equal(t, 1, state.participant.cleanupCalls)
	require.Equal(t, 1, state.environment.cleanupCalls)
}

func TestRunRejectsDuplicateControlReceiptBeforeAdmission(t *testing.T) {
	request := newEngineRequest(t)
	state := newOracleState(t, "", oracleSucceeded, false)
	state.participant.duplicateControlReceipt = true
	output, err := Run(context.Background(), request, state.factory, state.participant)
	var invariant *InvariantError
	require.ErrorAs(t, err, &invariant)
	require.True(t, invariant.ExecutionOccurred())
	require.Empty(t, output.AdmittedSet().Identity())
	require.Equal(t, 1, state.participant.cleanupCalls)
	require.Equal(t, 1, state.environment.cleanupCalls)
}

func TestRunAdmitsExactEvidenceCapacityBoundary(t *testing.T) {
	request := newEngineRequest(t)
	for _, test := range []struct {
		name         string
		extraCleanup int
		operational  string
		capture      string
		gapCount     int
	}{
		{name: "N", operational: "succeeded", capture: "closed"},
		{name: "N plus one", extraCleanup: 1, operational: "incomplete", capture: "partial", gapCount: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newOracleState(t, "", oracleSucceeded, false)
			state.factCounts = map[Phase]int{
				PhasePreparation: 128,
				PhaseRealization: 127,
				PhaseObservation: 3584,
				PhaseIsolation:   128,
				PhaseCleanup:     128,
			}
			state.extraEnvironmentCleanupFacts = test.extraCleanup
			output, err := Run(context.Background(), request, state.factory, state.participant)
			require.NoError(t, err)
			run := output.ExperimentRun()
			rawEvidence := output.RawEvidence()
			require.Equal(t, test.operational, run.OperationalStatus)
			require.Equal(t, test.capture, rawEvidence.CaptureStatus)
			require.Len(t, rawEvidence.Facts, 4096)
			require.Len(t, run.KnownGaps, test.gapCount)
			require.Equal(t, run.KnownGaps, rawEvidence.KnownGaps)
			requireExactExecutionSet(t, output.AdmittedSet())
		})
	}
}

func phaseStatuses(outcomes []artifactv2.PhaseOutcome) []string {
	statuses := make([]string, len(outcomes))
	for index, outcome := range outcomes {
		statuses[index] = outcome.Status
	}
	return statuses
}

func sourceStatus(closures []artifactv2.SourceClosure, source string) string {
	for _, closure := range closures {
		if closure.SourceDefinitionID == source {
			return closure.Status
		}
	}
	return ""
}

func requireExactExecutionSet(t *testing.T, admitted artifact.AdmittedSet) {
	t.Helper()
	var manifest struct {
		Members []json.RawMessage `json:"members"`
	}
	require.NoError(t, json.Unmarshal(admitted.ManifestBytes(), &manifest))
	require.Len(t, manifest.Members, 4)
	_, executable := admitted.Executable()
	require.False(t, executable)
}

type oraclePhaseContext struct {
	context.Context
	mu   sync.Mutex
	err  error
	done chan struct{}
}

func (c *oraclePhaseContext) Done() <-chan struct{} { return c.done }

func (c *oraclePhaseContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *oraclePhaseContext) terminate(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.err = err
		close(c.done)
	}
}

type oraclePhaseContextFactory struct{}

func (oraclePhaseContextFactory) phaseContext(
	parent context.Context,
	_ Phase,
	_ PhaseLimit,
	_ bool,
	_ time.Time,
) (context.Context, context.CancelFunc) {
	controlled := &oraclePhaseContext{Context: parent, done: make(chan struct{})}
	return controlled, func() { controlled.terminate(context.Canceled) }
}

type oracleState struct {
	t                            *testing.T
	target                       Phase
	terminal                     oracleTerminal
	cleanupFailure               bool
	deadlineCleanupFailure       bool
	cancelParent                 context.CancelFunc
	factCounts                   map[Phase]int
	nextFact                     int
	extraEnvironmentCleanupFacts int
	factoryContextTerminal       oracleTerminal
	factoryReceiptStatus         ReceiptStatus
	forcedReceiptStatuses        map[Phase]ReceiptStatus
	phaseTerminals               map[Phase]oracleTerminal
	factory                      *oracleFactory
	environment                  *oracleEnvironment
	participant                  *oracleParticipant
}

func newOracleState(
	t *testing.T,
	target Phase,
	terminal oracleTerminal,
	cleanupFailure bool,
) *oracleState {
	state := &oracleState{t: t, target: target, terminal: terminal, cleanupFailure: cleanupFailure}
	state.environment = &oracleEnvironment{state: state}
	state.factory = &oracleFactory{state: state}
	state.participant = &oracleParticipant{state: state}
	return state
}

func (s *oracleState) receipt(
	ctx context.Context,
	command Command,
	phase Phase,
	acquired []Resource,
	released []Resource,
) Receipt {
	terminal := oracleSucceeded
	if s.target == phase {
		terminal = s.terminal
	}
	if forced, ok := s.phaseTerminals[phase]; ok {
		terminal = forced
	}
	if phase == PhaseCleanup && s.cleanupFailure {
		terminal = oracleFailed
	}
	if phase == PhaseCleanup && s.deadlineCleanupFailure {
		terminal = oracleFailed
		ctx.(*oraclePhaseContext).terminate(context.DeadlineExceeded)
	}
	status := ReceiptAccepted
	switch terminal {
	case oracleFailed:
		status = ReceiptFailed
	case oracleTimedOut:
		status = ReceiptCanceled
		ctx.(*oraclePhaseContext).terminate(context.DeadlineExceeded)
	case oracleCanceled:
		status = ReceiptCanceled
		if s.cancelParent != nil {
			s.cancelParent()
		} else {
			ctx.(*oraclePhaseContext).terminate(context.Canceled)
		}
	}
	if forced, ok := s.forcedReceiptStatuses[phase]; ok {
		status = forced
	}
	if status != ReceiptAccepted && phase == PhaseCleanup {
		released = []Resource{}
	}
	facts := s.takeFacts(phase, s.factCounts[phase])
	receipt, err := NewReceipt(command, status, facts, acquired, released)
	require.NoError(s.t, err)
	return receipt
}

func (s *oracleState) takeFacts(phase Phase, count int) []Fact {
	facts := make([]Fact, count)
	for index := range facts {
		fact, err := NewFact(
			fmt.Sprintf("runtime.fact.%04d", s.nextFact),
			EvidenceSourceHistory,
			"umpire.evidence.kind.mechanical",
			[]string{},
			[]FactField{},
		)
		require.NoError(s.t, err)
		facts[index] = fact
		s.nextFact++
	}
	return facts
}

type oracleFactory struct{ state *oracleState }

func (f *oracleFactory) Prepare(
	ctx context.Context,
	_ CheckedRunRequest,
	command Command,
) (Environment, Receipt) {
	if f.state.factoryContextTerminal == oracleTimedOut {
		ctx.(*oraclePhaseContext).terminate(context.DeadlineExceeded)
	}
	if f.state.factoryContextTerminal == oracleCanceled {
		ctx.(*oraclePhaseContext).terminate(context.Canceled)
	}
	status := f.state.factoryReceiptStatus
	if status == "" {
		status = ReceiptAccepted
	}
	resource := mustOracleResource(f.state.t, ResourceEnvironment, "runtime.resource.environment")
	receipt, err := NewReceipt(command, status, []Fact{}, []Resource{resource}, []Resource{})
	require.NoError(f.state.t, err)
	return f.state.environment, receipt
}

type oracleEnvironment struct {
	state               *oracleState
	isolationCalls      int
	cleanupCalls        int
	isolationEntryError error
	cleanupEntryError   error
}

func (e *oracleEnvironment) Isolate(ctx context.Context, command Command) Receipt {
	e.isolationCalls++
	e.isolationEntryError = ctx.Err()
	return e.state.receipt(ctx, command, PhaseIsolation, []Resource{}, []Resource{})
}

func (e *oracleEnvironment) Cleanup(ctx context.Context, command Command) Receipt {
	e.cleanupCalls++
	e.cleanupEntryError = ctx.Err()
	resource := mustOracleResource(e.state.t, ResourceEnvironment, "runtime.resource.environment")
	facts := e.state.takeFacts(PhaseCleanup, e.state.extraEnvironmentCleanupFacts)
	receipt, err := NewReceipt(command, ReceiptAccepted, facts, []Resource{}, []Resource{resource})
	require.NoError(e.state.t, err)
	return receipt
}

type oracleParticipant struct {
	state                   *oracleState
	prepareCalls            int
	cleanupCalls            int
	observationEntryError   error
	cleanupEntryError       error
	invalidReceipt          bool
	duplicateControlReceipt bool
}

func (p *oracleParticipant) Prepare(ctx context.Context, _ Environment, command Command) Receipt {
	p.prepareCalls++
	resource := mustOracleResource(p.state.t, ResourceParticipant, "runtime.resource.participant")
	return p.state.receipt(ctx, command, PhasePreparation, []Resource{resource}, []Resource{})
}

func (p *oracleParticipant) Realize(ctx context.Context, _ Environment, command Command) Receipt {
	if p.invalidReceipt {
		return Receipt{}
	}
	if p.duplicateControlReceipt {
		fact, err := controlReceiptFact(command, "accepted")
		require.NoError(p.state.t, err)
		receipt, err := NewReceipt(command, ReceiptAccepted, []Fact{fact}, []Resource{}, []Resource{})
		require.NoError(p.state.t, err)
		return receipt
	}
	return p.state.receipt(ctx, command, PhaseRealization, []Resource{}, []Resource{})
}

func (p *oracleParticipant) Observe(ctx context.Context, _ Environment, command Command) Receipt {
	p.observationEntryError = ctx.Err()
	return p.state.receipt(ctx, command, PhaseObservation, []Resource{}, []Resource{})
}

func (p *oracleParticipant) Cleanup(ctx context.Context, _ Environment, command Command) Receipt {
	p.cleanupCalls++
	p.cleanupEntryError = ctx.Err()
	resource := mustOracleResource(p.state.t, ResourceParticipant, "runtime.resource.participant")
	return p.state.receipt(ctx, command, PhaseCleanup, []Resource{}, []Resource{resource})
}

func mustOracleResource(t *testing.T, kind ResourceKind, identity string) Resource {
	t.Helper()
	resource, err := NewResource(kind, identity)
	require.NoError(t, err)
	return resource
}

func newEngineRequest(t *testing.T) CheckedRunRequest {
	t.Helper()
	experimentBytes := readEngineFixture(t, "SwitchExperimentSpecV2.json")
	runtimeConfigurationBytes := readEngineFixture(t, "RuntimeConfigurationV2.json")
	experiment, err := artifact.DecodeExperimentV2(experimentBytes)
	require.NoError(t, err)
	runtimeConfiguration, err := artifact.DecodeRuntimeConfigurationV2(runtimeConfigurationBytes)
	require.NoError(t, err)
	profileCapabilities := []string{
		"runtime.capability.complete-history-read",
		"runtime.capability.ephemeral-server-lifecycle",
		"runtime.capability.worker-lifecycle",
	}
	experiment.Plan.CapabilityRequirementDefinitionIDs = append(
		append([]string{}, profileCapabilities...), "switch.capability.state",
	)
	experiment, err = artifactv2.SealExperiment(experiment)
	require.NoError(t, err)
	experimentBytes, err = artifactv2.CanonicalExperimentBytes(experiment)
	require.NoError(t, err)
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	runtimeConfiguration.Experiment = experimentBinding
	runtimeConfiguration.AuthorityProfile = artifactv2.AuthorityProfile{
		DefinitionID: "runtime.profile.ephemeral-local", Version: artifactv2.Natural("2"),
		BehaviorFingerprint:             "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RequiredCapabilityDefinitionIDs: profileCapabilities,
	}
	runtimeConfiguration.ParticipantBindings = []artifactv2.ParticipantBinding{{
		ParticipantDefinitionID: "runtime.participant.sdk",
		ProtocolDefinitionID:    "runtime.protocol.v2", ProtocolVersion: artifactv2.Natural("2"),
		ProgramDefinitionID:        "runtime.program.single-control",
		ProgramBehaviorFingerprint: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CapabilityDefinitionIDs:    []string{"switch.capability.state"},
	}}
	runtimeConfiguration.KnownGaps = []artifactv2.KnownGap{}
	runtimeConfiguration, err = artifactv2.SealRuntimeConfiguration(runtimeConfiguration)
	require.NoError(t, err)
	runtimeConfigurationBytes, err = artifactv2.CanonicalRuntimeConfigurationBytes(runtimeConfiguration)
	require.NoError(t, err)
	set, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experimentBytes},
		{Path: "artifacts/runtime-configuration.json", Encoded: runtimeConfigurationBytes},
	})
	require.NoError(t, err)
	action := experiment.Plan.RequestedActions[0].DefinitionID
	occurrence, err := NewOccurrence(experiment.Plan.LinearExtension[0].DefinitionID, action, 1)
	require.NoError(t, err)
	program, err := NewProgram(
		"runtime.program.single-control", 2,
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		[]string{experiment.Plan.TargetDefinitionID}, []string{action}, []Occurrence{occurrence},
		[]string{"switch.capability.state"},
	)
	require.NoError(t, err)
	authority, err := NewAuthority(
		"runtime.profile.ephemeral-local", 2,
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"switch.runtime.configuration", runtimeConfiguration.BehaviorFingerprint,
		profileCapabilities, CanonicalPhaseLimits(), 0, 1,
		"runtime.participant.sdk", "runtime.protocol.v2", 2, 1, 1, program,
	)
	require.NoError(t, err)
	request, err := CheckRequest(set, authority, "runtime.run.oracle", 0, 1)
	require.NoError(t, err)
	return request
}

func readEngineFixture(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "model", "Umpire", "Artifact", "Tests", "Fixtures", name,
	))
	require.NoError(t, err)
	return encoded
}

func TestOracleRowsRemainUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(exhaustiveOracle))
	for _, row := range exhaustiveOracle {
		key := fmt.Sprintf("%s/%s", row.target, row.terminal)
		_, duplicate := seen[key]
		require.False(t, duplicate, row.name)
		seen[key] = struct{}{}
	}
}
