package analysis

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

type PreparedCapabilityReview struct {
	Spec          target.Spec
	Review        target.CapabilityReview
	BuildAdapters []deterministicio.BuildAdapter
	Adapters      []deterministicio.Adapter
	root          string
}

func PrepareCapabilityReview(ctx context.Context, spec target.Spec) (_ PreparedCapabilityReview, retErr error) {
	if spec.PreparationRoot != "" {
		return PreparedCapabilityReview{}, errors.New("capability review preparation root must be owned by the review")
	}
	root, err := os.MkdirTemp("", "gomad3-compatibility-review-")
	if err != nil {
		return PreparedCapabilityReview{}, fmt.Errorf("create capability review preparation directory: %w", err)
	}
	keep := false
	defer func() {
		if !keep {
			retErr = errors.Join(retErr, os.RemoveAll(root))
		}
	}()
	if err := os.Chmod(root, 0o700); err != nil {
		return PreparedCapabilityReview{}, fmt.Errorf("make capability review preparation directory private: %w", err)
	}
	spec.PreparationRoot = root
	moduleCache, err := target.ReadModuleCache(ctx, spec.ToolchainRoot)
	if err != nil {
		return PreparedCapabilityReview{}, err
	}
	preparedSpec, adapters, err := deterministicio.Default().PrepareBuildAdapters(spec, moduleCache)
	if err != nil {
		return PreparedCapabilityReview{}, err
	}
	review, err := target.ReviewCapabilities(ctx, preparedSpec)
	if err != nil {
		return PreparedCapabilityReview{}, err
	}
	keep = true
	return PreparedCapabilityReview{
		Spec: preparedSpec, Review: review, BuildAdapters: adapters,
		Adapters: deterministicio.SelectedAdapters(adapters), root: root,
	}, nil
}

func (prepared *PreparedCapabilityReview) Close() error {
	if prepared == nil || prepared.root == "" {
		return nil
	}
	root := prepared.root
	prepared.root = ""
	return os.RemoveAll(root)
}
