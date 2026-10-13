package runner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native"
	nativebuild "go.temporal.io/server/tools/gomad3/backends/native/build"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

type preparationFixtureExecutor struct {
	run func(context.Context, backend.ExecutionRequest) (backend.ExecutionResult, error)
}

func (executor preparationFixtureExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	return executor.run(ctx, request)
}

func TestPreparationDependenciesForwardRealFixtureInputs(t *testing.T) {
	preparer := newFakePreparer(t)
	var mu sync.Mutex
	var requestSeeds, executionSeeds []uint64
	var outputs []*os.File
	var prepared target.Prepared
	var description backend.Description
	var runnerBuild string
	executor := preparationFixtureExecutor{run: func(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
		if request.IO == nil || !reflect.DeepEqual(request.IO.Profile, description.IOProfile) || !reflect.DeepEqual(request.Target, prepared) || request.RunnerBuild != runnerBuild {
			t.Errorf("forwarded execution inputs = %#v", request)
		}
		if request.Command != prepared.Path || request.Argv0 != prepared.Argv[0] || !slices.Equal(request.Args, prepared.Argv[1:]) {
			t.Errorf("forwarded target = %#v", request)
		}
		data, err := os.ReadFile(request.Command)
		if err != nil || string(data) != "fake prepared target" {
			t.Errorf("copied target bytes = %q, %v", data, err)
		}
		info, err := os.Stat(request.Command)
		if err != nil {
			return backend.ExecutionResult{}, err
		}
		if info.Mode().Perm() != 0o500 || uint64(info.Size()) != prepared.Size {
			t.Errorf("copied target mode/size = %v/%d", info.Mode(), info.Size())
		}
		if err := verifyPrepared(prepared); err != nil {
			return backend.ExecutionResult{}, err
		}
		var runOutputs []*os.File
		for _, writer := range []io.Writer{request.StdoutHead, request.StderrHead} {
			output, ok := writer.(*os.File)
			if !ok {
				t.Errorf("execution output = %T; want the real partial file", writer)
				continue
			}
			if _, err := output.Stat(); err != nil {
				return backend.ExecutionResult{}, err
			}
			runOutputs = append(runOutputs, output)
		}
		mu.Lock()
		requestSeeds = append(requestSeeds, request.Seed)
		executionSeeds = append(executionSeeds, seedFromEnvironment(request.Env))
		outputs = append(outputs, runOutputs...)
		mu.Unlock()
		return processResult(0, "", ""), nil
	}}
	config, _ := testConfig(t, preparer, executor, "7-9", PolicyAll, 2)
	runnerBuild = config.RunnerBuild
	var err error
	description, err = config.Backend.Describe(t.Context(), backend.Configuration{Target: config.Target})
	if err != nil {
		t.Fatal(err)
	}
	var campaignPath string
	config.Progress = func(event CampaignEvent) error {
		if event.Phase == ProgressPreparing {
			campaignPath = event.CampaignPath
		}
		return nil
	}
	dependencies := scriptedPreparationDependencies(t, preparer, executor)
	prepareFixture := dependencies.prepare
	dependencies.prepare = func(ctx context.Context, request preparationRequest) (target.Prepared, error) {
		want := config.Target
		want.PreparationRoot = filepath.Join(campaignPath, ".prepared")
		if !reflect.DeepEqual(request.Target, want) || !preparationEnvironmentIsEffective(request.Environment, config.Environment) || request.Preparer != preparer || request.Provider != config.Backend {
			t.Fatalf("preparation inputs = %#v; want %#v", request, want)
		}
		if ctx.Err() != nil {
			t.Fatal(ctx.Err())
		}
		var err error
		prepared, err = prepareFixture(ctx, request)
		if err != nil {
			return prepared, err
		}
		wantPrepared := preparer.prepared
		wantPrepared.Path = filepath.Join(want.PreparationRoot, "target")
		wantPrepared.Adapters = []record.TargetAdapter{}
		if !reflect.DeepEqual(prepared, wantPrepared) {
			t.Fatalf("prepared fixture = %#v; want %#v", prepared, wantPrepared)
		}
		return prepared, nil
	}
	result, err := exploreWith(context.Background(), config, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	slices.Sort(requestSeeds)
	slices.Sort(executionSeeds)
	if preparer.calls != 1 || result.Attempted != 3 || result.Succeeded != 3 || !slices.Equal(requestSeeds, []uint64{7, 8, 9}) || !slices.Equal(executionSeeds, requestSeeds) {
		t.Fatalf("calls=%d result=%#v request=%v execution=%v", preparer.calls, result, requestSeeds, executionSeeds)
	}
	for _, output := range outputs {
		if _, err := output.Stat(); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("execution output remained open: %v", err)
		}
	}
}

