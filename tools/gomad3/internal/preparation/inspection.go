package preparation

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

type Inspection struct {
	Spec          target.Spec
	Review        target.CapabilityReview
	BuildAdapters []deterministicio.BuildAdapter
	Adapters      []deterministicio.Adapter
	root          string
	remove        func(string) error
}

func Inspect(ctx context.Context, spec target.Spec) (Inspection, error) {
	return inspect(ctx, spec, os.RemoveAll)
}

func inspect(ctx context.Context, spec target.Spec, remove func(string) error) (_ Inspection, retErr error) {
	if spec.PreparationRoot != "" {
		return Inspection{}, errors.New("capability review preparation root must be owned by the review")
	}
	root, err := os.MkdirTemp("", "gomad3-compatibility-review-")
	if err != nil {
		return Inspection{}, &stageError{stage: StageAdapters, err: fmt.Errorf("create capability review preparation directory: %w", err)}
	}
	keep := false
	defer func() {
		if !keep {
			retErr = errors.Join(retErr, removeInspectionRoot(root, remove))
		}
	}()
	if err := os.Chmod(root, 0o700); err != nil {
		return Inspection{}, &stageError{stage: StageAdapters, err: fmt.Errorf("make capability review preparation directory private: %w", err)}
	}
	spec.PreparationRoot = root
	preparedSpec, adapters, err := deterministicio.Default().PrepareTargetBuildAdapters(ctx, spec)
	if err != nil {
		return Inspection{}, &stageError{stage: StageAdapters, err: err}
	}
	review, err := target.ReviewCapabilities(ctx, preparedSpec)
	if err != nil {
		return Inspection{}, &stageError{stage: StageReview, err: err}
	}
	keep = true
	return Inspection{
		Spec: preparedSpec, Review: review, BuildAdapters: adapters,
		Adapters: deterministicio.SelectedAdapters(adapters), root: root, remove: remove,
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
		return &stageError{stage: StageCleanup, err: fmt.Errorf("remove capability review preparation directory: %w", err)}
	}
	return nil
}
