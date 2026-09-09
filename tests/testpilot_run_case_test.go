//go:build test_dep && integration

package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tests/testcore"
)

// CaseBinding is everything a live Run needs that the Case itself cannot know: which physical
// namespace, task queue and Nexus endpoint its symbolic bindings resolve to, the identity the
// derived Profile carries, and whether this test owns creating the endpoint.
type CaseBinding struct {
	Identity       string
	Namespace      string
	TaskQueue      string
	NexusEndpoint  string
	CreateEndpoint bool
}

// bindCase provisions the resources the binding names, derives the Profile the Case implies, and
// prepares the Case over unchanged bytes against one Driver. It replaces the hand-written Profile
// and the manual provision-prepare sequence every live test used to repeat; MOD-12's Prepare then
// Run sequence is unchanged, only who assembles the Profile.
func bindCase(t *testing.T, env *testcore.TestEnv, source *testpilotpb.Case, binding CaseBinding) testpilotLiveCase {
	t.Helper()
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
		Identity: binding.Identity, Namespace: binding.Namespace,
		TaskQueue: binding.TaskQueue, NexusEndpoint: binding.NexusEndpoint,
	})
	require.NoError(t, err)
	resources := testpilotLiveResources{Namespace: binding.Namespace, TaskQueue: binding.TaskQueue}
	if binding.CreateEndpoint {
		resources.NexusEndpoint = binding.NexusEndpoint
	}
	return newTestpilotLiveCase(t, env, source, profile, resources, testpilotCleanupTimeout)
}

// runCase is the happy path over bindCase: load the named fixture, create every resource it names,
// run once, and fail the test on a Run error. Tests that vary the binding, run concurrently, or
// deliberately omit a resource call bindCase directly.
func runCase(t *testing.T, env *testcore.TestEnv, name string, binding CaseBinding) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	binding.CreateEndpoint = binding.NexusEndpoint != ""
	live := bindCase(t, env, loadTestpilotCase(t, name), binding)
	run, verdict, err := live.prepared.Run(env.Context(), live.driver)
	require.NoError(t, err)
	return run, verdict
}