func TestPreparationDependenciesOperationErrorsRemainUnchanged(t *testing.T) {
	primary := errors.New("injected operation failed")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := preparationRequest{Target: target.Spec{Kind: target.KindGoRun, Source: "fixture", PreparationRoot: t.TempDir()}, Environment: []string{"MODE=fixture"}, Provider: scriptProvider()}
	prepared := target.Prepared{Path: "fixture result", SHA256: "fixture digest", Size: 17}
	dependencies := executionDependencies{
		prepare: func(received context.Context, receivedRequest preparationRequest) (target.Prepared, error) {
			if received != ctx || !reflect.DeepEqual(receivedRequest, request) {
				t.Fatalf("prepare arguments = %v, %#v", received, receivedRequest)
			}
			return prepared, primary
		},
	}
	if result, err := dependencies.prepareTarget(ctx, request); !reflect.DeepEqual(result, prepared) || err != primary {
		t.Fatalf("prepare result = %#v, %v", result, err)
	}
}

func TestPreparationDependenciesFailuresStopAtOriginalStages(t *testing.T) {
	for _, operation := range []string{"prepare", "execution"} {
		t.Run(operation, func(t *testing.T) {
			primary := errors.New("injected " + operation + " failed")
			preparer := newFakePreparer(t)
			executor := &fakeExecutor{}
			config, _ := testConfig(t, preparer, executor, "7", PolicyAll, 1)
			dependencies := scriptedPreparationDependencies(t, preparer, executor)
			executionCalls := 0
			if operation == "prepare" {
				dependencies.prepare = func(context.Context, preparationRequest) (target.Prepared, error) { return target.Prepared{}, primary }
			}
			dependencies.executor = preparationFixtureExecutor{run: func(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
				executionCalls++
				prepared := request.Target
				partial := filepath.Join(filepath.Dir(filepath.Dir(prepared.Path)), ".partial", "00000000000000000000-7")
				for _, name := range []string{"stdout.head", "stderr.head"} {
					if _, err := os.Stat(filepath.Join(partial, name)); err != nil {
						t.Error(err)
					}
				}
				return backend.ExecutionResult{}, primary
			}}
			result, err := exploreWith(context.Background(), config, dependencies)
			if !errors.Is(err, primary) || len(executor.directories()) != 0 || result.CampaignPath == "" {
				t.Fatalf("failed operation result = %#v, %v, executions=%v", result, err, executor.directories())
			}
			if operation == "prepare" {
				if err != primary || preparer.calls != 0 || executionCalls != 0 || result.Attempted != 0 {
					t.Fatalf("preparation failure changed: %v calls=%d executions=%d result=%#v", err, preparer.calls, executionCalls, result)
				}
			} else {
				var hostError *HostError
				if !errors.As(err, &hostError) || hostError.Reason != "target_supervision" || preparer.calls != 1 || executionCalls != 1 || result.Attempted != 1 {
					t.Fatalf("execution failure changed: %v calls=%d executions=%d result=%#v", err, preparer.calls, executionCalls, result)
				}
			}
		})
	}
}

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

// Preparation validates against the effective recorded environment: every
// requested entry plus the Runner-owned I/O profile.
func preparationEnvironmentIsEffective(effective, requested []string) bool {
	for _, entry := range requested {
		if !slices.Contains(effective, entry) {
			return false
		}
	}
	return slices.ContainsFunc(effective, func(entry string) bool { return strings.HasPrefix(entry, "GOMAD3_IO_PROFILE=") })
}

func scriptedPreparationDependencies(t *testing.T, preparer Preparer, executor executionRunner) executionDependencies {
	t.Helper()
	if preparer == nil || executor == nil {
		t.Fatal("scripted preparation requires an explicit preparer and executor")
	}
	return executionDependencies{
		executor: executor,
		prepare: func(ctx context.Context, request preparationRequest) (target.Prepared, error) {
			if !reflect.DeepEqual(request.Preparer, preparer) || request.Target.PreparationRoot == "" || len(request.Target.AdapterReplacements) != 0 {
				t.Fatalf("scripted preparation request = %#v", request)
			}
			prepared, err := preparer.Prepare(ctx, request.Target)
			if err != nil {
				return target.Prepared{}, err
			}
			if prepared.Kind != request.Target.Kind || prepared.Source != request.Target.Source || len(prepared.Argv) == 0 || prepared.Argv[0] != "gomad3-target" || !slices.Equal(prepared.Argv[1:], request.Target.Args) || len(prepared.Adapters) != 0 {
				t.Fatalf("scripted prepared target = %#v", prepared)
			}
			if err := verifyPrepared(prepared); err != nil {
				t.Fatal(err)
			}
			prepared.Adapters = []record.TargetAdapter{}
			return prepared, nil
		},
	}
}

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
