package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

func TestIntegratedCooperativeFailureArtifactDetachedReplay(t *testing.T) {
	stock := integrationProvider(t)
	options := stock.options
	root := t.TempDir()
	options.CacheRoot = filepath.Join(root, "cache")
	compilerRoot := filepath.Join(root, "go")
	distribution, err := command(t.Context(), options.CompilerPath, []string{"env", "GOROOT"}, root, buildEnvironment(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(compilerRoot, os.DirFS(strings.TrimSpace(string(distribution)))); err != nil {
		t.Fatal(err)
	}
	options.CompilerPath = filepath.Join(compilerRoot, "bin/go")
	options.Config.Profile = wasi.CooperativeProfile
	options.RuntimeRoot = filepath.Join(root, "runtime-owner")
	if err := os.MkdirAll(options.RuntimeRoot, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gomad.go", "gomad_choicewire_generated.go"} {
		data, err := os.ReadFile(filepath.Join("../../gomad3/toolchain/runtime/overlay/src/runtime", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(options.RuntimeRoot, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("../testdata/runtime_probes/main.go")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"go.mod": []byte("module gomad.wasm.cooperative.fixture\n\ngo 1.27.1\n"), "main.go": fixture} {
		if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, BuildTags: []string{"test_dep"}, PreparationRoot: filepath.Join(root, "prepared-first")}
	first, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	spec.PreparationRoot = filepath.Join(root, "prepared-second")
	second, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildKey != second.BuildKey || first.SHA256 != second.SHA256 {
		t.Fatal("unchanged cooperative preparation did not reuse its complete build identity")
	}
	spec.PreparationRoot = ""
	const seeds = "0,1000003,2000006,3000009,4000012,5000015,6000018,7000021,8000024,9000027,10000030,11000033,12000036,13000039,14000042,15000045,16000048,17000051,18000054,19000057,20000060,21000063,22000066,23000069,24000072,25000075,26000078,27000081,28000084,29000087,30000090,31000093"
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: seeds, Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 10 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, ChoiceTraceLimit: 1 << 20, Diagnostics: true, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "task5-cooperative", Coverage: runner.CoverageNone, Target: spec})
	if err != nil || campaign.Failures != 1 || len(campaign.Artifacts) != 1 {
		t.Fatalf("cooperative canary discovery: %+v err=%v", campaign, err)
	}
	for _, directory := range []string{source, compilerRoot, options.RuntimeRoot, options.CacheRoot} {
		if err := os.Rename(directory, directory+"-unavailable"); err != nil {
			t.Fatal(err)
		}
	}
	replay, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: campaign.Artifacts[0], Backend: provider})
	if err != nil || !replay.Match || replay.ObservedRepetition || replay.ChoiceReplayStatus != runner.ChoiceReplayExact || replay.Artifact.Manifest.ReplayMode != record.ReplayExact || replay.Artifact.Manifest.Outcome.ExitCode == nil || *replay.Artifact.Manifest.Outcome.ExitCode != 17 {
		t.Fatalf("detached cooperative artifact replay: %+v err=%v", replay, err)
	}
	if replay.Artifact.Manifest.ChoiceProfile == nil || replay.Artifact.Manifest.ChoiceProfile.Trace.TapeSHA256 == "" || replay.Artifact.Manifest.IOProfile.Transcript != nil {
		t.Fatal("cooperative artifact lacks its exact choice tape or borrowed native I/O identity")
	}
	t.Logf("campaign=%s artifact=%s record=%s module=%s build=%s choice=%s replay=%s", campaign.CampaignPath, campaign.Artifacts[0], replay.Artifact.Manifest.RecordHash, first.SHA256, first.BuildKey, replay.Artifact.Manifest.ChoiceProfile.Trace.TapeSHA256, replay.ChoiceReplayStatus)
}
