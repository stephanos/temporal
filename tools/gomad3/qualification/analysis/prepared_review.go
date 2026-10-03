package analysis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"

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
	preparedSpec, adapters, err := deterministicio.Default().PrepareTargetBuildAdapters(ctx, spec)
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

// ReviewCompatibilityTarget prepares spec's adapters, reviews its target, and
// releases the preparation, as compatibility-pack discovery does.
func ReviewCompatibilityTarget(ctx context.Context, spec target.Spec) (target.CapabilityReview, error) {
	prepared, err := PrepareCapabilityReview(ctx, spec)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	return prepared.Review, prepared.Close()
}

// HostDeterministicProfile returns the identity of the deterministic I/O
// profile adapter bindings carry on this host, or ok false when the host has
// no deterministic I/O boundary.
func HostDeterministicProfile() (name, implementationSHA256 string, ok bool) {
	host := runtime.GOOS + "/" + runtime.GOARCH
	if !slices.Contains(deterministicio.BoundaryPlatforms(), host) {
		return "", "", false
	}
	profile := deterministicio.Default().Identity()
	return profile.Name, string(profile.ImplementationSHA256), true
}
