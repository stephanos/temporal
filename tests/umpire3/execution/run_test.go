package execution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
	"go.temporal.io/server/tests/umpire3/observation"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/scenario"
)

type fakeFactory struct {
	capabilities []protocol.CapabilityID
	session      Session
	prepareErr   error
	prepareCount int
}

type footprintFactory struct {
	fakeFactory
	report umpire3fault.Report
}

func TestActionRateWaitHonorsCancellation(t *testing.T) {
	previous := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, waitForActionRate(ctx, &previous, 1), context.Canceled)
}

func (f *footprintFactory) FootprintReport() (umpire3fault.Report, error) {
	return f.report, nil
}

type corroboratingFactFakeSession struct {
	*fakeSession
	facts         map[string][]observation.Fact
	corroborating map[string][][]observation.Fact
	err           error
}

type factFakeSession struct {
	*fakeSession
	facts map[string][]observation.Fact
}

func (s *factFakeSession) ObserveFacts(
	_ context.Context,
	checkpoint protocol.Checkpoint,
	_ Bindings,
) ([]observation.Fact, error) {
	return s.facts[checkpoint.Identifier], nil
}

func (s *corroboratingFactFakeSession) ObserveFacts(
	_ context.Context,
	checkpoint protocol.Checkpoint,
	_ Bindings,
) ([]observation.Fact, error) {
	return s.facts[checkpoint.Identifier], nil
}

func (s *corroboratingFactFakeSession) CorroborateFacts(
	_ context.Context,
	checkpoint protocol.Checkpoint,
	_ Bindings,
) ([][]observation.Fact, error) {
	return s.corroborating[checkpoint.Identifier], s.err
}

func (f *fakeFactory) Capabilities() []protocol.CapabilityID {
	return f.capabilities
}

func (f *fakeFactory) Prepare(context.Context, protocol.Experiment) (PreparedEnvironment, error) {
	f.prepareCount++
	return PreparedEnvironment{
		Session: f.session, Identity: testEnvironmentIdentity(f.capabilities),
	}, f.prepareErr
}

type fakeSession struct {
	mu             sync.Mutex
	realizeErr     map[string]error
	actionEvidence map[string]ActionEvidence
	groundings     map[string]map[string]string
	observations   map[string]Observation
	cleanup        CleanupResult
	realized       []string
	cleaned        bool
	cleanupCount   int
	faultRealizer  umpire3fault.Realizer
}

