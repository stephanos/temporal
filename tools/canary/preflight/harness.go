//go:build canary_harness

package preflight

import (
	"context"

	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
)

func CheckHarness(ctx context.Context, input Input, canonical []byte, profile testpilot.ProfileSpec) (*Scope, error) {
	return check(ctx, input, func(canary *policy.Policy, environment testpilotdriver.Environment) (*casebinding.Bound, error) {
		return casebinding.BindHarness(canonical, canary, environment, profile)
	})
}
