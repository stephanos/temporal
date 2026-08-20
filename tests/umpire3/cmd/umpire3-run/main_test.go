package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	umpire3runner "go.temporal.io/server/tests/umpire3/runner"
)

func TestExecutionProfileIsExplicitAndFailClosed(t *testing.T) {
	configuration := config{
		experimentPath: "experiment.json", outputPath: "result.json", address: "localhost:7233",
		namespace: "namespace", taskQueue: "queue", buildID: "build", timeout: time.Minute,
		profile: "grpc-only-black-box",
	}
	profile, err := umpire3runner.Validate(runnerOptions(configuration))
	require.NoError(t, err)
	require.Equal(t, environment.EvidenceProfilePublicGRPC, profile.EvidenceProfile)
	require.Equal(t, "none", profile.FaultAuthority)

	configuration.profile = "production-canary"
	_, err = umpire3runner.Validate(runnerOptions(configuration))
	require.EqualError(t, err, `unsupported execution profile "production-canary"`)
}

func TestExecutionProfileRequiresCompleteNexusConfiguration(t *testing.T) {
	configuration := config{
		experimentPath: "experiment.json", outputPath: "result.json", address: "localhost:7233",
		namespace: "namespace", taskQueue: "queue", buildID: "build", timeout: time.Minute,
		profile: "remote-deployment", nexusEndpoint: "endpoint",
	}
	_, err := umpire3runner.Validate(runnerOptions(configuration))
	require.EqualError(t, err, "nexus endpoint, service, and operation must be supplied together")
}