func (s *fakeSession) Realize(ctx context.Context, action protocol.Action, _ Bindings) (ActionEvidence, error) {
	select {
	case <-ctx.Done():
		return ActionEvidence{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realized = append(s.realized, action.Kind)
	evidence, exists := s.actionEvidence[action.Identifier]
	if exists {
		return evidence, s.realizeErr[action.Kind]
	}
	return ActionEvidence{
		Source: "fake", Outcome: protocol.ActionOutcomeApplied, Reference: action.Identifier,
		GroundedBindings: s.groundings[action.Identifier],
	}, s.realizeErr[action.Kind]
}

func (s *fakeSession) ObserveFacts(
	_ context.Context,
	checkpoint protocol.Checkpoint,
	_ Bindings,
) ([]observation.Fact, error) {
	observed, ok := s.observations[checkpoint.Identifier]
	if !ok {
		return nil, ErrObservationUnavailable
	}
	return fakeObservationFacts(checkpoint, observed), nil
}

func fakeObservationFacts(checkpoint protocol.Checkpoint, observed Observation) []observation.Fact {
	if !observed.Satisfied && checkpoint.Observation != "stale-success-absent" {
		return nil
	}
	causalReferences := append([]string(nil), observed.CausalReferences...)
	if observed.CausalReference != "" && !slices.Contains(causalReferences, observed.CausalReference) {
		causalReferences = append(causalReferences, observed.CausalReference)
	}
	sourceIdentity := observed.SourceIdentity
	if sourceIdentity == "" {
		sourceIdentity = observed.Source
	}
	fact := observation.Fact{
		Identifier: "fact/" + checkpoint.Identifier,
		Source: observation.Source{
			Identity: sourceIdentity, ClockDomain: observed.ClockDomain,
			Sequence: observed.SourceSequence, Reference: observed.Reference,
			CausalReferences: causalReferences, EntityIdentity: observed.EntityIdentity,
			Lineage: append([]string(nil), observed.Lineage...), PayloadDigest: observed.PayloadDigest,
		},
	}
	switch checkpoint.Observation {
	case "cancellation-accepted":
		fact.History = &observation.HistoryEvent{
			EventType: observation.NexusCancellationAccepted,
			EventID:   fact.Source.Sequence, OperationID: observed.EntityIdentity,
		}
	case "cancellation-won":
		fact.History = &observation.HistoryEvent{
			EventType: observation.NexusCancellationCommitted,
			EventID:   fact.Source.Sequence, OperationID: observed.EntityIdentity,
		}
	case "stale-success-absent":
		if observed.Satisfied {
			fact.Window = &observation.EvidenceWindow{
				Purpose: observation.NexusCancellationWindow,
				Closed:  true, ThroughSequence: fact.Source.Sequence,
			}
		} else {
			ownerEpoch, currentOwnerEpoch, cancellationCommitted := int64(1), int64(2), true
			fact.History = &observation.HistoryEvent{
				EventType: observation.NexusSuccessRecorded,
				EventID:   fact.Source.Sequence, OperationID: observed.EntityIdentity,
				OwnerEpoch: &ownerEpoch, CurrentOwnerEpoch: &currentOwnerEpoch,
				CancellationCommitted: &cancellationCommitted,
			}
		}
	case "update-accepted":
		fact.History = &observation.HistoryEvent{
			EventType: observation.WorkflowUpdateAccepted,
			EventID:   fact.Source.Sequence, WorkflowID: observed.EntityIdentity, RunID: "run",
		}
	case "update-completed":
		fact.History = &observation.HistoryEvent{
			EventType: observation.WorkflowUpdateCompleted,
			EventID:   fact.Source.Sequence, WorkflowID: observed.EntityIdentity, RunID: "run",
		}
	default:
		return nil
	}
	return []observation.Fact{fact}
}

func (s *fakeSession) Cleanup(context.Context) CleanupResult {
	s.cleaned = true
	s.cleanupCount++
	return s.cleanup
}

func (s *fakeSession) RecoveryMetadata() map[string]string {
	return map[string]string{"resource": "fake"}
}

func (s *fakeSession) FaultRealizer() umpire3fault.Realizer { return s.faultRealizer }

func TestRunRejectsUnsupportedCapabilitiesBeforePrepare(t *testing.T) {
	experiment := loadExperiment(t)
	factory := &fakeFactory{capabilities: []protocol.CapabilityID{protocol.CapabilityIDNexus}}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimUnsupported, result.Claim.Kind)
	require.Zero(t, factory.prepareCount)
}

func TestMissingCapabilitiesIncludesFaultRequirements(t *testing.T) {
	experiment := protocol.Experiment{Faults: []protocol.Fault{{
		RequiredCapabilities: []string{"failover-control"},
	}}}

	require.Equal(t, []string{"failover-control"}, missingCapabilities(experiment, nil))
}

func TestRunRejectsDeclaredFaultWithoutRealizer(t *testing.T) {
	experiment := loadExperiment(t)
	session := &factFakeSession{
		fakeSession: conformingSession(experiment),
		facts:       cancellationFacts(experiment, "public-history"),
	}
	session.faultRealizer = nil
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimUnsupported, result.Claim.Kind)
	require.ErrorContains(t, errors.New(result.Claim.Reason), "fault realizer")
	require.Empty(t, result.Actions)
	require.True(t, session.cleaned)
}

func TestRunRealizesFaultOverDeclaredActionInterval(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	realizer := &runtimeFaultRealizer{}
	factory := &faultFactory{
		fakeFactory: fakeFactory{capabilities: allCapabilities(experiment), session: session},
		realizer:    realizer,
	}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.Equal(t, protocol.ResultClassImplementationConforming, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, result.TrustBadge)
	require.Equal(t, []string{"install", "activate", "release", "cleanup"}, realizer.calls)
	require.Len(t, result.Faults, 1)
	require.True(t, result.Faults[0].Realized)
	require.True(t, result.Faults[0].Released)
	require.True(t, result.Faults[0].CleanupComplete)
	require.NotEmpty(t, result.Faults[0].Reference)
}

func TestResultAssuranceIsDerivedFromFinalClaim(t *testing.T) {
	for _, test := range []struct {
		claim         ClaimKind
		resultClass   protocol.ResultClass
		requiresTrace bool
	}{
		{claim: ClaimConforming, resultClass: protocol.ResultClassImplementationConforming},
		{claim: ClaimViolating, resultClass: protocol.ResultClassTraceWitness, requiresTrace: true},
		{claim: ClaimUnsupported, resultClass: protocol.ResultClassUnknown},
		{claim: ClaimInconclusive, resultClass: protocol.ResultClassUnknown},
		{claim: ClaimEvidenceFailure, resultClass: protocol.ResultClassUnknown},
	} {
		t.Run(string(test.claim), func(t *testing.T) {
			result := Result{Claim: Claim{Kind: test.claim}}
			finalizeAssurance(&result)
			require.Equal(t, test.resultClass, result.ResultClass)
			require.Equal(t, protocol.TrustBadgeTestedInstance, result.TrustBadge)
			if test.requiresTrace {
				require.ErrorContains(t, result.ValidateAssurance(), "semantic trace")
			} else {
				require.NoError(t, result.ValidateAssurance())
			}

			result.ResultClass = protocol.ResultClassFiniteExhaustive
			require.Error(t, result.ValidateAssurance())
		})
	}
}

