package command

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	environment "go.temporal.io/server/tests/umpire3/execution"
	umpire3runner "go.temporal.io/server/tests/umpire3/temporal"
)

func TestExecutionProfileIsExplicitAndFailClosed(t *testing.T) {
	t.Setenv("UMPIRE3_TEMPORAL_API_KEY", "token")
	configuration := runCompatibilityOptions{
		experimentPath: "experiment.json", outputPath: "result.json", address: "localhost:7233",
		namespace: "namespace", taskQueue: "queue", buildID: "build", timeout: time.Minute,
		profile: "grpc-only-black-box",
	}
	profile, err := umpire3runner.Validate(runCompatibilityRunnerOptions(configuration))
	require.NoError(t, err)
	require.Equal(t, environment.EvidenceProfilePublicGRPC, profile.Environment.EvidenceProfile)
	require.Equal(t, "none", profile.Environment.FaultAuthority)

	configuration.profile = "production-canary"
	_, err = umpire3runner.Validate(runCompatibilityRunnerOptions(configuration))
	require.EqualError(t, err, `unsupported execution profile "production-canary"`)
}

func TestExecutionProfileRequiresCompleteNexusConfiguration(t *testing.T) {
	t.Setenv("UMPIRE3_TEMPORAL_API_KEY", "token")
	configuration := runCompatibilityOptions{
		experimentPath: "experiment.json", outputPath: "result.json", address: "localhost:7233",
		namespace: "namespace", taskQueue: "queue", buildID: "build", timeout: time.Minute,
		profile: "remote-deployment", nexusEndpoint: "endpoint",
	}
	_, err := umpire3runner.Validate(runCompatibilityRunnerOptions(configuration))
	require.EqualError(t, err, "nexus endpoint, service, and operation must be supplied together")
}
