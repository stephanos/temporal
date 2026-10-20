package wasi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStressHostConditionsStayOutsideGuestInputs(t *testing.T) {
	t.Parallel()
	request := Request{Engine: PinnedEngine(), Config: Config{Environment: []string{"DECLARED=unchanged"}}}
	before, err := json.Marshal(request.Config)
	if err != nil {
		t.Fatal(err)
	}
	request.SetStressHostConditions(strings.Repeat("x", 4096), true)
	after, err := json.Marshal(request.Config)
	if err != nil || string(before) != string(after) {
		t.Fatalf("host perturbation changed declared inputs: %v", err)
	}
	if len(request.stressHostEnvironment) != 2 || request.DiagnosticConfigurationRevision() != PinnedEngine().Configuration+"-stress-jitter" {
		t.Fatal("stress helper not admitted")
	}
	request.SetStressHostConditions("", false)
	if len(request.stressHostEnvironment) != 1 {
		t.Fatal("jitter switch leaked into reference")
	}
}

func TestStressJitterHelperRefusedForQualification(t *testing.T) {
	t.Parallel()
	request := Request{Engine: PinnedEngine()}
	request.Engine.Configuration += "-stress-jitter"
	_, err := Run(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "requires test opt-in") {
		t.Fatalf("qualification admitted jitter helper: %v", err)
	}
	if _, err := prepareClusterBoot("guest", request); err == nil {
		t.Fatal("cluster qualification admitted jitter helper")
	}
}
