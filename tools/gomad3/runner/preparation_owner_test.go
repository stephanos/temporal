package runner

import (
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestCreateCampaignPlanPreparesAdapterAfterBundleRootExists(t *testing.T) {
	config, dependencies := testConfig(t, nil, &fakeExecutor{}, "1", PolicyAll, 1)
	module, err := filepath.Abs(filepath.Join("..", "deterministicio", "testdata", "sprig"))
	if err != nil {
		t.Fatal(err)
	}
	config.Target = target.Spec{
		Kind: target.KindGoTest, Source: ".", WorkingDir: module,
		ToolchainRoot: toolchainRoot(t),
	}
	_, _, err = deterministicio.Default().PrepareTargetBuildAdapters(t.Context(), config.Target)
	if err == nil || !strings.Contains(err.Error(), "preparation root") {
		t.Fatalf("pre-bundle adapter preparation error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := createCampaignPlanWith(t.Context(), CampaignPlanSpec{Campaign: config, Output: path}, dependencies); err != nil {
		t.Fatal(err)
	}
	opened, err := openCampaignPlan(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.prepared.Adapters) != 1 || opened.prepared.Adapters[0].Module != "github.com/Masterminds/sprig/v3" {
		t.Fatalf("plan adapters = %#v", opened.prepared.Adapters)
	}
}