func TestRunFaultCleanupFailureDowngradesConformance(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	realizer := &runtimeFaultRealizer{cleanupErr: errors.New("injected fault cleanup failure")}
	factory := &faultFactory{
		fakeFactory: fakeFactory{capabilities: allCapabilities(experiment), session: session},
		realizer:    realizer,
	}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.ErrorContains(t, errors.New(result.Faults[0].Error), "cleanup")
	require.False(t, result.Faults[0].CleanupComplete)
}

func TestRunPersistsRequiredLearnedFootprint(t *testing.T) {
	experiment := loadExperiment(t)
	factory := &footprintFactory{fakeFactory: fakeFactory{
		capabilities: allCapabilities(experiment), session: conformingSession(experiment),
	}, report: umpire3fault.Report{
		FormatVersion: umpire3fault.FootprintFormatVersion,
		Calls: []umpire3fault.Call{{
			Protocol: "http", Service: "nexus", Route: "/service/operation",
			Direction: umpire3fault.DirectionInbound, Role: umpire3fault.CallRoleInternal,
			Namespace: "namespace", Participant: "handler", Attempt: 1, Occurrence: 1,
			Interval: umpire3fault.Interval{Start: 1, Stop: 2},
		}},
		Declared: []umpire3fault.Footprint{{Protocol: "http", Service: "nexus", Route: "/service/operation"}},
	}}
	report, err := umpire3fault.BuildFootprintReport(factory.report.Declared, factory.report.Calls, nil)
	require.NoError(t, err)
	factory.report = report

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.NotNil(t, result.Footprint)
	require.True(t, result.Footprint.Complete)
	require.NotEmpty(t, result.Footprint.ReconciliationDigest)
}

func TestRunRejectsRequiredLearnedFootprintDrift(t *testing.T) {
	experiment := loadExperiment(t)
	report, err := umpire3fault.BuildFootprintReport(
		[]umpire3fault.Footprint{{Protocol: "grpc", Service: "matching", Route: "DispatchNexusTask"}},
		[]umpire3fault.Call{{
			Protocol: "http", Service: "nexus", Route: "/service/operation",
			Direction: umpire3fault.DirectionInbound, Role: umpire3fault.CallRoleInternal,
			Namespace: "namespace", Participant: "handler", Attempt: 1, Occurrence: 1,
			Interval: umpire3fault.Interval{Start: 1, Stop: 2},
		}}, nil)
	require.NoError(t, err)
	factory := &footprintFactory{fakeFactory: fakeFactory{
		capabilities: allCapabilities(experiment), session: conformingSession(experiment),
	}, report: report}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.ErrorContains(t, errors.New(result.Claim.Reason), "footprint reconciliation drift")
}

func TestRunConformsWithCompleteCausalEvidenceAndCleanup(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.Equal(t, 1, factory.prepareCount)
	require.Equal(t, 1, session.cleanupCount)
	require.True(t, session.cleaned)
	require.Len(t, result.Actions, len(experiment.Actions))
	require.Len(t, result.Observations, len(experiment.Checkpoints))
	require.True(t, result.Cleanup.Complete)
}

func TestRunRejectsMissingObservedActionOutcome(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	session.actionEvidence = map[string]ActionEvidence{
		experiment.Actions[0].Identifier: {
			Source: "fake", Reference: experiment.Actions[0].Identifier,
		},
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "observed action outcome is missing")
}

func TestRunRejectsAppliedOutcomeThatCanonicalModelCannotExecute(t *testing.T) {
	experiment := compiledNexusAttemptExperiment(t)
	session := conformingSession(experiment)
	session.actionEvidence = make(map[string]ActionEvidence, len(experiment.Actions))
	for _, action := range experiment.Actions {
		session.actionEvidence[action.Identifier] = ActionEvidence{
			Source: "fake", Reference: action.Identifier, Outcome: protocol.ActionOutcomeApplied,
		}
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, `canonical attempt replay rejects action "persist-success"`)
}

