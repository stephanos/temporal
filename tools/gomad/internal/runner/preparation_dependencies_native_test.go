//go:build gomad_native

package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/native"
	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
)

func TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard(t *testing.T) {
	profile := transcript.Default()
	_, guard := profile.BootstrapFrame(target.Prepared{}, "invalid", 0)
	if guard == nil || !strings.HasPrefix(guard.Error(), "deterministic I/O requires one of ") {
		t.Skip("unsupported-host refusal control requires an unqualified host")
	}
	for _, mode := range []string{"public", "executor only", "prepare only"} {
		t.Run(mode, func(t *testing.T) {
			preparer := newFakePreparer(t)
			executor := &fakeExecutor{}
			config, dependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
			config.Backend = native.New(native.Config{SupervisorCommand: []string{"unused"}, RunnerBuild: config.RunnerBuild})
			var err error
			if mode == "public" {
				_, err = Explore(context.Background(), config)
			} else {
				if mode == "prepare only" {
					dependencies = scriptedPreparationDependencies(t, preparer, executor)
					dependencies.executor = nil
				}
				_, err = exploreWith(context.Background(), config, dependencies)
			}
			if err == nil || !strings.Contains(err.Error(), guard.Error()) || preparer.calls != 1 || len(executor.directories()) != 0 {
				t.Fatalf("%s default = %v calls=%d executions=%v", mode, err, preparer.calls, executor.directories())
			}
			if mode == "prepare only" {
				var hostError *HostError
				if !errors.As(err, &hostError) || hostError.Reason != "target_supervision" {
					t.Fatalf("real bootstrap guard = %v", err)
				}
			} else {
				var preparationError *backend.PreparationError
				if !errors.As(err, &preparationError) || preparationError.Stage != "validation" {
					t.Fatalf("real preparation stage = %v", err)
				}
			}
		})
	}
}

func TestPreparationSourceCallerPreservation(t *testing.T) {
	root := os.Getenv("GOMAD_CALLER_INPUTS")
	if root == "" {
		t.Skip("matched original/current caller source controls requested separately")
	}
	stop := errors.New("stop after preparation")
	var snapshots []preparationCallerSnapshot
	for _, fixture := range []string{"simple", "adapter"} {
		for _, caller := range []string{"explore", "plan"} {
			t.Run(fixture+"/"+caller, func(t *testing.T) {
				toolchain := filepath.Join(root, fixture, caller, "toolchain")
				identity, err := nativebuild.ReadToolchainIdentity(toolchain)
				if err != nil {
					t.Fatal(err)
				}
				countFile := filepath.Join(toolchain, "build-count")
				buildRoot := filepath.Join(toolchain, "builds", identity.BuildKey)
				if _, err := os.Lstat(filepath.Join(buildRoot, "prepared-targets")); !os.IsNotExist(err) {
					t.Fatalf("initial prepared cache is not empty: %v", err)
				}
				var first target.Prepared
				for index, state := range []string{"fresh", "cache"} {
					preparationRoot := filepath.Join(root, fixture, caller, state)
					if err := os.Mkdir(preparationRoot, 0o700); err != nil {
						t.Fatal(err)
					}
					config := CampaignSpec{
						Seeds: "1", Parallel: 1, ExecutionTimeout: time.Second, OverallTimeout: time.Minute, TerminateGrace: 100 * time.Millisecond,
						OnFailure: PolicyAll, FailureBudget: 1, OutputLimit: 64, WorldTransitionLimit: 64, Environment: []string{"MODE=test"},
						RunnerBuild: "sha256:" + strings.Repeat("0", 64), Backend: native.New(native.Config{ToolchainRoot: toolchain, SupervisorCommand: []string{"unused"}}), Artifacts: filepath.Join(preparationRoot, "artifacts"),
						Target: target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: filepath.Join(root, fixture, "module"), PreparationRoot: preparationRoot, ToolchainRoot: toolchain},
					}
					var prepared target.Prepared
					if caller == "explore" {
						config.Progress = func(event CampaignEvent) error {
							if event.Phase == ProgressRunning {
								return stop
							}
							return nil
						}
						result, err := Explore(t.Context(), config)
						var hostError *HostError
						if !errors.Is(err, stop) || !errors.As(err, &hostError) || hostError.Reason != "progress_output" || result.Attempted != 0 {
							t.Fatalf("Explore() did not stop after validated preparation: %#v, %v", result, err)
						}
						plan, err := campaign.ReadResumePlan(result.CampaignPath)
						if err != nil {
							t.Fatal(err)
						}
						prepared = sourcePreparedFromPlan(plan)
						prepared.Path = filepath.Join(result.CampaignPath, filepath.FromSlash(plan.Prepared.Path))
					} else {
						path := filepath.Join(preparationRoot, "campaign.plan.json")
						if _, err := CreateCampaignPlan(t.Context(), CampaignPlanSpec{Campaign: config, Output: path}); err != nil {
							t.Fatal(err)
						}
						opened, err := campaign.OpenPortablePlan(path)
						if err != nil {
							t.Fatal(err)
						}
						prepared = opened.Prepared
					}
					if err := verifyPrepared(prepared); err != nil {
						t.Fatal(err)
					}
					count, err := os.ReadFile(countFile)
					if err != nil || strings.Count(string(count), "build\n") != 1 {
						t.Fatalf("%s build count = %q, %v, want exactly one build", state, count, err)
					}
					wantAdapters := 0
					if fixture == "adapter" {
						wantAdapters = 1
					}
					if len(prepared.Adapters) != wantAdapters {
						t.Fatalf("adapters = %#v", prepared.Adapters)
					}
					prepared.Path = "<prepared-path>"
					if index == 0 {
						first = prepared
					} else if !reflect.DeepEqual(prepared, first) || !reflect.DeepEqual(prepared.RecordTarget(), first.RecordTarget()) {
						t.Fatalf("fresh/cache caller targets differ: %#v %#v", first, prepared)
					}
					snapshots = append(snapshots, preparationCallerSnapshot{Fixture: fixture, Caller: caller, Cache: state, BuildCount: 1, Prepared: prepared, Target: prepared.RecordTarget()})
				}
				lock, err := hostfs.Try(filepath.Join(buildRoot, "target-cache", "gomad-cache.lock"))
				if err != nil {
					t.Fatal(err)
				}
				if err := lock.Release(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	encoded, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("GOMAD_CALLER_SNAPSHOT"), append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
