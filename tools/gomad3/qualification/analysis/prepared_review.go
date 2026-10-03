package analysis

import (
	"context"

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
