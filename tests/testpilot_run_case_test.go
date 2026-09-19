//go:build test_dep && integration

package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tests/testcore"
)

// caseBinding is everything a live Run needs that the Case itself cannot know: which physical
// namespace, task queue and Nexus endpoint its symbolic bindings resolve to, the identity the
// derived Profile carries, and whether this test owns creating the endpoint.
type CaseBinding struct {
	Identity       string
	Namespace      string
	TaskQueue      string
	NexusEndpoint  string
	CreateEndpoint bool
	// CleanupTimeout bounds every resource this binding creates. Zero takes the shared default;
	// a Case with more worker entrypoints to stop names a longer one.
	CleanupTimeout time.Duration
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
	cleanupTimeout := binding.CleanupTimeout
	if cleanupTimeout == 0 {
		cleanupTimeout = testpilotCleanupTimeout
	}
	return newTestpilotLiveCase(t, env, source, profile, resources, cleanupTimeout)
}

// defaultBinding is the binding a Case implies when the caller has nothing to say about it: one
// namespace, queue and endpoint named after the fixture. A Case that binds no Nexus endpoint gets
// none, because creating one it never references would authorize a resource the Case does not name.
func defaultBinding(name string, source *testpilotpb.Case) CaseBinding {
	binding := CaseBinding{
		Identity:  "umpire-" + name + "-profile",
		Namespace: "umpire-" + name,
		TaskQueue: "umpire-" + name + "-queue",
	}
	if bindsNexusEndpoint(source) {
		binding.NexusEndpoint = "umpire-" + name + "-endpoint"
	}
	return binding
}

// bindsNexusEndpoint reports whether the Case declares an endpoint role with its own resource
// binding, which is the only thing a Nexus endpoint name resolves.
func bindsNexusEndpoint(source *testpilotpb.Case) bool {
	for _, role := range source.GetProgram().GetRoles() {
		if role.GetKind() == testpilotpb.ROLE_KIND_ENDPOINT && role.GetResourceBindingId() != "" {
			return true
		}
	}
	return false
}

// runCase is the happy path: load the named fixture, create the resources its own name implies,
// run once, and fail the test on a Run error. A new live test is this call plus its Verdict
// assertions. Tests that vary the binding, run concurrently, or deliberately omit a resource call
// runCaseWithBinding or bindCase directly.
func runCase(t *testing.T, env *testcore.TestEnv, name string) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	source := loadTestpilotCase(t, name)
	return runBoundCase(t, env, source, defaultBinding(name, source))
}

// runCaseWithBinding is runCase over a binding the caller chose.
func runCaseWithBinding(t *testing.T, env *testcore.TestEnv, name string, binding CaseBinding) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	return runBoundCase(t, env, loadTestpilotCase(t, name), binding)
}

func runBoundCase(t *testing.T, env *testcore.TestEnv, source *testpilotpb.Case, binding CaseBinding) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	binding.CreateEndpoint = binding.NexusEndpoint != ""
	live := bindCase(t, env, source, binding)
	run, verdict, err := live.prepared.Run(env.Context(), live.driver)
	require.NoError(t, err)
	return run, verdict
}
