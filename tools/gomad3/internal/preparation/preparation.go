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
	Validate    func(target.Spec, target.Prepared, []string) error
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
	return prepareWith(ctx, request, preparationServices{
		adapters: profile.PrepareTargetBuildAdapters, target: target.Prepare,
		validate: profile.ValidatePreparedTarget, remove: remove,
	})
}

type preparationServices struct {
	adapters func(context.Context, target.Spec) (target.Spec, []deterministicio.BuildAdapter, error)
	target   func(context.Context, target.Spec) (target.Prepared, error)
	validate func(target.Spec, target.Prepared, []string) error
	remove   func(string) error
}

func prepareWith(ctx context.Context, request Request, services preparationServices) (prepared target.Prepared, retErr error) {
	spec := request.Target
	preparer := request.Preparer
	if spec.Backend != "" {
		if preparer == nil || request.Validate == nil {
			return target.Prepared{}, &stageError{stage: StageTarget, err: errors.New("external backend requires preparation and validation")}
		}
		prepared, err := preparer.Prepare(ctx, spec.Clone())
		if err != nil {
			return target.Prepared{}, &stageError{stage: StageTarget, err: err}
		}
		if prepared.Backend == nil || prepared.Backend.Name != spec.Backend || prepared.Kind != spec.Kind || prepared.Source != spec.Source || len(prepared.Argv) != len(spec.Args)+1 || prepared.Argv[0] != "gomad3-target" || len(prepared.Adapters) != 0 {
			return target.Prepared{}, &stageError{stage: StageValidation, err: errors.New("external backend prepared target identity does not match its specification")}
		}
		for index, argument := range spec.Args {
			if prepared.Argv[index+1] != argument {
				return target.Prepared{}, &stageError{stage: StageValidation, err: errors.New("external backend prepared target arguments changed")}
			}
		}
		if err := prepared.ValidateBackendPayloads(); err != nil {
			return target.Prepared{}, &stageError{stage: StageValidation, err: err}
		}
		prepared = prepared.CloneBackend()
		if err := request.Validate(spec.Clone(), prepared.CloneBackend(), append([]string(nil), request.Environment...)); err != nil {
			return target.Prepared{}, &stageError{stage: StageValidation, err: err}
		}
		return prepared.CloneBackend(), nil
	}
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
			if cleanupErr := services.remove(workspace); cleanupErr != nil {
				failure := &stageError{stage: StageCleanup, err: fmt.Errorf("remove deterministic I/O adapter workspace: %w", cleanupErr)}
				retErr = errors.Join(retErr, failure)
			}
		}()
		adapterSpec := spec
		adapterSpec.PreparationRoot = workspace
		spec, selectedAdapters, err = services.adapters(ctx, adapterSpec)
		if err != nil {
			return target.Prepared{}, &stageError{stage: StageAdapters, err: err}
		}
		spec.PreparationRoot = request.Target.PreparationRoot
		preparer = targetPreparer{prepare: services.target}
	}
	var err error
	prepared, err = preparer.Prepare(ctx, spec)
	if err != nil {
		return target.Prepared{}, &stageError{stage: StageTarget, err: err}
	}
	prepared.Adapters = executionAdapters(selectedAdapters)
	if err := services.validate(spec, prepared, request.Environment); err != nil {
		return target.Prepared{}, &stageError{stage: StageValidation, err: err}
	}
	if prepared.Backend != nil || len(prepared.BackendPayloads) != 0 {
		return target.Prepared{}, &stageError{stage: StageValidation, err: errors.New("native preparation returned external backend metadata")}
	}
	return prepared, nil
}

type targetPreparer struct {
	prepare func(context.Context, target.Spec) (target.Prepared, error)
}

func (preparer targetPreparer) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return preparer.prepare(ctx, spec)
}

func executionAdapters(adapters []deterministicio.BuildAdapter) []record.TargetAdapter {
	result := make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}
