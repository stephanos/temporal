package model_test

import (
	"context"
	"testing"

	model "umpire.local/model"
	annotationsv1 "umpire.local/model/umpire/annotations/v1"
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
