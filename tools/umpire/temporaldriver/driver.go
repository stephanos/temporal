// Package temporaldriver composes reference executors with the transport-only Driver.
package temporaldriver

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"umpire.local/temporal/reference/worker"
	rawworker "umpire.local/temporal/reference/worker/raw"
	sdkworker "umpire.local/temporal/reference/worker/sdk"
)

// NewFactory binds the configured executor to the Driver's owned connection.
func NewFactory(opts act.Options) (*act.Factory, error) {
	if opts.ExecutorKind == sdkworker.Kind {
		opts.ExecutorVersion = sdkworker.Version
		opts.ValidateProgram = sdkworker.Validate
		opts.Follow = sdkworker.Follow
	} else {
		opts.ExecutorVersion = rawworker.Version
		opts.ValidateProgram = rawworker.Validate
		opts.Follow = func(ctx context.Context, conn client.Client, program []byte) error {
			return rawworker.Follow(ctx, conn.WorkflowService(), program)
		}
	}
	validate := opts.ValidateProgram
	opts.ValidateProgram = func(data []byte) error {
		program, err := worker.Decode(data)
		if err != nil {
			return err
		}
		if program.Namespace != opts.Namespace {
			return fmt.Errorf("worker program namespace %q differs from configured namespace %q", program.Namespace, opts.Namespace)
		}
		if program.TaskQueue != opts.TaskQueue {
			return fmt.Errorf("worker program task queue %q differs from configured task queue %q", program.TaskQueue, opts.TaskQueue)
		}
		return validate(data)
	}
	return act.NewFactory(opts)
}
