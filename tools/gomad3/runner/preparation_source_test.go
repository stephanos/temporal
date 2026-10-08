package runner

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

type preparationCallerSnapshot struct {
	Fixture    string          `json:"fixture"`
	Caller     string          `json:"caller"`
	Cache      string          `json:"cache"`
	BuildCount int             `json:"build_count"`
	Prepared   target.Prepared `json:"prepared_from_validated_caller_plan"`
	Target     record.Target   `json:"record_target"`
}

func TestPreparationSourceCallerPreservation(t *testing.T) {
	root := os.Getenv("GOMAD3_CALLER_INPUTS")
	if root == "" {
		t.Skip("matched original/current caller source controls requested separately")
	}
	stop := errors.New("stop after preparation")
	var snapshots []preparationCallerSnapshot
	for _, fixture := range []string{"simple", "adapter"} {
		for _, caller := range []string{"explore", "plan"} {
			t.Run(fixture+"/"+caller, func(t *testing.T) {
				toolchain := filepath.Join(root, fixture, caller, "toolchain")
				identity, err := target.ReadToolchainIdentity(toolchain)
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
						RunnerBuild: "sha256:" + strings.Repeat("0", 64), SupervisorCommand: []string{"unused"}, Artifacts: filepath.Join(preparationRoot, "artifacts"),
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
						opened, err := openCampaignPlan(path)
						if err != nil {
							t.Fatal(err)
						}
						prepared = opened.prepared
					}
					if err := prepared.Verify(); err != nil {
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
	if err := os.WriteFile(os.Getenv("GOMAD3_CALLER_SNAPSHOT"), append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func sourcePreparedFromPlan(plan campaign.CampaignPlan) target.Prepared {
	targetRecord := plan.Prepared.Target
	return target.Prepared{
		Kind: target.Kind(targetRecord.Kind), Source: targetRecord.Source, SHA256: string(targetRecord.SHA256), Size: uint64(targetRecord.Size),
		Argv: append([]string(nil), targetRecord.Argv...), BuildTags: append([]string(nil), targetRecord.BuildTags...), Adapters: cloneAdapters(targetRecord.Adapters), Compatibility: cloneCompatibility(targetRecord.Compatibility), BuildInfo: cloneBuildInfo(targetRecord.BuildInfo),
		GoVersion: plan.Toolchain.GoVersion, BuildKey: plan.Toolchain.BuildKey, TargetGOOS: plan.Toolchain.TargetGOOS, TargetGOARCH: plan.Toolchain.TargetGOARCH,
		CapabilityMode: target.CapabilityMode(targetRecord.CapabilityMode), CapabilityManifest: target.CapabilityManifestFromRecord(targetRecord.CapabilityManifest),
	}
}
