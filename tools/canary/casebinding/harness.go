//go:build canary_harness

package casebinding

import (
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

func BindHarness(canonical []byte, canary *policy.Policy, environment testpilotdriver.Environment, profile testpilot.ProfileSpec) (*Bound, error) {
	if canary == nil || canary.AuthorityClass != policy.AuthorityHarness || canary.EvaluationProfile != "canary-harness" {
		return nil, errors.New("an explicit Case requires a canary-harness policy and harness authority")
	}
	identity, err := recordedrun.CaseIdentity(canonical)
	if err != nil {
		return nil, err
	}
	if identity != canary.CaseIdentity {
		return nil, errors.New("the harness Case identity differs from the policy")
	}
	if profile.Identity != canary.CaseProfile {
		return nil, errors.New("the harness Profile identity differs from the policy")
	}
	source, err := testpilot.DecodeCaseProtoJSON(canonical)
	if err != nil {
		return nil, err
	}
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	if err != nil {
		return nil, err
	}
	environment.Identity = canary.CaseProfile
	expected, err := testpilotdriver.DeriveProfile(source, catalog, environment)
	if err != nil {
		return nil, err
	}
	frozen := profile.Snapshot()
	if frozen.Catalog == nil || frozen.Catalog.Identity() != catalog.Identity() {
		return nil, errors.New("the harness Profile requires the tree's catalog")
	}
	bindings := slices.Clone(frozen.EnvironmentBindings)
	wanted := slices.Clone(expected.EnvironmentBindings)
	byID := func(a, b testpilot.EnvironmentBinding) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	}
	slices.SortFunc(bindings, byID)
	slices.SortFunc(wanted, byID)
	if !slices.Equal(bindings, wanted) {
		return nil, errors.New("the harness Profile bindings differ from the policy's coordinates")
	}
	prepared, err := testpilot.Prepare(source, frozen)
	if err != nil {
		return nil, fmt.Errorf("prepare the harness Case: %w", err)
	}
	return &Bound{Source: source, CaseIdentity: identity, Profile: frozen, Prepared: prepared}, nil
}
