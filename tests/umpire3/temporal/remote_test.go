package temporal

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	environment "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestValidateIsExplicitAndFailClosed(t *testing.T) {
	options := validOptions()
	options.Profile = "grpc-only-black-box"
	profile, err := Validate(options)
	require.NoError(t, err)
	require.Equal(t, environment.EvidenceProfilePublicGRPC, profile.Environment.EvidenceProfile)
	require.Equal(t, "none", profile.Environment.FaultAuthority)

	options.Profile = "production-canary"
	_, err = Validate(options)
	require.EqualError(t, err, `unsupported execution profile "production-canary"`)
}

func TestValidateRequiresCompleteNexusConfiguration(t *testing.T) {
	options := validOptions()
	options.NexusEndpoint = "endpoint"
	_, err := Validate(options)
	require.EqualError(t, err, "nexus endpoint, service, and operation must be supplied together")
}

func TestWorkerStartFailureCleansPreparedSessionExactlyOnce(t *testing.T) {
	experiment := runnerExperiment(t)
	session := &lifecycleSession{}
	factory := &lifecycleFactory{session: session}
	worker := &failingWorker{startErr: errors.New("injected start failure")}

	owned, err := OwnWorkerLifecycle(factory, worker, time.Minute)
	require.NoError(t, err)
	_, err = owned.Prepare(context.Background(), experiment)
	require.ErrorContains(t, err, "start SDK worker: injected start failure")
	require.Equal(t, 1, factory.prepareCount)
	require.Equal(t, 1, session.cleanupCount)
	require.Zero(t, worker.stopCount)
}

func TestWorkerLifecycleStopsAndCleansUpExactlyOnce(t *testing.T) {
	experiment := runnerExperiment(t)
	session := &lifecycleSession{}
	factory := &lifecycleFactory{session: session}
	worker := &failingWorker{}

	owned, err := OwnWorkerLifecycle(factory, worker, time.Minute)
	require.NoError(t, err)
	prepared, err := owned.Prepare(context.Background(), experiment)
	require.NoError(t, err)
	require.Equal(t, 1, factory.prepareCount)
	require.Equal(t, environment.CleanupResult{Complete: true}, prepared.Session.Cleanup(context.Background()))
	require.Equal(t, environment.CleanupResult{Complete: true}, prepared.Session.Cleanup(context.Background()))
	require.Equal(t, 1, session.cleanupCount)
	require.Equal(t, 1, worker.stopCount)
}

func TestPrepareFailureCleansPartialSessionExactlyOnce(t *testing.T) {
	experiment := runnerExperiment(t)
	session := &lifecycleSession{}
	factory := &lifecycleFactory{session: session, prepareErr: errors.New("injected prepare failure")}
	worker := &failingWorker{}

	owned, err := OwnWorkerLifecycle(factory, worker, time.Minute)
	require.NoError(t, err)
	_, err = owned.Prepare(context.Background(), experiment)
	require.ErrorContains(t, err, "injected prepare failure")
	require.Equal(t, 1, session.cleanupCount)
	require.Zero(t, worker.stopCount)
}

func validOptions() Options {
	return Options{
		Address: "localhost:7233", Namespace: "namespace", TaskQueue: "queue",
		BuildID: "build", Profile: "remote-deployment", APIKey: "token", Timeout: time.Minute,
	}
}

type lifecycleFactory struct {
	session      environment.Session
	prepareErr   error
	prepareCount int
}

func (*lifecycleFactory) Capabilities() []protocol.CapabilityID {
	return []protocol.CapabilityID{protocol.CapabilityIDHistoryObservation}
}

func (f *lifecycleFactory) Prepare(
	context.Context,
	protocol.Experiment,
) (environment.PreparedEnvironment, error) {
	f.prepareCount++
	return environment.PreparedEnvironment{
		Session: f.session,
		Identity: environment.EnvironmentIdentity{
			Name: "test", BuildID: "build", ConfigurationIdentity: "configuration",
			EvidenceProfile: environment.EvidenceProfilePublicGRPC, DrivingAuthority: "driver",
			ObservationAuthority: "observer", FaultAuthority: "none",
			IsolationIdentity: "namespace/queue", RetentionClass: "semantic-redacted",
			Capabilities: f.Capabilities(),
		},
	}, f.prepareErr
}

type lifecycleSession struct {
	cleanupCount int
}

func (*lifecycleSession) Realize(
	context.Context,
	protocol.Action,
	environment.Bindings,
) (environment.ActionEvidence, error) {
	return environment.ActionEvidence{}, nil
}

func (*lifecycleSession) Observe(
	context.Context,
	protocol.Checkpoint,
	environment.Bindings,
) (environment.Observation, error) {
	return environment.Observation{}, nil
}

func (s *lifecycleSession) Cleanup(context.Context) environment.CleanupResult {
	s.cleanupCount++
	return environment.CleanupResult{Complete: true}
}

func (*lifecycleSession) RecoveryMetadata() map[string]string { return nil }

type failingWorker struct {
	startErr  error
	stopCount int
}

func (w *failingWorker) Start() error { return w.startErr }

func (w *failingWorker) Stop() { w.stopCount++ }

func runnerExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
