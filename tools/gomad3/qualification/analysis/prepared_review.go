package analysis

import (
	"context"
	"runtime"
	"slices"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/preparation"
	"go.temporal.io/server/tools/gomad3/target"
)

type PreparedCapabilityReview struct {
	Spec          target.Spec
	Review        target.CapabilityReview
	BuildAdapters []deterministicio.BuildAdapter
	Adapters      []deterministicio.Adapter
	inspection    preparation.Inspection
}

func PrepareCapabilityReview(ctx context.Context, spec target.Spec) (PreparedCapabilityReview, error) {
	inspected, err := preparation.Inspect(ctx, spec)
	if err != nil {
		return PreparedCapabilityReview{}, err
	}
	return PreparedCapabilityReview{
		Spec: inspected.Spec, Review: inspected.Review, BuildAdapters: inspected.BuildAdapters,
		Adapters: inspected.Adapters, inspection: inspected,
	}, nil
}

func (prepared *PreparedCapabilityReview) Close() error {
	if prepared == nil {
		return nil
	}
	return prepared.inspection.Close()
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