func TestRunAcceptsSuppressedOutcomeWithoutApplyingAbstractTransition(t *testing.T) {
	experiment := compiledNexusAttemptExperiment(t)
	session := conformingSession(experiment)
	session.actionEvidence = make(map[string]ActionEvidence, len(experiment.Actions))
	for _, action := range experiment.Actions {
		outcome := protocol.ActionOutcomeApplied
		if action.Kind == string(protocol.ActionKindPersistSuccess) {
			outcome = protocol.ActionOutcomeSuppressed
		}
		session.actionEvidence[action.Identifier] = ActionEvidence{
			Source: "fake", Reference: action.Identifier, Outcome: outcome,
		}
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
}

func TestRuntimeResultBindsStoredEvidenceDigest(t *testing.T) {
	experiment := loadExperiment(t)
	session := &factFakeSession{
		fakeSession: conformingSession(experiment),
		facts:       cancellationFacts(experiment, "public-history"),
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Regexp(t, `^sha256:[0-9a-f]{64}$`, result.EvidenceDigest)
	require.NoError(t, result.ValidateEvidenceDigest())

	originalReference := result.Facts[0].Source.Reference
	result.Facts[0].Source.Reference = "mutated-after-execution"
	require.ErrorContains(t, result.ValidateEvidenceDigest(), "does not match")
	result.Facts[0].Source.Reference = originalReference

	originalReason := result.Evidence.Claims[0].Reason
	result.Evidence.Claims[0].Reason = "mutated after execution"
	require.ErrorContains(t, result.ValidateEvidenceDigest(), "does not match")
	result.Evidence.Claims[0].Reason = originalReason

	result.Actions[0].Evidence.Outcome = protocol.ActionOutcomeSuppressed
	require.ErrorContains(t, result.ValidateEvidenceDigest(), "does not match")
}

func TestNormalizeEvidenceBuildsGraphWithoutDiscardingObservations(t *testing.T) {
	result := Result{
		Claim: Claim{Kind: ClaimViolating, Property: "property"},
		Observations: []Observation{{
			CheckpointID: "checkpoint", Kind: "kind", Satisfied: true,
			Source: "history", SourceIdentity: "history", ClockDomain: "history-sequence",
			SourceSequence: 1, ObservedAtUnixNano: 1, Reference: "history/1",
			EntityIdentity: "entity", Lineage: []string{"namespace", "entity"},
		}},
	}
	require.NoError(t, result.NormalizeEvidence(1<<20))
	require.Len(t, result.Observations, 1)
	require.Len(t, result.Evidence.Facts, 1)
	require.Len(t, result.Evidence.Claims, 1)
	require.NoError(t, result.ValidateEvidenceDigest())
}

func TestInterpretedObservationRetainsCompleteSupportBoundary(t *testing.T) {
	facts := make([]observation.Fact, 3)
	for index := range facts {
		sequence := int64(index + 1)
		facts[index] = observation.Fact{
			Identifier: []string{"pending", "completed", "progressed"}[index],
			Source: observation.Source{
				Identity: "history", ClockDomain: "history-sequence", Sequence: sequence,
				Reference:        "reference-" + []string{"1", "2", "3"}[index],
				CausalReferences: []string{"cause-" + []string{"1", "2", "3"}[index]},
				EntityIdentity:   "entity", Lineage: []string{"namespace", "entity"},
			},
		}
	}

	interpreted, err := interpretedObservation(protocol.Checkpoint{
		Identifier: "progress", Observation: "entity-progressed",
	}, observation.Evaluation{
		Value: observation.True, Support: []string{"pending", "completed", "progressed"},
	}, facts)
	require.NoError(t, err)
	require.Equal(t, int64(3), interpreted.SourceSequence)
	require.Equal(t, "reference-3", interpreted.Reference)
	require.Equal(t, "cause-3", interpreted.CausalReference)
	require.Equal(t, []string{"cause-1", "cause-2", "cause-3"}, interpreted.CausalReferences)
	require.Equal(t, []string{"pending", "completed", "progressed"}, interpreted.SupportingFacts)
}

func TestRunReportsAllowedFailureAsDegradedWithoutChangingClaim(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	last := experiment.Actions[len(experiment.Actions)-1]
	session.actionEvidence = map[string]ActionEvidence{
		last.Identifier: {
			Source: "fake", Outcome: protocol.ActionOutcomeApplied,
			SourceIdentity: "fake", Reference: "history/failed",
			EntityIdentity: "workflow/run", Lineage: []string{"workflow", "run"},
			TerminalState: "failed", TerminalDisposition: protocol.TerminalDispositionFailure,
		},
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.Equal(t, OutcomeDegraded, result.Outcome.Kind)
	require.Equal(t, "failed", result.Outcome.Terminal)
}

func TestRunReportsViolationAsFlaggedWithoutTerminal(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	observation := session.observations["no-stale-success"]
	observation.Satisfied = false
	session.observations["no-stale-success"] = observation
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimViolating, result.Claim.Kind)
	require.Equal(t, OutcomeFlagged, result.Outcome.Kind)
	require.NotNil(t, result.Trace)
	require.NoError(t, result.Trace.Validate())
	require.Equal(t, protocol.SemanticTraceProducerLive, result.Trace.Producer)
	require.Len(t, result.Trace.Steps, len(experiment.Actions))
	for _, step := range result.Trace.Steps {
		require.Equal(t, protocol.ActionOutcomeApplied, step.Outcome)
	}
}

func TestRunRetainsIndependentlyInterpretedCorroboratingFacts(t *testing.T) {
	experiment := loadExperiment(t)
	primaryFacts := cancellationFacts(experiment, "public-history")
	corroboratingFacts := cancellationFacts(experiment, "internal-history")
	corroborating := make(map[string][][]observation.Fact, len(corroboratingFacts))
	for identifier, facts := range corroboratingFacts {
		corroborating[identifier] = [][]observation.Fact{facts}
	}
	session := &corroboratingFactFakeSession{
		fakeSession:   conformingSession(experiment),
		facts:         primaryFacts,
		corroborating: corroborating,
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.Len(t, result.Observations, 2*len(experiment.Checkpoints))
	require.Contains(t, observationSources(result.Observations), "public-history")
	require.Contains(t, observationSources(result.Observations), "internal-history")
}

func TestRunRejectsPrimaryFactsWithConflictingLineage(t *testing.T) {
	experiment := loadExperiment(t)
	facts := cancellationFacts(experiment, "public-history")
	conflicting := facts["cancellation-accepted"][0]
	conflicting.Identifier = "public-history/conflicting-lineage"
	conflicting.Source.Lineage = []string{experiment.ExperimentID, "different-operation"}
	facts["cancellation-accepted"] = append(facts["cancellation-accepted"], conflicting)
	session := &factFakeSession{fakeSession: conformingSession(experiment), facts: facts}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "inconsistent identity")
}

func TestRunRejectsPrimaryFactsWithAmbiguousClockDomain(t *testing.T) {
	experiment := loadExperiment(t)
	facts := cancellationFacts(experiment, "public-history")
	ambiguous := facts["cancellation-accepted"][0]
	ambiguous.Identifier = "public-history/ambiguous-clock"
	ambiguous.Source.ClockDomain = "different-history-sequence"
	facts["cancellation-accepted"] = append(facts["cancellation-accepted"], ambiguous)
	session := &factFakeSession{fakeSession: conformingSession(experiment), facts: facts}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "inconsistent identity")
}

func TestRunRejectsPrimaryFactsWithWrongEntityIdentity(t *testing.T) {
	experiment := loadExperiment(t)
	facts := cancellationFacts(experiment, "public-history")
	wrongEntity := facts["cancellation-accepted"][0]
	wrongEntity.Identifier = "public-history/wrong-entity"
	wrongEntity.Source.EntityIdentity = "different-operation"
	facts["cancellation-accepted"] = append(facts["cancellation-accepted"], wrongEntity)
	session := &factFakeSession{fakeSession: conformingSession(experiment), facts: facts}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "conflicts")
}

