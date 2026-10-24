// Package temporaldriver composes reference executors with the transport-only Driver.
package temporaldriver

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	rawworker "umpire.local/temporal/reference/worker/raw"
	sdkworker "umpire.local/temporal/reference/worker/sdk"
)

// NewFactory binds the configured executor to the Driver's owned connection.
func NewFactory(opts act.Options) (*act.Factory, error) {
	if opts.ExecutorKind == sdkworker.Kind {
		opts.ExecutorVersion = sdkworker.Version
		opts.ValidateProgram = sdkworker.Validate
		opts.Follow = sdkworker.Follow
		opts.ContextPropagators = []workflow.ContextPropagator{sdkworker.ScenarioPropagator()}
	} else {
		opts.ExecutorVersion = rawworker.Version
		opts.ValidateProgram = rawworker.Validate
		opts.Follow = func(ctx context.Context, conn client.Client, program []byte) error {
			return rawworker.Follow(ctx, conn.WorkflowService(), program)
		}
	}
	return act.NewFactory(opts)
}
