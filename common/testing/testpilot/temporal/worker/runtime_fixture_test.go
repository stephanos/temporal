package worker

import (
	"testing"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
)

func preparedRuntimeFixture(t *testing.T, reply replyKind, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	return preparedRuntimeFixtureWithProfile(t, reply, nil, modify...)
}

func preparedRuntimeFixtureForNamespace(t *testing.T, namespace string, reply replyKind, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	return preparedRuntimeFixtureWithProfile(t, reply, func(profile *testpilot.ProfileSpec) {
		for index := range profile.EnvironmentBindings {
			if profile.EnvironmentBindings[index].ID == "namespace" {
				profile.EnvironmentBindings[index].Value = namespace
			}
		}
	}, modify...)
}

func preparedRuntimeFixtureWithProfile(t *testing.T, reply replyKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	t.Helper()
	return facadetest.Capture(t, preparedRuntimeCase(t, reply, modifyProfile, modify...))
}

// preparedRuntimeCase is the shared runtime fixture under the command types this Driver realizes.
func preparedRuntimeCase(t *testing.T, reply replyKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) *testpilot.PreparedCase {
	t.Helper()
	fixtureReply := facadetest.SyncReply
	if reply == replyAsynchronous {
		fixtureReply = facadetest.AsyncReply
	}
	return facadetest.RuntimeCase(t, fixtureReply, CommandTypes(), modifyProfile, modify...)
}