func TestRunMissingAuthoritativeClosureIsInconclusive(t *testing.T) {
	experiment := loadExperiment(t)
	facts := cancellationFacts(experiment, "public-history")
	delete(facts, "no-stale-success")
	session := &factFakeSession{fakeSession: conformingSession(experiment), facts: facts}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "required evidence or window closure is missing")
}

func TestRunRejectsCausalFactWithoutPredecessorReference(t *testing.T) {
	experiment := loadExperiment(t)
	facts := cancellationFacts(experiment, "public-history")
	missingCausalReference := facts["cancellation-won"][0]
	missingCausalReference.Source.CausalReferences = nil
	facts["cancellation-won"] = []observation.Fact{missingCausalReference}
	session := &factFakeSession{fakeSession: conformingSession(experiment), facts: facts}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.Contains(t, result.Omissions[0], "causal reference")
}

func TestRunRejectsCorroboratingFactsWithConflictingLineage(t *testing.T) {
	experiment := loadExperiment(t)
	primaryFacts := cancellationFacts(experiment, "public-history")
	corroboratingFacts := cancellationFacts(experiment, "internal-history")
	conflicting := corroboratingFacts["cancellation-accepted"][0]
	conflicting.Identifier = "internal-history/conflicting-lineage"
	conflicting.Source.Lineage = []string{experiment.ExperimentID, "different-operation"}
	corroboratingFacts["cancellation-accepted"] = append(
		corroboratingFacts["cancellation-accepted"], conflicting)
	corroborating := make(map[string][][]observation.Fact, len(corroboratingFacts))
	for identifier, facts := range corroboratingFacts {
		corroborating[identifier] = [][]observation.Fact{facts}
	}
	session := &corroboratingFactFakeSession{
		fakeSession:   conformingSession(experiment),
		facts:         primaryFacts,
		corroborating: corroborating,
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "inconsistent identity")
}

