package preparation

import (
	"context"
	"errors"
	"fmt"
	"os"

	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

type Inspection struct {
	Spec          target.Spec
	Review        target.CapabilityReview
	BuildAdapters []transcript.BuildAdapter
	Adapters      []transcript.Adapter
	root          string
	remove        func(string) error
}

func Inspect(ctx context.Context, spec target.Spec) (Inspection, error) {
	return inspect(ctx, spec, os.RemoveAll)
}

func inspect(ctx context.Context, spec target.Spec, remove func(string) error) (_ Inspection, retErr error) {
	return inspectWith(ctx, spec, inspectionServices{
		adapters: transcript.Default().PrepareTargetBuildAdapters,
		review:   nativebuild.ReviewCapabilities, remove: remove,
	})
}

type inspectionServices struct {
	adapters func(context.Context, target.Spec) (target.Spec, []transcript.BuildAdapter, error)
	review   func(context.Context, target.Spec) (target.CapabilityReview, error)
	remove   func(string) error
}

func inspectWith(ctx context.Context, spec target.Spec, services inspectionServices) (_ Inspection, retErr error) {
	if spec.PreparationRoot != "" {
		return Inspection{}, errors.New("capability review preparation root must be owned by the review")
	}
	root, err := os.MkdirTemp("", "gomad-compatibility-review-")
	if err != nil {
		return Inspection{}, &stageError{stage: StageAdapters, err: fmt.Errorf("create capability review preparation directory: %w", err)}
	}
	keep := false
	defer func() {
		if !keep {
			cleanupErr := removeInspectionRoot(root, services.remove)
			if retErr != nil && cleanupErr != nil {
				retErr = &inspectionFailure{primary: retErr, cleanup: cleanupErr}
			} else {
				retErr = errors.Join(retErr, cleanupErr)
			}
		}
	}()
	if err := os.Chmod(root, 0o700); err != nil {
		return Inspection{}, &stageError{stage: StageAdapters, err: fmt.Errorf("make capability review preparation directory private: %w", err)}
	}
	spec.PreparationRoot = root
	preparedSpec, adapters, err := services.adapters(ctx, spec)
	if err != nil {
		return Inspection{}, &stageError{stage: StageAdapters, err: err}
	}
	review, err := services.review(ctx, preparedSpec)
	if err != nil {
		return Inspection{}, &stageError{stage: StageReview, err: err}
	}
	keep = true
	return Inspection{
		Spec: preparedSpec, Review: review, BuildAdapters: adapters,
		Adapters: transcript.SelectedAdapters(adapters), root: root, remove: services.remove,
	}, nil
}

func (inspected *Inspection) Close() error {
	if inspected == nil || inspected.root == "" {
		return nil
	}
	root := inspected.root
	inspected.root = ""
	return removeInspectionRoot(root, inspected.remove)
}

func removeInspectionRoot(root string, remove func(string) error) error {
	if err := remove(root); err != nil {
		return &stageError{stage: StageCleanup, err: err}
	}
	return nil
}

type inspectionFailure struct {
	primary error
	cleanup error
}

func (failure *inspectionFailure) Error() string {
	return errors.Join(failure.primary, failure.cleanup).Error()
}

func (failure *inspectionFailure) Unwrap() []error {
	return []error{failure.primary, failure.cleanup}
}

// InspectionErrorParts separates an inspection's primary failure from its
// owned-workspace cleanup failure without projecting through other wrappers.
func InspectionErrorParts(err error) (primary, cleanup error) {
	if failure, ok := err.(*inspectionFailure); ok {
		return failure.primary, failure.cleanup
	}
	return err, nil
}
