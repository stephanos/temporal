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

func integrationProvider(t *testing.T) *Provider {
	t.Helper()
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Fatal("explicit stock compiler is required for integration gate")
	}
	helper, err := filepath.Abs("../wasmhost/target/release/gomad3-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	hash := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(record.HashBytes(data))
	}
	config := wasi.Config{Profile: wasi.StockProfile, Environment: []string{"PWD=/workspace"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
	provider, err := New(Options{CompilerPath: compiler, CompilerSHA256: hash(compiler), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: hash(helper), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: t.TempDir(), Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestIntegratedIntentionalFailureObservedReplay(t *testing.T) {
	provider := integrationProvider(t)
	fixture, err := filepath.Abs("../../gomad3/internal/gomadtool/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: time.Millisecond * 100, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failures != 1 || len(result.Artifacts) != 1 {
		t.Fatalf("intentional failure campaign: %#v", result)
	}
	replay, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: result.Artifacts[0], Backend: provider})
	if err != nil || !replay.Match || !replay.ObservedRepetition || replay.Artifact.Manifest.ReplayMode != record.ReplayObserved || replay.Artifact.Manifest.IOProfile.Transcript != nil {
		t.Fatalf("observed replay: %#v %v", replay, err)
	}
	if !strings.Contains(stringMustRead(t, filepath.Join(result.Artifacts[0], "stdout")), `deterministic failure after reading "pending"`) {
		t.Fatal("did not run original intentional failure fixture")
	}
	t.Logf("campaign=%s artifact=%s record=%s failure=%s", result.CampaignPath, result.Artifacts[0], replay.Artifact.Manifest.RecordHash, replay.Artifact.Manifest.Outcome.FailureSignature)
}
func stringMustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