func TestRunRejectsDisagreeingCorroboratingFacts(t *testing.T) {
	experiment := loadExperiment(t)
	primaryFacts := cancellationFacts(experiment, "public-history")
	corroboratingFacts := cancellationFacts(experiment, "internal-history")
	staleSuccess := corroboratingFacts["no-stale-success"][0]
	staleSuccess.Identifier = "internal-history/stale-success"
	staleSuccess.Source.Sequence = 2
	staleSuccess.Source.Reference = "internal-history/stale-success"
	ownerEpoch, currentOwnerEpoch, cancellationCommitted := int64(1), int64(2), true
	staleSuccess.Window = nil
	staleSuccess.History = &observation.HistoryEvent{
		EventType: observation.NexusSuccessRecorded, EventID: 2, OperationID: "operation",
		OwnerEpoch: &ownerEpoch, CurrentOwnerEpoch: &currentOwnerEpoch,
		CancellationCommitted: &cancellationCommitted,
	}
	corroboratingFacts["no-stale-success"] = append(
		corroboratingFacts["no-stale-success"], staleSuccess)
	corroborating := make(map[string][][]observation.Fact, len(corroboratingFacts))
	for identifier, facts := range corroboratingFacts {
		corroborating[identifier] = [][]observation.Fact{facts}
	}
	session := &corroboratingFactFakeSession{
		fakeSession: conformingSession(experiment), facts: primaryFacts, corroborating: corroborating,
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "contradict")
}

func TestRunFailsClosedWhenAdvertisedCorroborationIsUnavailable(t *testing.T) {
	experiment := loadExperiment(t)
	session := &corroboratingFactFakeSession{
		fakeSession: conformingSession(experiment),
		facts:       cancellationFacts(experiment, "public-history"),
		err:         errors.New("history service unavailable"),
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "history service unavailable")
	require.NotEmpty(t, result.Omissions)
}

func TestRunCleansUpAfterActionFailure(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	session.realizeErr = map[string]error{"retry-task": errors.New("injected failure")}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.True(t, session.cleaned)
}

func TestRunMissingEvidenceIsInconclusive(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	delete(session.observations, "cancellation-won")
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.NotEmpty(t, result.Omissions)
}

func TestRunOptionalOmissionRequiredByPropertyIsInconclusive(t *testing.T) {
	experiment := loadExperiment(t)
	for index := range experiment.Checkpoints {
		if experiment.Checkpoints[index].Identifier == "cancellation-won" {
			experiment.Checkpoints[index].OmissionPolicy = "optional"
		}
	}
	session := conformingSession(experiment)
	delete(session.observations, "cancellation-won")
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
}

func TestRunContradictingEvidenceIsViolating(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	observation := session.observations["no-stale-success"]
	observation.Satisfied = false
	session.observations["no-stale-success"] = observation
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimViolating, result.Claim.Kind)
}

func TestRunMissingRequiredPositiveEvidenceIsInconclusive(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	observation := session.observations["cancellation-accepted"]
	observation.Satisfied = false
	session.observations["cancellation-accepted"] = observation
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
}

func TestRunObservesCompilerCheckpointsAfterActions(t *testing.T) {
	experiment := loadExperiment(t)
	for index := range experiment.Actions {
		experiment.Actions[index].PreCheckpoint = ""
		experiment.Actions[index].PostCheckpoint = ""
	}
	session := conformingSession(experiment)
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.Len(t, result.Observations, len(experiment.Checkpoints))
}

func TestRunGroundsAndReusesCompilerIdentity(t *testing.T) {
	experiment := compiledUpdateExperiment(t)
	session := conformingSession(experiment)
	session.groundings = map[string]map[string]string{"start": {"run-id": "concrete-run-id"}}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.Equal(t, "concrete-run-id", result.Bindings["run-id"])
}

func TestRunRejectsMissingProjectionGrounding(t *testing.T) {
	experiment := compiledUpdateExperiment(t)
	session := conformingSession(experiment)
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "did not ground")
}

func TestRunHonorsCooperativeCancellation(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	factory := &blockingFactory{capabilities: allCapabilities(experiment), session: session}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := Run(ctx, Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
}

func TestRunCleansPartiallyPreparedEnvironment(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	factory := &fakeFactory{
		capabilities: allCapabilities(experiment),
		session:      session,
		prepareErr:   errors.New("prepare failed after allocation"),
	}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.True(t, session.cleaned)
}

func TestRunCleanupFailureDowngradesConformance(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	session.cleanup = CleanupResult{Complete: false, Error: "injected cleanup failure"}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.NotEmpty(t, result.Cleanup.RecoverableResources)
}

func TestRunIncomparableOrderingIsInconclusive(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	observation := session.observations["cancellation-won"]
	observation.CausalReference = ""
	session.observations["cancellation-won"] = observation
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.Contains(t, result.Omissions[0], "causal reference")
}

func TestRunMissingIdentityLineageIsEvidenceFailure(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	observation := session.observations["cancellation-won"]
	observation.Lineage = nil
	session.observations["cancellation-won"] = observation
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.NotEmpty(t, result.Omissions)
}

func TestContradictoryEvidenceIsEvidenceFailure(t *testing.T) {
	result := Result{Claim: Claim{Kind: ClaimConforming, Property: "property"}}
	result.Observations = []Observation{
		completeObservation("first", true, "api"),
		completeObservation("second", false, "history"),
	}

	finalizeEvidenceGraph(&result, 1<<20)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "contradictory evidence")
}

