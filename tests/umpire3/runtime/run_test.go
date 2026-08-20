//nolint:revive // The package name is the public Umpire3 runtime.Run seam.
package runtime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type fakeFactory struct {
	capabilities []string
	session      *fakeSession
	prepareErr   error
	prepareCount int
}

func (f *fakeFactory) Capabilities() []string {
	return f.capabilities
}

func (f *fakeFactory) Prepare(context.Context, protocol.Experiment) (environment.Session, error) {
	f.prepareCount++
	return f.session, f.prepareErr
}

type fakeSession struct {
	mu           sync.Mutex
	realizeErr   map[string]error
	observations map[string]environment.Observation
	cleanup      environment.CleanupResult
	realized     []string
	cleaned      bool
}

func (s *fakeSession) Realize(ctx context.Context, action protocol.Action, _ environment.Bindings) (environment.ActionEvidence, error) {
	select {
	case <-ctx.Done():
		return environment.ActionEvidence{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realized = append(s.realized, action.Kind)
	return environment.ActionEvidence{Source: "fake", Reference: action.Identifier}, s.realizeErr[action.Kind]
}

func (s *fakeSession) Observe(_ context.Context, checkpoint protocol.Checkpoint, _ environment.Bindings) (environment.Observation, error) {
	observation, ok := s.observations[checkpoint.Identifier]
	if !ok {
		return environment.Observation{}, environment.ErrObservationUnavailable
	}
	return observation, nil
}

func (s *fakeSession) Cleanup(context.Context) environment.CleanupResult {
	s.cleaned = true
	return s.cleanup
}

func (s *fakeSession) RecoveryMetadata() map[string]string {
	return map[string]string{"resource": "fake"}
}

func TestRunRejectsUnsupportedCapabilitiesBeforePrepare(t *testing.T) {
	experiment := loadExperiment(t)
	factory := &fakeFactory{capabilities: []string{"nexus"}}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimUnsupported, result.Claim.Kind)
	require.Zero(t, factory.prepareCount)
}

func TestRunConformsWithCompleteCausalEvidenceAndCleanup(t *testing.T) {
	experiment := loadExperiment(t)
	session := conformingSession(experiment)
	factory := &fakeFactory{capabilities: allCapabilities(experiment), session: session}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, ClaimConforming, result.Claim.Kind)
	require.True(t, session.cleaned)
	require.Len(t, result.Actions, len(experiment.Actions))
	require.Len(t, result.Observations, len(experiment.Checkpoints))
	require.True(t, result.Cleanup.Complete)
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

func TestRunAllowsExplicitlyOptionalOmission(t *testing.T) {
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
	require.Equal(t, ClaimConforming, result.Claim.Kind)
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
	session.cleanup = environment.CleanupResult{Complete: false, Error: "injected cleanup failure"}
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
	capabilities []string
	session      environment.Session
}

func (f *blockingFactory) Capabilities() []string { return f.capabilities }
func (f *blockingFactory) Prepare(ctx context.Context, _ protocol.Experiment) (environment.Session, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func loadExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}

func allCapabilities(experiment protocol.Experiment) []string {
	seen := make(map[string]struct{})
	var capabilities []string
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			if _, exists := seen[capability]; !exists {
				seen[capability] = struct{}{}
				capabilities = append(capabilities, capability)
			}
		}
	}
	return capabilities
}

func conformingSession(experiment protocol.Experiment) *fakeSession {
	observations := make(map[string]environment.Observation, len(experiment.Checkpoints))
	for index, checkpoint := range experiment.Checkpoints {
		observations[checkpoint.Identifier] = environment.Observation{
			CheckpointID:    checkpoint.Identifier,
			Kind:            checkpoint.Observation,
			Satisfied:       true,
			Source:          "fake",
			SourceSequence:  int64(index + 1),
			CausalReference: "fake-causal-chain",
		}
	}
	return &fakeSession{
		realizeErr:   make(map[string]error),
		observations: observations,
		cleanup:      environment.CleanupResult{Complete: true},
	}
}
