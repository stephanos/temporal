package runner

import (
	"errors"
	"testing"

	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
)

func TestIsCapacityErrorFindsTypedCapacityThroughHostFailure(t *testing.T) {
	err := &HostError{Reason: "artifact_publication", Err: &campaignstore.ArtifactCapacityError{
		Limit: campaignstore.ArtifactLimitTotalBytes, Required: 2, Maximum: 1, Outcome: campaignstore.CapacityInfrastructureFailure,
	}}
	if !IsCapacityError(err) {
		t.Fatalf("IsCapacityError(%v) = false", err)
	}
	if IsCapacityError(errors.New("capacity")) {
		t.Fatal("IsCapacityError() classified an untyped message")
	}
}