func TestRunRejectsCountBudgetBeforePrepare(t *testing.T) {
	experiment := loadExperiment(t)
	factory := &fakeFactory{capabilities: allCapabilities(experiment)}

	_, err := Run(context.Background(), Request{
		Experiment:  experiment,
		Environment: factory,
		Limits:      Limits{MaxActions: 1, MaxObservations: 1},
	})
	require.ErrorContains(t, err, "count budget")
	require.Zero(t, factory.prepareCount)
}

type blockingFactory struct {
	capabilities []protocol.CapabilityID
	session      Session
}

type faultFactory struct {
	fakeFactory
	realizer umpire3fault.Realizer
}

func (f *faultFactory) FaultRealizer() umpire3fault.Realizer { return f.realizer }

type runtimeFaultRealizer struct {
	calls      []string
	cleanupErr error
}

func (r *runtimeFaultRealizer) Install(context.Context, umpire3fault.Term) (string, error) {
	r.calls = append(r.calls, "install")
	return "sensitive-runtime-handle", nil
}

func (r *runtimeFaultRealizer) Activate(context.Context, string) error {
	r.calls = append(r.calls, "activate")
	return nil
}

func (r *runtimeFaultRealizer) Release(context.Context, string) error {
	r.calls = append(r.calls, "release")
	return nil
}

func (r *runtimeFaultRealizer) Cleanup(context.Context, string) error {
	r.calls = append(r.calls, "cleanup")
	return r.cleanupErr
}

func (r *runtimeFaultRealizer) RealizationEvidence(context.Context, string) (umpire3fault.RealizationEvidence, error) {
	return umpire3fault.RealizationEvidence{
		SourceIdentity: "runtime-fault-test", Reference: "runtime-fault-test/fired",
		EntityIdentity: "runtime-fault-scope",
	}, nil
}

func (f *blockingFactory) Capabilities() []protocol.CapabilityID { return f.capabilities }
func (f *blockingFactory) Prepare(ctx context.Context, _ protocol.Experiment) (PreparedEnvironment, error) {
	<-ctx.Done()
	return PreparedEnvironment{Session: f.session, Identity: testEnvironmentIdentity(f.capabilities)}, ctx.Err()
}

func loadExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}

func allCapabilities(experiment protocol.Experiment) []protocol.CapabilityID {
	seen := make(map[protocol.CapabilityID]struct{})
	var capabilities []protocol.CapabilityID
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			identifier := protocol.CapabilityID(capability)
			if _, exists := seen[identifier]; !exists {
				seen[identifier] = struct{}{}
				capabilities = append(capabilities, identifier)
			}
		}
	}
	for _, fault := range experiment.Faults {
		for _, capability := range fault.RequiredCapabilities {
			identifier := protocol.CapabilityID(capability)
			if _, exists := seen[identifier]; !exists {
				seen[identifier] = struct{}{}
				capabilities = append(capabilities, identifier)
			}
		}
	}
	return capabilities
}

func testEnvironmentIdentity(capabilities []protocol.CapabilityID) EnvironmentIdentity {
	return EnvironmentIdentity{
		Name: "test", BuildID: "build", ConfigurationIdentity: "configuration",
		EvidenceProfile: EvidenceProfileInProcessHooks, DrivingAuthority: "test-driver",
		ObservationAuthority: "test-observer", FaultAuthority: "test-faults",
		IsolationIdentity: "namespace/queue", RetentionClass: "semantic-redacted",
		Capabilities: append([]protocol.CapabilityID(nil), capabilities...),
	}
}

func conformingSession(experiment protocol.Experiment) *fakeSession {
	observations := make(map[string]Observation, len(experiment.Checkpoints))
	for index, checkpoint := range experiment.Checkpoints {
		observations[checkpoint.Identifier] = Observation{
			CheckpointID:    checkpoint.Identifier,
			Kind:            checkpoint.Observation,
			Satisfied:       true,
			Source:          "fake",
			SourceIdentity:  "fake-source",
			ClockDomain:     "fake-sequence",
			SourceSequence:  int64(index + 1),
			Reference:       "fake/observation/" + checkpoint.Identifier,
			CausalReference: "fake-causal-chain",
			EntityIdentity:  "fake-entity",
			Lineage:         []string{"fake-namespace", "fake-entity"},
		}
	}
	return &fakeSession{
		realizeErr:    make(map[string]error),
		observations:  observations,
		cleanup:       CleanupResult{Complete: true},
		faultRealizer: &runtimeFaultRealizer{},
	}
}

