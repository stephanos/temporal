//go:build canary_harness

package testharness

import (
	"context"

	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/canary/preflight"
)

func PrepareCase(ctx context.Context, input preflight.Input, canonical []byte, profile testpilot.ProfileSpec) (*preflight.Scope, error) {
	return preflight.CheckHarness(ctx, input, canonical, profile)
}
