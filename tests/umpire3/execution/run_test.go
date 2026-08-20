package execution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
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

func (f *footprintFactory) FootprintReport() (umpire3fault.Report, error) {
	return f.report, nil
}

type corroboratingFakeSession struct {
	*fakeSession
	observations map[string][]Observation
	err          error
}

func (s *corroboratingFakeSession) Corroborate(
	_ context.Context,
	checkpoint protocol.Checkpoint,
	_ Bindings,
) ([]Observation, error) {
	return s.observations[checkpoint.Identifier], s.err
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
		Source: "fake", Reference: action.Identifier, GroundedBindings: s.groundings[action.Identifier],
	}, s.realizeErr[action.Kind]
}

func (s *fakeSession) Observe(_ context.Context, checkpoint protocol.Checkpoint, _ Bindings) (Observation, error) {
	observation, ok := s.observations[checkpoint.Identifier]
	if !ok {
		return Observation{}, ErrObservationUnavailable
	}
	return observation, nil
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
	session := conformingSession(experiment)
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
		claim       ClaimKind
		resultClass protocol.ResultClass
	}{
		{claim: ClaimConforming, resultClass: protocol.ResultClassImplementationConforming},
		{claim: ClaimViolating, resultClass: protocol.ResultClassTraceWitness},
		{claim: ClaimUnsupported, resultClass: protocol.ResultClassUnknown},
		{claim: ClaimInconclusive, resultClass: protocol.ResultClassUnknown},
		{claim: ClaimEvidenceFailure, resultClass: protocol.ResultClassUnknown},
	} {
		t.Run(string(test.claim), func(t *testing.T) {
			result := Result{Claim: Claim{Kind: test.claim}}
			finalizeAssurance(&result)
			require.Equal(t, test.resultClass, result.ResultClass)
			require.Equal(t, protocol.TrustBadgeTestedInstance, result.TrustBadge)
			require.NoError(t, result.ValidateAssurance())

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

func TestRunReportsAllowedFailureAsDegradedWithoutChangingClaim(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	last := experiment.Actions[len(experiment.Actions)-1]
	session.actionEvidence = map[string]ActionEvidence{
		last.Identifier: {
			Source: "fake", SourceIdentity: "fake", Reference: "history/failed",
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
}

func TestRunRetainsIndependentCorroboratingEvidence(t *testing.T) {
	experiment := loadExperiment(t)
	primary := conformingSession(experiment)
	corroborating := make(map[string][]Observation, len(experiment.Checkpoints))
	for identifier, observation := range primary.observations {
		observation.Source = "history-service"
		observation.SourceIdentity = "history-service-cluster"
		observation.ClockDomain = "history-service-event-id"
		observation.Reference = "history-service/" + identifier
		corroborating[identifier] = []Observation{observation}
	}
	session := &corroboratingFakeSession{fakeSession: primary, observations: corroborating}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.Len(t, result.Observations, 2*len(experiment.Checkpoints))
	require.Contains(t, observationSources(result.Observations), "fake")
	require.Contains(t, observationSources(result.Observations), "history-service")
}

func TestRunRejectsDisagreeingCorroboratingEvidence(t *testing.T) {
	experiment := loadExperiment(t)
	primary := conformingSession(experiment)
	observation := primary.observations[experiment.Checkpoints[0].Identifier]
	observation.Satisfied = !observation.Satisfied
	observation.Source = "history-service"
	observation.SourceIdentity = "history-service-cluster"
	observation.Reference = "history-service/disagreement"
	session := &corroboratingFakeSession{
		fakeSession: primary,
		observations: map[string][]Observation{
			experiment.Checkpoints[0].Identifier: {observation},
		},
	}
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimEvidenceFailure, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "contradictory evidence")
}

func TestRunFailsClosedWhenAdvertisedCorroborationIsUnavailable(t *testing.T) {
	experiment := loadExperiment(t)
	primary := conformingSession(experiment)
	session := &corroboratingFakeSession{fakeSession: primary, err: errors.New("history service unavailable")}
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

func TestRunUsesGeneratedPropertyProgramInsteadOfEveryObservationBoolean(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	observation := session.observations["cancellation-accepted"]
	observation.Satisfied = false
	session.observations["cancellation-accepted"] = observation
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
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

func TestRunMissingIdentityLineageIsInconclusive(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	observation := session.observations["cancellation-won"]
	observation.Lineage = nil
	session.observations["cancellation-won"] = observation
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimInconclusive, result.Claim.Kind)
	require.Contains(t, result.Omissions[0], "lineage")
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