func completeObservation(identifier string, satisfied bool, source string) Observation {
	return Observation{
		CheckpointID: identifier, Kind: "same-observation", Satisfied: satisfied,
		Source: source, SourceIdentity: source, ClockDomain: source + "-sequence", SourceSequence: 1,
		ObservedAtUnixNano: 1, Reference: source + "/reference", EntityIdentity: "entity",
		Lineage: []string{"namespace", "entity"},
	}
}

func observationSources(observations []Observation) []string {
	sources := make([]string, len(observations))
	for index, observation := range observations {
		sources[index] = observation.Source
	}
	return sources
}

func cancellationFacts(
	experiment protocol.Experiment,
	source string,
) map[string][]observation.Fact {
	factSource := func(sequence int64) observation.Source {
		return observation.Source{
			Identity: source, ClockDomain: source + "-sequence", Sequence: sequence,
			Reference: source + "/reference", CausalReferences: []string{source + "/cause"},
			EntityIdentity: "operation", Lineage: []string{experiment.ExperimentID, "operation"},
		}
	}
	return map[string][]observation.Fact{
		"cancellation-accepted": {{
			Identifier: source + "/accepted", Source: factSource(1),
			History: &observation.HistoryEvent{
				EventType: observation.NexusCancellationAccepted, EventID: 1, OperationID: "operation",
			},
		}},
		"cancellation-won": {{
			Identifier: source + "/committed", Source: factSource(2),
			History: &observation.HistoryEvent{
				EventType: observation.NexusCancellationCommitted, EventID: 2, OperationID: "operation",
			},
		}},
		"no-stale-success": {{
			Identifier: source + "/window", Source: factSource(3),
			Window: &observation.EvidenceWindow{
				Purpose: observation.NexusCancellationWindow, Closed: true, ThroughSequence: 3,
			},
		}},
	}
}

func compiledUpdateExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	runID := scenario.Symbol{Name: "run-id", Type: protocol.SemanticTypeIDIdentity}
	suite, err := scenario.Compile(context.Background(), scenario.Scenario{
		Identifier: "runtime-update-identity",
		Target:     protocol.TargetIDWorkflowUpdateLifecycle,
		Resources: []scenario.Resource{
			{Identifier: "workflow", Kind: protocol.EntityKindWorkflow},
			{Identifier: "update", Kind: protocol.EntityKindWorkflowUpdate},
		},
		Root: scenario.OnePath(
			scenario.Action("start", protocol.ActionKindStartUpdate),
			scenario.Bind(runID, scenario.Project("start", "update-id", protocol.SemanticTypeIDIdentity)),
			scenario.Action("dispatch", protocol.ActionKindDispatchWorkflowTask,
				scenario.WithArgument("update", runID.Value())),
			scenario.Action("accept", protocol.ActionKindAcceptUpdate),
			scenario.Action("history", protocol.ActionKindRecordUpdateHistory),
			scenario.Action("complete-task", protocol.ActionKindCompleteWorkflowTask),
			scenario.Action("complete", protocol.ActionKindCompleteUpdate),
			scenario.Require(protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory),
		),
	}, scenario.Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	return suite.Experiments[0]
}

func compiledNexusAttemptExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	suite, err := scenario.Compile(context.Background(), scenario.Scenario{
		Identifier: "runtime-nexus-attempt",
		Target:     protocol.TargetIDNexusCancellation,
		Resources: []scenario.Resource{
			{Identifier: "operation", Kind: protocol.EntityKindNexusOperation},
			{Identifier: "worker", Kind: protocol.EntityKindNexusWorker},
		},
		Root: scenario.OnePath(
			scenario.Action("schedule", protocol.ActionKindScheduleOperation),
			scenario.Action("dispatch", protocol.ActionKindDispatchTask),
			scenario.Action("cancel", protocol.ActionKindRequestCancellation),
			scenario.Action("commit", protocol.ActionKindCommitCancellation),
			scenario.Action("ownership", protocol.ActionKindAcquireOwnership),
			scenario.Action("returned", protocol.ActionKindWorkerReturnsSuccess),
			scenario.Action("persist", protocol.ActionKindPersistSuccess),
			scenario.Require(protocol.PropertyIDNexusCancellationWonExcludesSuccess),
		),
	}, scenario.Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	return suite.Experiments[0]
}
