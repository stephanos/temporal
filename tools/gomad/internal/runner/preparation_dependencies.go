package runner

import (
	"context"
	"errors"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

func verifyPrepared(prepared target.Prepared) error {
	return prepared.Verify()
}

type preparationRequest struct {
	Target      target.Spec
	Environment []string
	Preparer    Preparer
	Provider    backend.Provider
}

type executionDependencies struct {
	executor executionRunner
	prepare  func(context.Context, preparationRequest) (target.Prepared, error)
}

func (dependencies executionDependencies) prepareTarget(ctx context.Context, request preparationRequest) (target.Prepared, error) {
	if dependencies.prepare != nil {
		return dependencies.prepare(ctx, request)
	}
	if request.Provider == nil {
		return target.Prepared{}, errors.New("backend provider is required")
	}
	preparer := request.Preparer
	if preparer == nil {
		preparer = request.Provider
	}
	prepared, err := preparer.Prepare(ctx, request.Target.Clone())
	if err != nil {
		var failure *backend.PreparationError
		if errors.As(err, &failure) {
			return target.Prepared{}, err
		}
		return target.Prepared{}, &backend.PreparationError{Stage: "target", Err: err}
	}
	if err := request.Provider.ValidatePrepared(request.Target.Clone(), prepared, append([]string(nil), request.Environment...)); err != nil {
		return target.Prepared{}, &backend.PreparationError{Stage: "validation", Err: err}
	}
	return prepared.CloneBackend(), nil
}

func (dependencies executionDependencies) injected() bool {
	return dependencies.executor != nil || dependencies.prepare != nil
}
