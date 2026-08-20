package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
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

func validOptions() Options {
	return Options{
		Address: "localhost:7233", Namespace: "namespace", TaskQueue: "queue",
		BuildID: "build", Profile: "remote-deployment", Timeout: time.Minute,
	}
}
