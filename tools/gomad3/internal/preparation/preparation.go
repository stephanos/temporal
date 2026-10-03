package preparation

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

type Preparer interface {
	Prepare(context.Context, target.Spec) (target.Prepared, error)
}

type Request struct {
	Target      target.Spec
	Environment []string
	Preparer    Preparer
}

type Stage string

const (
	StageAdapters   Stage = "adapters"
	StageTarget     Stage = "target"
	StageValidation Stage = "validation"
	StageReview     Stage = "review"
	StageCleanup    Stage = "cleanup"
)

type stageError struct {
	stage Stage
	err   error
}

func (failure *stageError) Error() string { return failure.err.Error() }
func (failure *stageError) Unwrap() error { return failure.err }

func StageOf(err error) Stage {
	var failure *stageError
	if errors.As(err, &failure) {
		return failure.stage
	}
	return ""
}

func Prepare(ctx context.Context, request Request) (target.Prepared, error) {
	return prepare(ctx, request, os.RemoveAll)
}

func prepare(ctx context.Context, request Request, remove func(string) error) (prepared target.Prepared, retErr error) {
	profile := deterministicio.Default()
	spec := request.Target
	preparer := request.Preparer
	selectedAdapters := []deterministicio.BuildAdapter{}
	if preparer == nil {
		if err := os.MkdirAll(spec.PreparationRoot, 0o700); err != nil {
			return target.Prepared{}, &stageError{stage: StageAdapters, err: fmt.Errorf("create preparation root: %w", err)}
		}
		workspace, err := os.MkdirTemp(spec.PreparationRoot, ".adapter-work-")
		if err != nil {
			return target.Prepared{}, &stageError{stage: StageAdapters, err: fmt.Errorf("create deterministic I/O adapter workspace: %w", err)}
		}
		defer func() {
			if cleanupErr := remove(workspace); cleanupErr != nil {
				failure := &stageError{stage: StageCleanup, err: fmt.Errorf("remove deterministic I/O adapter workspace: %w", cleanupErr)}
				retErr = errors.Join(retErr, failure)
			}
		}()
		adapterSpec := spec
		adapterSpec.PreparationRoot = workspace
		spec, selectedAdapters, err = profile.PrepareTargetBuildAdapters(ctx, adapterSpec)
		if err != nil {
			return target.Prepared{}, &stageError{stage: StageAdapters, err: err}
		}
		spec.PreparationRoot = request.Target.PreparationRoot
		preparer = targetPreparer{}
	}
	var err error
	prepared, err = preparer.Prepare(ctx, spec)
	if err != nil {
		return target.Prepared{}, &stageError{stage: StageTarget, err: err}
	}
	prepared.Adapters = executionAdapters(selectedAdapters)
	if err := profile.ValidatePreparedTarget(spec, prepared, request.Environment); err != nil {
		return target.Prepared{}, &stageError{stage: StageValidation, err: err}
	}
	return prepared, nil
}

type targetPreparer struct{}

func (targetPreparer) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return target.Prepare(ctx, spec)
}

func executionAdapters(adapters []deterministicio.BuildAdapter) []record.TargetAdapter {
	result := make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}
