package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestValidateIsExplicitAndFailClosed(t *testing.T) {
	options := validOptions()
	options.Profile = "grpc-only-black-box"
	profile, err := Validate(options)
	require.NoError(t, err)
	require.Equal(t, environment.EvidenceProfilePublicGRPC, profile.EvidenceProfile)
	require.Equal(t, "none", profile.FaultAuthority)

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

	_, err := executePrepared(context.Background(), experiment, validOptions(), factory, worker)
	require.ErrorContains(t, err, "start SDK worker: injected start failure")
	require.Equal(t, 1, factory.prepareCount)
	require.Equal(t, 1, session.cleanupCount)
	require.Zero(t, worker.stopCount)
}

func validOptions() Options {
	return Options{
		Address: "localhost:7233", Namespace: "namespace", TaskQueue: "queue",
		BuildID: "build", Profile: "remote-deployment", Timeout: time.Minute,
	}
}

type lifecycleFactory struct {
	session      environment.Session
	prepareCount int
}

func (*lifecycleFactory) Capabilities() []string { return nil }

func (f *lifecycleFactory) Prepare(context.Context, protocol.Experiment) (environment.Session, error) {
	f.prepareCount++
	return f.session, nil
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
