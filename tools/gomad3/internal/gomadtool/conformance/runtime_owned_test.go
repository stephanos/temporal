package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func TestRuntimeOwnedControlProbe(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if retained := os.Getenv("GOMAD3_RUNTIME_OWNED_DIR"); retained != "" {
		workspace = retained
		if err := os.MkdirAll(workspace, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	report := Report{Mode: "runtime-owned-control"}
	campaign := runtimeCampaign{ctx: context.Background(), config: Config{Root: root, Go: filepath.Join(root, ".toolchain", "bin", "go")}, testdata: filepath.Join(root, "internal", "gomadtool", "conformance", "testdata"), workspace: workspace, run: hostexec.Run, report: &report}
	fixture, err := campaign.build("runtime-owned", "./runtime_owned", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := campaign.requireRuntimeOwned(fixture); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeOwnedRejectsPreviousController(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal", "gomadtool", "conformance", "testdata", "runtime_owned", "previous-controller.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var plan choice.ReplayPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if _, err := choice.ValidateReplayPlan(plan, plan.Identity); err != nil {
		t.Fatalf("baseline tape is invalid: %v", err)
	}
	identity := plan.Identity
	identity.ImplementationSHA256, err = choice.ImplementationIdentity(identity.ToolchainBuildKey)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ImplementationSHA256 == plan.Identity.ImplementationSHA256 {
		t.Fatal("scheduler change did not change controller identity independently of the build key")
	}
	if _, err := choice.ValidateReplayPlan(plan, identity); !errors.Is(err, choice.ErrInvalidReplayPlan) {
		t.Fatalf("preceding controller tape was not rejected: %v", err)
	}
}
