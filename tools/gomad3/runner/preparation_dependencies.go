package runner

import (
	"context"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/preparation"
	"go.temporal.io/server/tools/gomad3/target"
)

type executionDependencies struct {
	executor  executionRunner
	prepare   func(context.Context, preparation.Request) (target.Prepared, error)
	bootstrap func(deterministicio.Spec, target.Prepared, string, uint64) ([]byte, error)
}

func (dependencies executionDependencies) prepareTarget(ctx context.Context, request preparation.Request) (target.Prepared, error) {
	if dependencies.prepare != nil {
		return dependencies.prepare(ctx, request)
	}
	return preparation.Prepare(ctx, request)
}

func (dependencies executionDependencies) bootstrapFrame(profile deterministicio.Spec, prepared target.Prepared, runner string, seed uint64) ([]byte, error) {
	if dependencies.bootstrap != nil {
		return dependencies.bootstrap(profile, prepared, runner, seed)
	}
	return profile.BootstrapFrame(prepared, runner, seed)
}

func (dependencies executionDependencies) injected() bool {
	return dependencies.executor != nil || dependencies.prepare != nil || dependencies.bootstrap != nil
}
