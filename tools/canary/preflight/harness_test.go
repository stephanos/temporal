//go:build canary_harness

package preflight

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
)

func TestHarnessBindingUsesTheFunctionalCaseAndLeavesProductionPinned(t *testing.T) {
	fixture, profile, canary := harnessFixture(t)
	production := configured(t)
	pinned := casebinding.Case()
	before, err := casebinding.Bind(production, testCoordinates.Driver())
	require.NoError(t, err)
	reads := registered()
	scope, err := CheckHarness(t.Context(), input(canary, dispatch(canary), testCoordinates, reads), fixture.Bytes, profile)
	require.NoError(t, err)
	require.Equal(t, 1, reads.reads)
	protorequire.ProtoEqual(t, fixture.Source, scope.Prepared.Snapshot())
	functional, err := testpilot.Prepare(fixture.Source, profile)
	require.NoError(t, err)
	require.Equal(t, functional.Identity(), scope.Prepared.Identity())
	fixture.Bytes[0] = '!'
	profile.EnvironmentBindings[0].Value = "mutated"
	protorequire.ProtoEqual(t, fixture.Source, scope.Prepared.Snapshot())
	require.Equal(t, functional.Identity(), scope.Prepared.Identity())
	after, err := casebinding.Bind(production, testCoordinates.Driver())
	require.NoError(t, err)
	require.Equal(t, before.Prepared.Identity(), after.Prepared.Identity())
	require.Equal(t, pinned, casebinding.Case())
	rejected, err := Check(t.Context(), input(canary, dispatch(canary), testCoordinates, reads))
	require.Error(t, err)
	require.Nil(t, rejected)
	require.Equal(t, 1, reads.reads)
}

func TestHarnessRejectsMissingCapabilityBeforeIO(t *testing.T) {
	fixture, profile, canary := harnessFixture(t)
	profile.Opcodes = slices.DeleteFunc(profile.Opcodes, func(op testpilot.Opcode) bool { return op == testpilot.Finish })
	reads := registered()
	scope, err := CheckHarness(t.Context(), input(canary, dispatch(canary), testCoordinates, reads), fixture.Bytes, profile)
	require.Nil(t, scope)
	require.ErrorContains(t, err, "unsupported at activity.complete-attempt")
	refusal, ok := AsRefusal(err)
	require.True(t, ok)
	require.Equal(t, StatusCaseMismatch, refusal.Status)
	require.Zero(t, reads.reads)
}

// The canary reaches a server it does not run, so its environment supplies no delivery control: the
// Case that holds a delivery is refused at preparation, naming the actuator, before the first read.
// The same Case under the same coordinates is admitted once the environment supplies the control,
// so the refusal is the capability's and nothing else's.
func TestHarnessRejectsTheHoldDeliveryActuatorBeforeIO(t *testing.T) {
	fixture, err := generatedHarnessCase(t, "activity-race-heldAdmission.staleDelivery-case.json")
	require.NoError(t, err)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	canary := configured(t)
	canary.AuthorityClass, canary.EvaluationProfile = policy.AuthorityHarness, "canary-harness"
	canary.CaseIdentity, err = recordedrun.CaseIdentity(fixture.Bytes)
	require.NoError(t, err)
	environment := testCoordinates.Driver()
	environment.Identity = canary.CaseProfile
	require.False(t, environment.DeliveryControl)
	profile, err := temporal.DeriveProfile(fixture.Source, catalog, environment)
	require.NoError(t, err)

	reads := registered()
	scope, err := CheckHarness(t.Context(), input(canary, dispatch(canary), testCoordinates, reads), fixture.Bytes, profile)
	require.Nil(t, scope)
	require.ErrorContains(t, err, "unsupported at controller.hold-dispatch")
	refusal, ok := AsRefusal(err)
	require.True(t, ok)
	require.Equal(t, StatusCaseMismatch, refusal.Status)
	require.Zero(t, reads.reads)

	environment.DeliveryControl = true
	capable, err := temporal.DeriveProfile(fixture.Source, catalog, environment)
	require.NoError(t, err)
	_, err = testpilot.Prepare(fixture.Source, capable)
	require.NoError(t, err)
}

func TestHarnessBindingRejectsMismatchedAuthorityAndResources(t *testing.T) {
	for _, mismatch := range []string{"authority", "evaluation", "identity", "profile", "namespace", "workflow", "coordinates"} {
		t.Run(mismatch, func(t *testing.T) {
			fixture, profile, canary := harnessFixture(t)
			reads := registered()
			in := input(canary, dispatch(canary), testCoordinates, reads)
			switch mismatch {
			case "authority":
				canary.AuthorityClass = policy.AuthorityProtectedWorkflow
			case "evaluation":
				canary.EvaluationProfile = "production-canary"
			case "identity":
				canary.CaseIdentity = "another"
			case "profile":
				profile.Identity = "another"
			case "namespace":
				profile.EnvironmentBindings[0].Value = "another"
			case "workflow":
				in.Lookup = lookupOf(map[string]string{})
			case "coordinates":
				in.Coordinates.TaskQueue = "another"
			default:
				require.FailNow(t, "unknown mismatch", mismatch)
			}
			scope, err := CheckHarness(t.Context(), in, fixture.Bytes, profile)
			require.Error(t, err)
			require.Nil(t, scope)
			require.Zero(t, reads.reads)
		})
	}
}

func harnessFixture(t *testing.T) (*testpilotcore.ModelCase, testpilot.ProfileSpec, *policy.Policy) {
	t.Helper()
	fixture, err := generatedHarnessCase(t, "activity-completion-case.json")
	require.NoError(t, err)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	canary := configured(t)
	canary.AuthorityClass, canary.EvaluationProfile = policy.AuthorityHarness, "canary-harness"
	canary.CaseIdentity, err = recordedrun.CaseIdentity(fixture.Bytes)
	require.NoError(t, err)
	environment := testCoordinates.Driver()
	environment.Identity = canary.CaseProfile
	profile, err := temporal.DeriveProfile(fixture.Source, catalog, environment)
	require.NoError(t, err)
	return fixture, profile, canary
}

func generatedHarnessCase(t *testing.T, file string) (*testpilotcore.ModelCase, error) {
	t.Helper()
	directory := filepath.Join("..", "..", "..", "model", "cases")
	entries, err := testpilotcore.GeneratedCases(directory)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.File == file {
			return testpilotcore.LoadGeneratedCase(directory, entry)
		}
	}
	return nil, fmt.Errorf("no generated Case %s", file)
}
