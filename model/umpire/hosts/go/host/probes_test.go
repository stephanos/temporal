package host_test

import (
	"context"
	"testing"

	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	model "umpire.local/model/host"
)

func TestDispatchProbeRejectsMessageDescriptor(t *testing.T) {
	const messageName = "umpire.model.v1.ExploreRequest"
	response, err := model.DispatchProbe(context.Background(), nil, &annotationsv1.ProbeResponse{
		Method: "/" + messageName + "/Call",
	}, messageName)
	if err == nil || err.Error() != "unsupported probe service" || response != nil {
		t.Fatalf("response=%v error=%v", response, err)
	}
}
