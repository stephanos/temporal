package runner

import (
	"errors"
	"testing"

	"go.temporal.io/server/tools/gomadv3/runner/internal/campaign"
)

func TestIsCapacityErrorFindsTypedCapacityThroughHostFailure(t *testing.T) {
	err := &HostError{Reason: "artifact_publication", Err: &campaign.ArtifactCapacityError{
		Limit: campaign.ArtifactLimitTotalBytes, Required: 2, Maximum: 1, Outcome: campaign.CapacityInfrastructureFailure,
	}}
	if !IsCapacityError(err) {
		t.Fatalf("IsCapacityError(%v) = false", err)
	}
	if IsCapacityError(errors.New("capacity")) {
		t.Fatal("IsCapacityError() classified an untyped message")
	}
}
